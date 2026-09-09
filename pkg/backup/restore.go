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

// This file implements restore functionality for backup archives created by CreateBackup.
// See package documentation for usage examples.

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

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// allowedRestoreGVKs defines the set of resource types that restore is permitted to apply.
var allowedRestoreGVKs = map[schema.GroupVersionKind]bool{
	{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknUser"}:                   true,
	{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknUserGroup"}:              true,
	{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknOperatorTarget"}:         true,
	{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Kind: "KrknOperatorTargetProvider"}: true,
	{Group: "", Version: "v1", Kind: "Secret"}:                                              true,
	{Group: "", Version: "v1", Kind: "ConfigMap"}:                                           true,
}

// RestoreConfig holds restore configuration.
type RestoreConfig struct {
	// Namespace is the Kubernetes namespace where resources will be restored.
	Namespace string
	// BackupPath is the path to a backup archive created by CreateBackup.
	// Must be a gzipped tar archive (.tar.gz) with a krkn-backup/ directory.
	BackupPath string
}

// RestoreBackup restores operator configuration from a backup archive
func RestoreBackup(ctx context.Context, k8sClient client.Client, config RestoreConfig) error {
	logger := log.FromContext(ctx)

	logger.Info("Starting restore", "namespace", config.Namespace, "backupPath", config.BackupPath)

	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("krkn-restore-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0700); err != nil {
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

		applied, failed, err := applyResourcesFromFile(ctx, k8sClient, config.Namespace, backupDir, entry.Name())
		if err != nil {
			logger.Error(err, "Failed to apply resources from file", "file", entry.Name())
			errCount++
			continue
		}

		applyCount += applied
		errCount += failed
		logger.V(1).Info("Applied resources from file", "file", entry.Name(), "applied", applied, "failed", failed)
	}

	logger.Info("Restore completed", "applied", applyCount, "failed", errCount)

	if applyCount == 0 {
		return fmt.Errorf("no resources were restored")
	}

	if errCount > 0 {
		return fmt.Errorf("restore completed with %d resource failures out of %d total", errCount, applyCount+errCount)
	}

	if err := refreshTargetStatuses(ctx, k8sClient, config.Namespace); err != nil {
		return fmt.Errorf("failed to refresh restored target status: %w", err)
	}

	return nil
}

// applyResourcesFromFile reads a JSON file and applies all resources.
// Returns the number of successfully applied resources and the number of failures.
func applyResourcesFromFile(ctx context.Context, k8sClient client.Client, namespace, backupDir, fileName string) (applied int, failed int, err error) {
	logger := log.FromContext(ctx)

	backupRoot, err := os.OpenRoot(backupDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open backup directory: %w", err)
	}
	defer backupRoot.Close()

	data, err := backupRoot.ReadFile(fileName)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read file: %w", err)
	}

	var listResponse map[string]interface{}
	if err := json.Unmarshal(data, &listResponse); err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	items, ok := listResponse["items"].([]interface{})
	if !ok {
		return 0, 0, fmt.Errorf("invalid list format: missing items array")
	}

	for _, item := range items {
		obj := &unstructured.Unstructured{}
		objMap, ok := item.(map[string]interface{})
		if !ok {
			logger.V(1).Info("Skipping non-object item")
			failed++
			continue
		}

		obj.Object = objMap

		gvk := obj.GroupVersionKind()
		if !allowedRestoreGVKs[gvk] {
			logger.V(1).Info("Skipping disallowed resource type", "gvk", gvk.String())
			failed++
			continue
		}

		obj.SetNamespace(namespace)

		// Validate Secrets: must have required labels to prevent unauthorized overwrites
		if obj.GetKind() == "Secret" {
			if obj.GetName() == "krkn-operator-jwt" {
				logger.V(1).Info("Rejecting JWT signing Secret from portable restore")
				failed++
				continue
			}
			if !isOperatorManagedSecret(obj) {
				logger.V(1).Info("Rejecting Secret without operator management label", "name", obj.GetName())
				failed++
				continue
			}
		}
		if obj.GetKind() == "ConfigMap" && !isOperatorManagedConfigMap(obj) {
			logger.V(1).Info("Rejecting ConfigMap without provider configuration label", "name", obj.GetName())
			failed++
			continue
		}

		if applyErr := applyResource(ctx, k8sClient, obj); applyErr != nil {
			logger.Error(applyErr, "Failed to apply resource", "kind", obj.GetKind(), "name", obj.GetName())
			failed++
			continue
		}

		applied++
		logger.V(1).Info("Applied resource", "kind", obj.GetKind(), "name", obj.GetName())
	}

	return applied, failed, nil
}

// applyResource applies a single resource to the cluster.
// Creates the resource if it does not exist, or updates it if it does.
func applyResource(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
	archivedStatus, hasArchivedStatus, err := unstructured.NestedFieldCopy(obj.Object, "status")
	if err != nil {
		return fmt.Errorf("failed to read archived status: %w", err)
	}

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())

	err = k8sClient.Get(ctx, client.ObjectKey{
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

	// The normal create/update response can omit the status subresource. Use
	// the copy captured before that API call when restoring archived status.
	if hasArchivedStatus && archivedStatus != nil && isKrknCRD(obj) {
		if err := restoreArchivedStatus(ctx, k8sClient, obj, archivedStatus); err != nil {
			return err
		}
	}

	return nil
}

func restoreArchivedStatus(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured, status interface{}) error {
	freshObj := &unstructured.Unstructured{}
	freshObj.SetGroupVersionKind(obj.GroupVersionKind())
	if err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, freshObj); err != nil {
		return fmt.Errorf("failed to get resource for status update: %w", err)
	}

	if err := unstructured.SetNestedField(freshObj.Object, status, "status"); err != nil {
		return fmt.Errorf("failed to set archived status: %w", err)
	}
	if err := k8sClient.Status().Update(ctx, freshObj); err != nil {
		return fmt.Errorf("failed to update resource status: %w", err)
	}
	return nil
}

// refreshTargetStatuses determines readiness from the destination cluster,
// rather than trusting the source cluster's archived readiness value.
func refreshTargetStatuses(ctx context.Context, k8sClient client.Client, namespace string) error {
	logger := log.FromContext(ctx)
	targets := &krknv1alpha1.KrknOperatorTargetList{}
	if err := k8sClient.List(ctx, targets, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list restored targets: %w", err)
	}

	for i := range targets.Items {
		target := &targets.Items[i]
		ready := false

		secret := &corev1.Secret{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      target.Spec.SecretUUID,
		}, secret)
		if err == nil {
			if secretData, ok := secret.Data["kubeconfig"]; ok {
				kubeconfigBase64, decodeErr := kubeconfig.UnmarshalSecretData(secretData)
				if decodeErr == nil {
					if livenessErr := provider.CheckClusterLiveness(ctx, kubeconfigBase64, 0); livenessErr == nil {
						ready = true
					} else {
						logger.Info("Restored target liveness check failed", "clusterName", target.Spec.ClusterName, "error", livenessErr)
					}
				} else {
					logger.Info("Restored target kubeconfig could not be decoded", "clusterName", target.Spec.ClusterName, "error", decodeErr)
				}
			}
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Secret for restored target %q: %w", target.Spec.ClusterName, err)
		} else {
			logger.Info("Restored target Secret was not found", "clusterName", target.Spec.ClusterName)
		}

		target.Status.Ready = ready
		target.Status.LastUpdated = metav1.Now()
		if err := k8sClient.Status().Update(ctx, target); err != nil {
			return fmt.Errorf("failed to update restored target %q status: %w", target.Spec.ClusterName, err)
		}
	}

	return nil
}

// extractTarGz extracts a gzipped tar archive to a directory
func extractTarGz(archivePath, destDir string) error {
	archiveRoot, err := os.OpenRoot(filepath.Dir(archivePath))
	if err != nil {
		return fmt.Errorf("failed to open archive directory: %w", err)
	}
	defer archiveRoot.Close()

	file, err := archiveRoot.Open(filepath.Base(archivePath))
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
	destRoot, err := os.OpenRoot(destDir)
	if err != nil {
		return fmt.Errorf("failed to open extraction directory: %w", err)
	}
	defer destRoot.Close()

	cumulativeBytes := int64(0)
	fileCount := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %w", err)
		}

		// Archive entries are slash-separated. Convert them to the current
		// platform's path format before validating and opening them beneath destRoot.
		relPath := filepath.Clean(filepath.FromSlash(header.Name))
		if header.Name == "" || filepath.IsAbs(relPath) || relPath == "." || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid path in archive (path traversal detected): %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := destRoot.MkdirAll(relPath, 0700); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			if err := destRoot.MkdirAll(filepath.Dir(relPath), 0700); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			written, err := extractFile(destRoot, relPath, tarReader)
			if err != nil {
				return err
			}
			cumulativeBytes += written
			fileCount++

			// Prevent cumulative exhaustion of pod storage
			const maxCumulativeExtractSize = 1 << 30 // 1 GB
			if cumulativeBytes > maxCumulativeExtractSize {
				return fmt.Errorf("extracted archive exceeds maximum total size (1 GB): cumulative=%d bytes, files=%d", cumulativeBytes, fileCount)
			}
		}
	}

	return nil
}

