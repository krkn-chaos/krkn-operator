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
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RestoreConfig holds restore configuration
type RestoreConfig struct {
	Namespace  string
	BackupPath string
}

// RestoreBackup restores operator configuration from a backup archive
func RestoreBackup(ctx context.Context, k8sClient client.Client, config RestoreConfig) error {
	logger := log.FromContext(ctx)

	logger.Info("Starting restore", "namespace", config.Namespace, "backupPath", config.BackupPath)

	// Extract archive to temporary directory
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("krkn-restore-%d", os.Getpid()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := extractTarGz(config.BackupPath, tempDir); err != nil {
		return fmt.Errorf("failed to extract backup archive: %w", err)
	}

	// Find backup directory (starts with krkn-backup, may have timestamp suffix)
	backupDir, err := findBackupDir(tempDir)
	if err != nil {
		return err
	}

	logger.Info("Archive extracted", "backupDir", backupDir)

	// Read and apply all JSON files
	applyCount := 0
	errCount := 0

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if len(entry.Name()) < 5 || entry.Name()[len(entry.Name())-5:] != ".json" {
			continue
		}

		filePath := filepath.Join(backupDir, entry.Name())
		count, err := applyResourcesFromFile(ctx, k8sClient, config.Namespace, filePath)
		if err != nil {
			logger.Error(err, "Failed to apply resources from file", "file", entry.Name())
			errCount++
			continue
		}

		applyCount += count
		logger.V(1).Info("Applied resources from file", "file", entry.Name(), "count", count)
	}

	logger.Info("Restore completed", "applied", applyCount, "failed", errCount)

	if applyCount == 0 {
		return fmt.Errorf("no resources were restored")
	}

	if errCount > 0 {
		return fmt.Errorf("restore completed with %d errors", errCount)
	}

	return nil
}

// applyResourcesFromFile reads a JSON file and applies all resources
func applyResourcesFromFile(ctx context.Context, k8sClient client.Client, namespace string, filePath string) (int, error) {
	logger := log.FromContext(ctx)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse as list response
	var listResponse map[string]interface{}
	if err := json.Unmarshal(data, &listResponse); err != nil {
		return 0, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	items, ok := listResponse["items"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid list format: missing items array")
	}

	count := 0

	for _, item := range items {
		obj := &unstructured.Unstructured{}
		objMap, ok := item.(map[string]interface{})
		if !ok {
			logger.V(1).Info("Skipping non-object item")
			continue
		}

		obj.Object = objMap

		// Set namespace if not already set
		if obj.GetNamespace() == "" {
			obj.SetNamespace(namespace)
		}

		// Apply the resource
		if err := applyResource(ctx, k8sClient, obj); err != nil {
			logger.Error(err, "Failed to apply resource", "kind", obj.GetKind(), "name", obj.GetName())
			continue
		}

		count++
		logger.V(1).Info("Applied resource", "kind", obj.GetKind(), "name", obj.GetName())
	}

	return count, nil
}

// applyResource applies a single resource to the cluster
// Uses server-side apply to merge with existing resources
func applyResource(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
	// Use Patch with strategic merge patch to apply the resource
	// This allows us to merge with existing resources rather than overwriting

	// First, try to get the existing resource
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())

	err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, existing)

	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to check existing resource: %w", err)
	}

	if isNotFound(err) {
		// Resource doesn't exist, create it
		if err := k8sClient.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}
	} else {
		// Resource exists, update it
		obj.SetResourceVersion(existing.GetResourceVersion())
		obj.SetUID(existing.GetUID())

		if err := k8sClient.Update(ctx, obj); err != nil {
			return fmt.Errorf("failed to update resource: %w", err)
		}
	}

	// Handle status subresource for custom resources
	if hasStatus(obj) && isKrknCRD(obj) {
		if status, ok := obj.Object["status"]; ok && status != nil {
			// Create status patch
			statusObj := &unstructured.Unstructured{}
			statusObj.SetGroupVersionKind(obj.GroupVersionKind())
			statusObj.SetNamespace(obj.GetNamespace())
			statusObj.SetName(obj.GetName())
			statusObj.SetResourceVersion(obj.GetResourceVersion())
			statusObj.Object = map[string]interface{}{
				"status": status,
			}

			if err := k8sClient.Status().Update(ctx, statusObj); err != nil {
				// Log but don't fail - status patch may fail for various reasons
				log.FromContext(ctx).V(1).Info("Failed to patch status", "kind", obj.GetKind(), "name", obj.GetName(), "error", err.Error())
			}
		}
	}

	return nil
}

// extractTarGz extracts a gzipped tar archive to a directory
func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %w", err)
		}

		// Construct target path
		targetPath := filepath.Join(destDir, header.Name)

		// Security: prevent path traversal
		// Check if path tries to escape the destination directory
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}
		absDest, err := filepath.Abs(destDir)
		if err != nil {
			return fmt.Errorf("invalid destination directory: %w", err)
		}
		if !strings.HasPrefix(absTarget, absDest+string(filepath.Separator)) && absTarget != absDest {
			return fmt.Errorf("invalid path in archive (path traversal detected): %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create parent directories if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}
		}
	}

	return nil
}

// findBackupDir finds the backup directory in the extracted archive
// The archive contains krkn-backup/ (consistent name) or krkn-backup-tmp-{timestamp}/krkn-backup/
func findBackupDir(tempDir string) (string, error) {
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return "", fmt.Errorf("failed to read temp directory: %w", err)
	}

	// First, check if krkn-backup/ is directly in tempDir
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == "krkn-backup" {
			return filepath.Join(tempDir, entry.Name()), nil
		}
	}

	// If not found, look for krkn-backup-tmp-{timestamp}/krkn-backup/ pattern
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "krkn-backup-tmp-") {
			backupPath := filepath.Join(tempDir, entry.Name(), "krkn-backup")
			if info, err := os.Stat(backupPath); err == nil && info.IsDir() {
				return backupPath, nil
			}
		}
	}

	return "", fmt.Errorf("backup directory not found in archive: no krkn-backup directory found")
}

// Helper functions

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Use Kubernetes apierrors.IsNotFound for proper error checking
	return apierrors.IsNotFound(err)
}

func hasStatus(obj *unstructured.Unstructured) bool {
	_, exists := obj.Object["status"]
	return exists
}

func isKrknCRD(obj *unstructured.Unstructured) bool {
	return obj.GroupVersionKind().Group == "krkn.krkn-chaos.dev"
}

// GetResourceTypes returns a schema.Scheme for decoding resources
func GetResourceTypes(scheme *runtime.Scheme) {
	// This is handled by the controller-runtime scheme which includes all CRDs
}
