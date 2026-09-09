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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

func TestStripMetadata(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":              "my-secret",
			"namespace":         "default",
			"resourceVersion":   "12345",
			"uid":               "abc-123",
			"creationTimestamp": "2024-01-01T00:00:00Z",
			"generation":        int64(1),
			"managedFields":     []interface{}{},
			"labels":            map[string]interface{}{"app": "test"},
			"annotations": map[string]interface{}{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
				"custom-annotation": "keep-me",
			},
		},
	}

	result := stripMetadata(obj)

	metadata := result["metadata"].(map[string]interface{})

	if _, ok := metadata["resourceVersion"]; ok {
		t.Error("resourceVersion should be stripped")
	}
	if _, ok := metadata["uid"]; ok {
		t.Error("uid should be stripped")
	}
	if _, ok := metadata["creationTimestamp"]; ok {
		t.Error("creationTimestamp should be stripped")
	}
	if _, ok := metadata["generation"]; ok {
		t.Error("generation should be stripped")
	}
	if _, ok := metadata["managedFields"]; ok {
		t.Error("managedFields should be stripped")
	}

	if metadata["name"] != "my-secret" {
		t.Error("name should be preserved")
	}
	if metadata["namespace"] != "default" {
		t.Error("namespace should be preserved")
	}
	if _, ok := metadata["labels"]; !ok {
		t.Error("labels should be preserved")
	}

	annotations := metadata["annotations"].(map[string]interface{})
	if _, ok := annotations["kubectl.kubernetes.io/last-applied-configuration"]; ok {
		t.Error("last-applied-configuration annotation should be stripped")
	}
	if annotations["custom-annotation"] != "keep-me" {
		t.Error("custom annotations should be preserved")
	}
}

func TestStripMetadataRemovesEmptyAnnotations(t *testing.T) {
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test",
			"annotations": map[string]interface{}{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
			},
		},
	}

	result := stripMetadata(obj)
	metadata := result["metadata"].(map[string]interface{})

	if _, ok := metadata["annotations"]; ok {
		t.Error("Empty annotations map should be removed")
	}
}

func TestValidateBackupName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "backup-2026-01-01", wantErr: false},
		{name: "", wantErr: true},
		{name: ".", wantErr: true},
		{name: "..", wantErr: true},
		{name: "nested/backup", wantErr: true},
		{name: `nested\\backup`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBackupName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateBackupName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestCreateAndExtractTarGz(t *testing.T) {
	// Mimic the real backup layout: tempParent/krkn-backup/test.json
	// createTarGz uses filepath.Base(sourceDir) as the root entry in the archive
	tempParent := filepath.Join(t.TempDir(), "krkn-backup-tmp")
	contentDir := filepath.Join(tempParent, "krkn-backup")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	testContent := []byte(`{"items": [{"name": "test"}]}`)
	if err := os.WriteFile(filepath.Join(contentDir, "test.json"), testContent, 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "test-backup.tar.gz")
	// Archive from the parent that contains krkn-backup/
	if err := createTarGz(tempParent, archivePath); err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("Archive not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Archive is empty")
	}

	// Extract and verify round-trip
	extractDir := t.TempDir()
	if err := extractTarGz(archivePath, extractDir); err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// Archive root is basename of sourceDir ("krkn-backup-tmp"), containing "krkn-backup/"
	extractedFile := filepath.Join(extractDir, "krkn-backup-tmp", "krkn-backup", "test.json")
	data, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}
	if string(data) != string(testContent) {
		t.Errorf("Extracted content mismatch: got %q, want %q", string(data), string(testContent))
	}
}

