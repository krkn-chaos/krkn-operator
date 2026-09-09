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
	"os"
	"path/filepath"
	"testing"
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

// CreateBackup and RestoreBackup are tested through handler-level integration tests
// in internal/api/backup_restore_handlers_test.go which verify the full flow including
// archive creation, extraction, resource application, and error handling.
