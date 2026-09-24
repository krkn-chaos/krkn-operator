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

// Package backup provides backup and restore functionality for krkn-operator configuration.
//
// CreateBackup serializes the configuration exposed by the Settings page into
// a compressed tar.gz archive.
//
// RestoreBackup extracts and applies resources from a backup archive with a GVK allowlist
// and ownership validation to prevent unauthorized overwrites.
//
// Usage:
//
//	config := BackupConfig{
//	  Namespace:  "default",
//	  OutputDir:  "/tmp",
//	  BackupName: "my-backup",
//	}
//	archivePath, err := CreateBackup(ctx, k8sClient, config)
//
//	restoreConfig := RestoreConfig{
//	  Namespace:  "default",
//	  BackupPath: archivePath,
//	}
//	err = RestoreBackup(ctx, k8sClient, restoreConfig)
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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// BackupConfig holds backup configuration.
type BackupConfig struct {
	// Namespace is the Kubernetes namespace containing resources to backup.
	Namespace string
	// OutputDir is the directory where the backup archive will be written.
	// If empty, defaults to current directory.
	OutputDir string
	// BackupName is the name prefix for the backup archive (without .tar.gz extension).
	// If empty, a timestamp-based name is generated.
	BackupName string
}

// resourceType describes a resource to backup
type resourceType struct {
	gvk          schema.GroupVersionKind
	singular     string
	selector     client.MatchingLabels
	hasLabels    []string
	excludeNames []string
}

// CreateBackup creates a backup of operator configuration
func CreateBackup(ctx context.Context, k8sClient client.Client, config BackupConfig) (string, error) {
	logger := log.FromContext(ctx)

	if config.OutputDir == "" {
		config.OutputDir = "."
	}
	if err := os.MkdirAll(config.OutputDir, 0700); err != nil {
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

	// Keep this list limited to the Settings page. Runtime registration objects,
	// ACM integration data, file metadata, and JWT signing keys are not portable
	// operator settings.
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
		// Secrets (by labels)
		{
			gvk:          schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			singular:     "auth-secrets",
			selector:     map[string]string{"app.kubernetes.io/component": "authentication"},
			excludeNames: []string{"krkn-operator-jwt"},
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
		// Provider configuration values are stored in explicitly labeled ConfigMaps.
		{
			gvk:      schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
			singular: "provider-configmaps",
			selector: map[string]string{provider.ProviderConfigLabel: provider.ProviderConfigLabelValue},
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
			if removeErr := os.Remove(outputPath); removeErr != nil && !os.IsNotExist(removeErr) {
				logger.Error(removeErr, "Failed to remove empty backup file", "path", outputPath)
			}
		}
	}

	logger.Info("Resource backup completed", "backed_up", backupCount, "failed", errCount)

	if backupCount == 0 {
		return "", fmt.Errorf("no resources were backed up")
	}

	if errCount > 0 {
		return "", fmt.Errorf("backup incomplete: %d resource types failed to backup (backed up %d successfully)", errCount, backupCount)
	}

	// Create tar.gz archive
	archiveName := config.BackupName
	if archiveName == "" {
		archiveName = fmt.Sprintf("krkn-backup-%d", time.Now().Unix())
	}
	if err := validateBackupName(archiveName); err != nil {
		return "", err
	}
	archivePath := filepath.Join(config.OutputDir, archiveName+".tar.gz")

	// Write to a temp file first, then rename for atomic creation
	tmpArchivePath := archivePath + ".tmp"
	if err := createTarGz(tempParent, tmpArchivePath); err != nil {
		if removeErr := os.Remove(tmpArchivePath); removeErr != nil && !os.IsNotExist(removeErr) {
			logger.Error(removeErr, "Failed to remove temporary archive", "path", tmpArchivePath)
		}
		return "", fmt.Errorf("failed to create archive: %w", err)
	}
	if err := os.Rename(tmpArchivePath, archivePath); err != nil {
		if removeErr := os.Remove(tmpArchivePath); removeErr != nil && !os.IsNotExist(removeErr) {
			logger.Error(removeErr, "Failed to remove temporary archive", "path", tmpArchivePath)
		}
		return "", fmt.Errorf("failed to finalize archive: %w", err)
	}

	logger.Info("Backup completed successfully", "archivePath", archivePath, "backupCount", backupCount)

	return archivePath, nil
}

func validateBackupName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return fmt.Errorf("invalid backup name: must be a single file name")
	}
	return nil
}

// backupResourceType backs up all instances of a specific resource type
func backupResourceType(ctx context.Context, k8sClient client.Client, namespace string, resType resourceType, outputPath string) error {
	logger := log.FromContext(ctx)

	// List resources based on resource type
	var items []unstructured.Unstructured

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
	if len(resType.excludeNames) > 0 {
		filtered := items[:0]
		for _, item := range items {
			excluded := false
			for _, name := range resType.excludeNames {
				if item.GetName() == name {
					excluded = true
					break
				}
			}
			if !excluded {
				filtered = append(filtered, item)
			}
		}
		items = filtered
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
func createTarGz(sourceDir, targetPath string) (retErr error) {
	targetRoot, err := os.OpenRoot(filepath.Dir(targetPath))
	if err != nil {
		return fmt.Errorf("failed to open archive output directory: %w", err)
	}
	defer targetRoot.Close()

	tarFile, err := targetRoot.OpenFile(filepath.Base(targetPath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer func() {
		if closeErr := tarFile.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close tar file: %w", closeErr)
		}
	}()

	sourceRoot, err := os.OpenRoot(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to open source directory: %w", err)
	}
	defer sourceRoot.Close()

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

		file, err := sourceRoot.Open(relPath)
		if err != nil {
			return err
		}

		_, copyErr := io.Copy(tarWriter, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})

	if walkErr != nil {
		if closeErr := tarWriter.Close(); closeErr != nil {
			return fmt.Errorf("archive walk failed: %w; failed to close tar writer: %v", walkErr, closeErr)
		}
		if closeErr := gzWriter.Close(); closeErr != nil {
			return fmt.Errorf("archive walk failed: %w; failed to close gzip writer: %v", walkErr, closeErr)
		}
		return walkErr
	}

	if err := tarWriter.Close(); err != nil {
		if gzipErr := gzWriter.Close(); gzipErr != nil {
			return fmt.Errorf("failed to close tar writer: %w; failed to close gzip writer: %v", err, gzipErr)
		}
		return fmt.Errorf("failed to finalize tar archive: %w", err)
	}

	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize gzip compression: %w", err)
	}

	return nil
}