// 256 MB limit per extracted file to prevent pod storage exhaustion
const maxExtractFileSize = 256 << 20

func extractFile(root *os.Root, relativePath string, r io.Reader) (int64, error) {
	outFile, err := root.OpenFile(relativePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, io.LimitReader(r, maxExtractFileSize+1))
	if err != nil {
		return 0, fmt.Errorf("failed to extract file: %w", err)
	}
	if written > maxExtractFileSize {
		return 0, fmt.Errorf("file %s exceeds maximum allowed size (%d bytes)", filepath.Base(relativePath), maxExtractFileSize)
	}
	return written, nil
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

// isOperatorManagedSecret checks if a Secret has labels indicating it was in the backup
// Accepts Secrets that match any of the backup selectors:
// - krkn-target-uuid (target credentials)
// - app.kubernetes.io/component: authentication (auth secrets)
// - app.kubernetes.io/component: user-auth (user auth secrets)
// - app.kubernetes.io/component: elasticsearch-config (elasticsearch secrets)
// - app.kubernetes.io/component: registry (registry secrets)
func isOperatorManagedSecret(obj *unstructured.Unstructured) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}

	// Allow target secrets (have krkn-target-uuid label)
	if _, ok := labels["krkn-target-uuid"]; ok {
		return true
	}

	// Allow secrets with component label matching any backup selector
	if component, ok := labels["app.kubernetes.io/component"]; ok {
		switch component {
		case "authentication", "user-auth", "elasticsearch-config", "registry":
			return true
		}
	}

	return false
}

func isOperatorManagedConfigMap(obj *unstructured.Unstructured) bool {
	labels := obj.GetLabels()
	return labels != nil && labels[provider.ProviderConfigLabel] == provider.ProviderConfigLabelValue
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return apierrors.IsNotFound(err)
}

func isKrknCRD(obj *unstructured.Unstructured) bool {
	return obj.GroupVersionKind().Group == "krkn.krkn-chaos.dev"
}
