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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package controller

import (
	"context"
	"encoding/base64"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
)

func TestTranslateVolumesToFileMounts(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name          string
		volumes       map[string]string
		setupFiles    []*corev1.ConfigMap
		expectError   bool
		expectCount   int
		validateMount func(t *testing.T, mounts []krknv1alpha1.FileMount)
	}{
		{
			name:        "empty volumes",
			volumes:     nil,
			setupFiles:  []*corev1.ConfigMap{},
			expectError: false,
			expectCount: 0,
		},
		{
			name: "single file valid",
			volumes: map[string]string{
				"550e8400-e29b-41d4-a716-446655440001": "/config/test.yaml",
			},
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440001",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "550e8400-e29b-41d4-a716-446655440001",
						},
					},
					Data: map[string]string{
						"test.yaml": "key: value\nfoo: bar",
					},
				},
			},
			expectError: false,
			expectCount: 1,
			validateMount: func(t *testing.T, mounts []krknv1alpha1.FileMount) {
				if mounts[0].Name != "test.yaml" {
					t.Errorf("expected name 'test.yaml', got '%s'", mounts[0].Name)
				}
				if mounts[0].MountPath != "/config/test.yaml" {
					t.Errorf("expected mount path '/config/test.yaml', got '%s'", mounts[0].MountPath)
				}
				// Decode content and verify
				decoded, err := base64.StdEncoding.DecodeString(mounts[0].Content)
				if err != nil {
					t.Fatalf("failed to decode content: %v", err)
				}
				expectedContent := "key: value\nfoo: bar"
				if string(decoded) != expectedContent {
					t.Errorf("expected content '%s', got '%s'", expectedContent, string(decoded))
				}
			},
		},
		{
			name: "multiple files",
			volumes: map[string]string{
				"550e8400-e29b-41d4-a716-446655440001": "/config/test1.yaml",
				"550e8400-e29b-41d4-a716-446655440002": "/data/test2.json",
			},
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440001",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "550e8400-e29b-41d4-a716-446655440001",
						},
					},
					Data: map[string]string{
						"test1.yaml": "config: value",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440002",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "550e8400-e29b-41d4-a716-446655440002",
						},
					},
					Data: map[string]string{
						"test2.json": `{"data": "value"}`,
					},
				},
			},
			expectError: false,
			expectCount: 2,
			validateMount: func(t *testing.T, mounts []krknv1alpha1.FileMount) {
				// Find mounts by name (order not guaranteed)
				var mount1, mount2 *krknv1alpha1.FileMount
				for i := range mounts {
					if mounts[i].Name == "test1.yaml" {
						mount1 = &mounts[i]
					}
					if mounts[i].Name == "test2.json" {
						mount2 = &mounts[i]
					}
				}

				if mount1 == nil || mount2 == nil {
					t.Fatal("expected both mounts to be present")
				}

				if mount1.MountPath != "/config/test1.yaml" {
					t.Errorf("expected mount1 path '/config/test1.yaml', got '%s'", mount1.MountPath)
				}
				if mount2.MountPath != "/data/test2.json" {
					t.Errorf("expected mount2 path '/data/test2.json', got '%s'", mount2.MountPath)
				}
			},
		},
		{
			name: "file not found",
			volumes: map[string]string{
				"non-existent-uuid": "/config/test.yaml",
			},
			setupFiles:  []*corev1.ConfigMap{},
			expectError: true,
			expectCount: 0,
		},
		{
			name: "file configmap has no data",
			volumes: map[string]string{
				"550e8400-e29b-41d4-a716-446655440003": "/config/test.yaml",
			},
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440003",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "550e8400-e29b-41d4-a716-446655440003",
						},
					},
					Data: map[string]string{},
				},
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name: "complex content with special characters",
			volumes: map[string]string{
				"550e8400-e29b-41d4-a716-446655440004": "/data/script.sh",
			},
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440004",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "550e8400-e29b-41d4-a716-446655440004",
						},
					},
					Data: map[string]string{
						"script.sh": "#!/bin/bash\necho \"Hello $USER\"\nif [ -f /tmp/test ]; then\n  echo \"File exists\"\nfi",
					},
				},
			},
			expectError: false,
			expectCount: 1,
			validateMount: func(t *testing.T, mounts []krknv1alpha1.FileMount) {
				decoded, err := base64.StdEncoding.DecodeString(mounts[0].Content)
				if err != nil {
					t.Fatalf("failed to decode content: %v", err)
				}
				expected := "#!/bin/bash\necho \"Hello $USER\"\nif [ -f /tmp/test ]; then\n  echo \"File exists\"\nfi"
				if string(decoded) != expected {
					t.Errorf("expected content '%s', got '%s'", expected, string(decoded))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake client
			objects := []runtime.Object{}
			for _, file := range tt.setupFiles {
				objects = append(objects, file)
			}
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objects...).
				Build()

			// Create reconciler
			reconciler := &KrknGraphRunReconciler{
				Client: fakeClient,
			}

			// Call translateVolumesToFileMounts
			mounts, err := reconciler.translateVolumesToFileMounts(context.Background(), tt.volumes)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check count
			if len(mounts) != tt.expectCount {
				t.Errorf("expected %d mounts, got %d", tt.expectCount, len(mounts))
			}

			// Run custom validation if provided
			if tt.validateMount != nil && len(mounts) > 0 {
				tt.validateMount(t, mounts)
			}
		})
	}
}

func TestLoadFileConfigMapByID(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name        string
		fileID      string
		setupFiles  []*corev1.ConfigMap
		expectError bool
		expectName  string
	}{
		{
			name:   "file found",
			fileID: "550e8400-e29b-41d4-a716-446655440001",
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440001",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "550e8400-e29b-41d4-a716-446655440001",
						},
					},
					Data: map[string]string{
						"test.yaml": "content",
					},
				},
			},
			expectError: false,
			expectName:  "file-550e8400-e29b-41d4-a716-446655440001",
		},
		{
			name:        "file not found",
			fileID:      "non-existent-uuid",
			setupFiles:  []*corev1.ConfigMap{},
			expectError: true,
		},
		{
			name:   "multiple files with same ID - error",
			fileID: "550e8400-e29b-41d4-a716-446655440002",
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-duplicate-1",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "550e8400-e29b-41d4-a716-446655440002",
						},
					},
					Data: map[string]string{
						"test1.yaml": "content1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-duplicate-2",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "550e8400-e29b-41d4-a716-446655440002",
						},
					},
					Data: map[string]string{
						"test2.yaml": "content2",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake client
			objects := []runtime.Object{}
			for _, file := range tt.setupFiles {
				objects = append(objects, file)
			}
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objects...).
				Build()

			// Create reconciler
			reconciler := &KrknGraphRunReconciler{
				Client:    fakeClient,
				Namespace: "default",
			}

			// Call loadFileConfigMapByID
			configMap, err := reconciler.loadFileConfigMapByID(context.Background(), tt.fileID)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify name if successful
			if !tt.expectError && configMap != nil {
				if configMap.Name != tt.expectName {
					t.Errorf("expected name '%s', got '%s'", tt.expectName, configMap.Name)
				}
			}
		})
	}
}
