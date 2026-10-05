// Package crdinstaller synchronizes Kubernetes CRDs from a bundled manifest directory.
package crdinstaller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

type crdClient interface {
	Create(context.Context, *apiextensionsv1.CustomResourceDefinition, metav1.CreateOptions) (*apiextensionsv1.CustomResourceDefinition, error)
	Get(context.Context, string, metav1.GetOptions) (*apiextensionsv1.CustomResourceDefinition, error)
	Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*apiextensionsv1.CustomResourceDefinition, error)
}

// Sync creates missing CRDs and updates changed specs from the provided directory.
func Sync(ctx context.Context, config *rest.Config, directory string) ([]string, error) {
	clientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create apiextensions client: %w", err)
	}
	client := clientset.ApiextensionsV1().CustomResourceDefinitions()

	names, err := applyCRDs(ctx, client, directory)
	if err != nil {
		return nil, err
	}
	if err := waitForEstablished(ctx, client, names); err != nil {
		return nil, err
	}
	return names, nil
}

func applyCRDs(ctx context.Context, client crdClient, directory string) ([]string, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open CRD directory %q: %w", directory, err)
	}
	defer root.Close()

	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("read CRD directory %q: %w", directory, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(directory, entry.Name())
		data, err := root.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read CRD manifest %q: %w", path, err)
		}
		var desired apiextensionsv1.CustomResourceDefinition
		if err := yaml.Unmarshal(data, &desired); err != nil {
			return nil, fmt.Errorf("decode CRD manifest %q: %w", path, err)
		}
		if desired.APIVersion != "apiextensions.k8s.io/v1" || desired.Kind != "CustomResourceDefinition" || desired.Name == "" {
			return nil, fmt.Errorf("manifest %q is not a named apiextensions.k8s.io/v1 CustomResourceDefinition", path)
		}
		if err := applyCRD(ctx, client, &desired); err != nil {
			return nil, fmt.Errorf("synchronize CRD %q from %q: %w", desired.Name, path, err)
		}
		names = append(names, desired.Name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no CRD manifests found in %q", directory)
	}
	return names, nil
}

func applyCRD(ctx context.Context, client crdClient, desired *apiextensionsv1.CustomResourceDefinition) error {
	existing, err := client.Get(ctx, desired.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := client.Create(ctx, desired, metav1.CreateOptions{}); err == nil {
			return nil
		} else if !apierrors.IsAlreadyExists(err) {
			return err
		}
		existing, err = client.Get(ctx, desired.Name, metav1.GetOptions{})
	}
	if err != nil {
		return err
	}
	if apiequality.Semantic.DeepEqual(existing.Spec, desired.Spec) {
		return nil
	}

	patch, err := json.Marshal(struct {
		Spec apiextensionsv1.CustomResourceDefinitionSpec `json:"spec"`
	}{Spec: desired.Spec})
	if err != nil {
		return fmt.Errorf("encode desired CRD spec: %w", err)
	}
	_, err = client.Patch(ctx, desired.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}

func waitForEstablished(ctx context.Context, client crdClient, names []string) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		var pending []string
		for _, name := range names {
			crd, err := client.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("get CRD %q while waiting for establishment: %w", name, err)
			}
			established := false
			for _, condition := range crd.Status.Conditions {
				if condition.Type == apiextensionsv1.NamesAccepted && condition.Status == apiextensionsv1.ConditionFalse {
					return fmt.Errorf("CRD %q rejected its names: %s: %s", name, condition.Reason, condition.Message)
				}
				if condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue {
					established = true
				}
			}
			if !established {
				pending = append(pending, name)
			}
		}
		if len(pending) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for CRDs to become Established %v: %w", pending, ctx.Err())
		case <-ticker.C:
		}
	}
}
