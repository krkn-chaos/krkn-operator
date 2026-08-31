/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BackupConfig holds backup configuration
type BackupConfig struct {
	Namespace  string
	OutputDir  string
	BackupName string
}

// resourceType describes a resource to backup
type resourceType struct {
	gvk       schema.GroupVersionKind
	singular  string
	selector  client.MatchingLabels
	hasLabels []string
}

// CreateBackup creates a backup of operator configuration
func CreateBackup(ctx context.Context, k8sClient client.Client, config BackupConfig) (string, error) {
	logger := log.FromContext(ctx)

	if config.OutputDir == "" {
		config.OutputDir = "."
	}
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create temporary directory for backup files
	// Use unique parent directory with timestamp, but keep backup directory name consistent
	tempParent := filepath.Join(os.TempDir(), fmt.Sprintf("krkn-backup-tmp-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempParent, 0700); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempParent)

	tempDir := filepath.Join(tempParent, "krkn-backup")
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	logger.Info("Starting backup", "namespace", config.Namespace, "tempDir", tempDir)

	// Define resources to backup
	resources := []resourceType{
		// Custom resources
		{
			gvk:      schema.GroupVersionKind{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknUser"},
			singular: "krknuser",
		},
		{
			gvk:      schema.GroupVersionKind{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknUserGroup"},
			singular: "krknusergroup",
		},
		{
			gvk:      schema.GroupVersionKind{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknOperatorTarget"},
			singular: "krknoperatortarget",
		},
		{
			gvk:      schema.GroupVersionKind{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknOperatorTargetProvider"},
			singular: "krknoperatortargetprovider",
		},
		{
			gvk:      schema.GroupVersionKind{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknFileType"},
			singular: "krknfiletype",
		},
		// Secrets (by name and labels)
		{
			gvk:      schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			singular: "jwt-secret",
		},
		{
			gvk:      schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			singular: "auth-secrets",
			selector: map[string]string{"app.kubernetes.io/component": "authentication"},
		},
		{
			gvk:      schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			singular: "user-auth-secrets",
			selector: map[string]string{"app.kubernetes.io/component": "user-auth"},
		},
		{
			gvk:       schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			singular:  "target-secrets",
			hasLabels: []string{"krkn-target-uuid"},
		},
		{
			gvk:      schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			singular: "elasticsearch-secrets",
			selector: map[string]string{"app.kubernetes.io/component": "elasticsearch-config"},
		},
		{
			gvk:      schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			singular: "registry-secrets",
			selector: map[string]string{"app.kubernetes.io/component": "registry"},
		},
		// ConfigMaps
		{
			gvk:       schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
			singular:  "file-configmaps",
			hasLabels: []string{"files.krkn.krkn-chaos.dev/file-id"},
		},
		{
			gvk:      schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
			singular: "acm-config",
		},
	}

	// Backup each resource type
	backupCount := 0
	errCount := 0

	for _, resType := range resources {
		filename := resType.singular + ".json"
		outputPath := filepath.Join(tempDir, filename)

		if err := backupResourceType(ctx, k8sClient, config.Namespace, resType, outputPath); err != nil {
			logger.Error(err, "Failed to backup resource type", "type", resType.singular)
			errCount++
			continue
		}

		info, err := os.Stat(outputPath)
		if err == nil && info.Size() > 0 {
			backupCount++
		} else {
			os.Remove(outputPath)
		}
	}

	logger.Info("Resource backup completed", "backed_up", backupCount, "failed", errCount)

	if backupCount == 0 {
		return "", fmt.Errorf("no resources were backed up")
	}

	// Create tar.gz archive
	archiveName := config.BackupName
	if archiveName == "" {
		archiveName = fmt.Sprintf("krkn-backup-%d", time.Now().Unix())
	}
	archivePath := filepath.Join(config.OutputDir, archiveName+".tar.gz")

	// Write to a temp file first, then rename for atomic creation
	tmpArchivePath := archivePath + ".tmp"
	if err := createTarGz(tempParent, tmpArchivePath); err != nil {
		os.Remove(tmpArchivePath)
		return "", fmt.Errorf("failed to create archive: %w", err)
	}
	if err := os.Rename(tmpArchivePath, archivePath); err != nil {
		os.Remove(tmpArchivePath)
		return "", fmt.Errorf("failed to finalize archive: %w", err)
	}

	logger.Info("Backup completed successfully", "archivePath", archivePath, "backupCount", backupCount)

	return archivePath, nil
}

// backupResourceType backs up all instances of a specific resource type
func backupResourceType(ctx context.Context, k8sClient client.Client, namespace string, resType resourceType, outputPath string) error {
	logger := log.FromContext(ctx)

	// List resources based on resource type
	var items []unstructured.Unstructured

	if resType.gvk.Kind == "Secret" && resType.singular == "jwt-secret" {
		secret := &corev1.Secret{}
		if err := k8sClient.Get(ctx, client.ObjectKey{
			Name:      "krkn-operator-jwt",
			Namespace: namespace,
		}, secret); err != nil {
			if apierrors.IsNotFound(err) {
				logger.V(1).Info("JWT secret not found, skipping")
				return nil
			}
			return fmt.Errorf("failed to get JWT secret: %w", err)
		}
		obj := &unstructured.Unstructured{}
		if err := k8sClient.Scheme().Convert(secret, obj, nil); err != nil {
			return fmt.Errorf("failed to convert secret: %w", err)
		}
		items = append(items, *obj)
	} else if resType.gvk.Kind == "ConfigMap" && resType.singular == "acm-config" {
		cm := &corev1.ConfigMap{}
		if err := k8sClient.Get(ctx, client.ObjectKey{
			Name:      "krkn-operator-acm-config",
			Namespace: namespace,
		}, cm); err != nil {
			if apierrors.IsNotFound(err) {
				logger.V(1).Info("ACM config not found, skipping")
				return nil
			}
			return fmt.Errorf("failed to get ACM config: %w", err)
		}
		obj := &unstructured.Unstructured{}
		if err := k8sClient.Scheme().Convert(cm, obj, nil); err != nil {
			return fmt.Errorf("failed to convert configmap: %w", err)
		}
		items = append(items, *obj)
	} else {
		listObj := &unstructured.UnstructuredList{}
		listObj.SetGroupVersionKind(resType.gvk)
		opts := []client.ListOption{client.InNamespace(namespace)}

		if len(resType.selector) > 0 {
			opts = append(opts, client.MatchingLabels(resType.selector))
		}
		if len(resType.hasLabels) > 0 {
			opts = append(opts, client.HasLabels(resType.hasLabels))
		}

		if err := k8sClient.List(ctx, listObj, opts...); err != nil {
			if meta.IsNoMatchError(err) {
				logger.V(1).Info("Resource type not registered, skipping", "type", resType.gvk.Kind)
				return nil
			}
			return fmt.Errorf("failed to list %s resources: %w", resType.singular, err)
		}

		items = listObj.Items
	}

	if len(items) == 0 {
		logger.V(1).Info("No resources found", "type", resType.gvk.Kind)
		return nil
	}

	// Build list response
	listResponse := map[string]interface{}{
		"apiVersion": resType.gvk.GroupVersion().String(),
		"kind":       "List",
		"items":      items,
	}

	// Strip metadata from each item
	strippedItems := make([]interface{}, len(items))
	for i, item := range items {
		strippedItems[i] = stripMetadata(item.Object)
	}
	listResponse["items"] = strippedItems

	// Write to file
	data, err := json.MarshalIndent(listResponse, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal resources: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	logger.V(1).Info("Backed up resource type", "type", resType.gvk.Kind, "count", len(items))
	return nil
}

// stripMetadata removes cluster-specific metadata from a resource
func stripMetadata(obj map[string]interface{}) map[string]interface{} {
	if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
		// Delete cluster-specific fields
		delete(metadata, "resourceVersion")
		delete(metadata, "uid")
		delete(metadata, "creationTimestamp")
		delete(metadata, "generation")
		delete(metadata, "managedFields")

		// Clean up annotations
		if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
			delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
			if len(annotations) == 0 {
				delete(metadata, "annotations")
			}
		}

		obj["metadata"] = metadata
	}
	return obj
}

// createTarGz creates a compressed tar archive from a directory
func createTarGz(sourceDir, targetPath string) error {
	tarFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer tarFile.Close()

	gzWriter := gzip.NewWriter(tarFile)
	tarWriter := tar.NewWriter(gzWriter)

	baseDir := filepath.Base(sourceDir)

	walkErr := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = filepath.Join(baseDir, relPath)

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})

	if walkErr != nil {
		tarWriter.Close()
		gzWriter.Close()
		return walkErr
	}

	if err := tarWriter.Close(); err != nil {
		gzWriter.Close()
		return fmt.Errorf("failed to finalize tar archive: %w", err)
	}

	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize gzip compression: %w", err)
	}

	return nil
}