func TestExtractTarGzPathTraversal(t *testing.T) {
	// Create a malicious tar.gz with a path traversal entry
	archivePath := filepath.Join(t.TempDir(), "malicious.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add an entry that tries to escape the destination directory
	header := &tar.Header{
		Name: "../../../etc/evil",
		Mode: 0644,
		Size: 4,
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}

	tw.Close()
	gw.Close()
	f.Close()

	destDir := t.TempDir()
	err = extractTarGz(archivePath, destDir)
	if err == nil {
		t.Fatal("Expected error for path traversal, got nil")
	}
}

func TestFindBackupDir(t *testing.T) {
	t.Run("direct krkn-backup directory", func(t *testing.T) {
		tempDir := t.TempDir()
		backupDir := filepath.Join(tempDir, "krkn-backup")
		os.MkdirAll(backupDir, 0755)

		found, err := findBackupDir(tempDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if found != backupDir {
			t.Errorf("Expected %s, got %s", backupDir, found)
		}
	})

	t.Run("nested krkn-backup-tmp pattern", func(t *testing.T) {
		tempDir := t.TempDir()
		nestedDir := filepath.Join(tempDir, "krkn-backup-tmp-1234567890", "krkn-backup")
		os.MkdirAll(nestedDir, 0755)

		found, err := findBackupDir(tempDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if found != nestedDir {
			t.Errorf("Expected %s, got %s", nestedDir, found)
		}
	})

	t.Run("no backup directory", func(t *testing.T) {
		tempDir := t.TempDir()
		os.MkdirAll(filepath.Join(tempDir, "other-dir"), 0755)

		_, err := findBackupDir(tempDir)
		if err == nil {
			t.Error("Expected error for missing backup directory")
		}
	})
}

func TestCreateTarGzFlushErrors(t *testing.T) {
	// createTarGz should succeed with valid input and properly flush
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "out.tar.gz")
	if err := createTarGz(sourceDir, archivePath); err != nil {
		t.Fatalf("createTarGz should succeed: %v", err)
	}

	// Verify the archive is valid by reading it back
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("Archive is not valid gzip: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	fileCount := 0
	for {
		_, err := tr.Next()
		if err != nil {
			break
		}
		fileCount++
	}
	if fileCount == 0 {
		t.Error("Archive contains no entries")
	}
}

func TestApplyResourceRestoresArchivedStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	k8sClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&krknv1alpha1.KrknOperatorTarget{}).
		Build()

	target := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "krkn.krkn-chaos.dev/v1alpha1",
		"kind":       "KrknOperatorTarget",
		"metadata": map[string]interface{}{
			"name":      "target",
			"namespace": "test",
		},
		"spec": map[string]interface{}{
			"clusterName": "cluster",
			"secretType":  "kubeconfig",
			"secretUUID":  "target-secret",
			"uuid":        "target",
		},
		"status": map[string]interface{}{
			"ready": true,
		},
	}}

	if err := applyResource(context.Background(), k8sClient, target); err != nil {
		t.Fatalf("applyResource() error = %v", err)
	}

	var restored krknv1alpha1.KrknOperatorTarget
	if err := k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: "test",
		Name:      "target",
	}, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Status.Ready {
		t.Fatal("archived status.ready was not restored")
	}
}

func TestRefreshTargetStatusesMarksUnreachableTargetNotReady(t *testing.T) {
	kubeconfigBase64, err := kubeconfig.GenerateFromToken("cluster", "https://127.0.0.1:1", "", "token", true)
	if err != nil {
		t.Fatal(err)
	}
	secretData, err := kubeconfig.MarshalSecretData(kubeconfigBase64)
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	k8sClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&krknv1alpha1.KrknOperatorTarget{}).
		WithObjects(
			&krknv1alpha1.KrknOperatorTarget{
				ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "test"},
				Spec: krknv1alpha1.KrknOperatorTargetSpec{
					ClusterName: "cluster",
					SecretType:  "kubeconfig",
					SecretUUID:  "target-secret",
					UUID:        "target",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "target-secret", Namespace: "test"},
				Data:       map[string][]byte{"kubeconfig": secretData},
			},
		).
		Build()

	if err := refreshTargetStatuses(context.Background(), k8sClient, "test"); err != nil {
		t.Fatalf("refreshTargetStatuses() error = %v", err)
	}

	var target krknv1alpha1.KrknOperatorTarget
	if err := k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: "test",
		Name:      "target",
	}, &target); err != nil {
		t.Fatal(err)
	}
	if target.Status.Ready {
		t.Fatal("unreachable destination target should not be ready")
	}
}

func TestCreateBackupIncludesLabeledProviderConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	const namespace = "krkn-operator-system"
	k8sClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      provider.LegacyProviderConfigMapName,
				Namespace: namespace,
			},
			Data: map[string]string{"API_PORT": "8080"},
		}).
		Build()

	changed, err := provider.BackfillLegacyProviderConfigLabel(context.Background(), k8sClient, namespace)
	if err != nil {
		t.Fatalf("BackfillLegacyProviderConfigLabel() error = %v", err)
	}
	if !changed {
		t.Fatal("BackfillLegacyProviderConfigLabel() changed = false, want true")
	}

	archivePath, err := CreateBackup(context.Background(), k8sClient, BackupConfig{
		Namespace:  namespace,
		OutputDir:  t.TempDir(),
		BackupName: "provider-config-backup",
	})
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	archiveFile, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archiveFile.Close()

	gzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	found := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(header.Name) != "provider-configmaps.json" {
			continue
		}

		contents, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(contents), provider.LegacyProviderConfigMapName) {
			t.Fatalf("provider ConfigMap missing from backup: %s", contents)
		}
		found = true
	}

	if !found {
		t.Fatal("provider-configmaps.json not found in backup")
	}
}

// CreateBackup and RestoreBackup are tested through handler-level integration tests
// in internal/api/backup_restore_handlers_test.go which verify the full flow including
// archive creation, extraction, resource application, and error handling.
