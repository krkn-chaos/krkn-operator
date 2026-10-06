// Package crdinstaller synchronizes Kubernetes CRDs from a bundled manifest directory.
package crdinstaller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/krkn-chaos/krkn-operator/internal/crdmigration"
)

type crdClient interface {
	Create(context.Context, *apiextensionsv1.CustomResourceDefinition, metav1.CreateOptions) (*apiextensionsv1.CustomResourceDefinition, error)
	Get(context.Context, string, metav1.GetOptions) (*apiextensionsv1.CustomResourceDefinition, error)
	Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*apiextensionsv1.CustomResourceDefinition, error)
}

type deploymentClient interface {
	Get(context.Context, string, metav1.GetOptions) (*appsv1.Deployment, error)
}

type configMapClient interface {
	Get(context.Context, string, metav1.GetOptions) (*corev1.ConfigMap, error)
	Create(context.Context, *corev1.ConfigMap, metav1.CreateOptions) (*corev1.ConfigMap, error)
	Update(context.Context, *corev1.ConfigMap, metav1.UpdateOptions) (*corev1.ConfigMap, error)
	Delete(context.Context, string, metav1.DeleteOptions) error
}

const (
	operatorLabelKey   = "app.kubernetes.io/name"
	operatorLabelValue = "krkn-operator"
	guardLabelKey      = "app.kubernetes.io/component"
	guardLabelValue    = "crd-migration"
	guardDeadline      = 6 * time.Minute
)

// Stage records legacy custom-resource values before the operator deployment is upgraded.
// If the post-upgrade hook fails, retry the Helm upgrade to resume synchronization.
func Stage(ctx context.Context, config *rest.Config, namespace string, logger logr.Logger) error {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create Kubernetes client for migration guard: %w", err)
	}
	if err := ensureMigrationGuard(ctx, kubeClient.CoreV1().ConfigMaps(namespace)); err != nil {
		return err
	}
	guardClient := kubeClient.CoreV1().ConfigMaps(namespace)
	logStaleMigrationGuard(ctx, guardClient, namespace, "stageStartedAt", logger)
	if err := markMigrationStarted(ctx, guardClient, "stageStartedAt"); err != nil {
		return err
	}
	logger.Info("custom resource migration guard is active", "namespace", namespace)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create dynamic client for custom resource migration: %w", err)
	}
	if err := crdmigration.Stage(ctx, dynamicClient, namespace, logger); err != nil {
		return fmt.Errorf("stage custom resource migration before operator rollout: %w", err)
	}
	return nil
}

// Sync creates missing CRDs and updates changed specs after the new operator is ready.
// Failures leave the migration guard in place; retry the Helm upgrade to resume.
func Sync(ctx context.Context, config *rest.Config, directory, namespace, operatorDeployment string, logger logr.Logger) ([]string, error) {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client for operator rollout check: %w", err)
	}
	guardClient := kubeClient.CoreV1().ConfigMaps(namespace)
	if err := ensureMigrationGuard(ctx, guardClient); err != nil {
		return nil, err
	}
	logStaleMigrationGuard(ctx, guardClient, namespace, "syncStartedAt", logger)
	if err := markMigrationStarted(ctx, guardClient, "syncStartedAt"); err != nil {
		return nil, err
	}
	if err := waitForDeploymentReady(ctx, kubeClient.AppsV1().Deployments(namespace), operatorDeployment, logger); err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client for custom resource migration: %w", err)
	}
	if err := crdmigration.Stage(ctx, dynamicClient, namespace, logger); err != nil {
		return nil, fmt.Errorf("refresh custom resource migration snapshots after operator rollout: %w", err)
	}
	logger.Info("refreshed migration snapshots after operator rollout", "namespace", namespace)

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
	if err := crdmigration.Complete(ctx, dynamicClient, namespace, logger); err != nil {
		return nil, fmt.Errorf("complete custom resource migration after CRD synchronization: %w", err)
	}
	if err := removeMigrationGuard(ctx, kubeClient.CoreV1().ConfigMaps(namespace)); err != nil {
		return nil, err
	}
	logger.Info("custom resource migration guard removed", "namespace", namespace)
	return names, nil
}

func ensureMigrationGuard(ctx context.Context, client configMapClient) error {
	existing, err := client.Get(ctx, crdmigration.GuardConfigMapName, metav1.GetOptions{})
	if err == nil {
		if existing.Labels[operatorLabelKey] != operatorLabelValue || existing.Labels[guardLabelKey] != guardLabelValue {
			return fmt.Errorf("ConfigMap %q already exists and is not the operator CRD migration guard", crdmigration.GuardConfigMapName)
		}
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get custom resource migration guard ConfigMap %q: %w", crdmigration.GuardConfigMapName, err)
	}
	_, err = client.Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdmigration.GuardConfigMapName,
			Labels: map[string]string{
				operatorLabelKey: operatorLabelValue,
				guardLabelKey:    guardLabelValue,
			},
		},
		Data: map[string]string{"phase": "crd-migration"},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return ensureMigrationGuard(ctx, client)
	}
	if err != nil {
		return fmt.Errorf("create custom resource migration guard ConfigMap %q: %w", crdmigration.GuardConfigMapName, err)
	}
	return nil
}

func logStaleMigrationGuard(ctx context.Context, client configMapClient, namespace, timestampKey string, logger logr.Logger) {
	guard, err := client.Get(ctx, crdmigration.GuardConfigMapName, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "unable to inspect custom resource migration guard age", "namespace", namespace)
		return
	}
	value := guard.Data[timestampKey]
	if value == "" {
		return
	}
	startedAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		logger.Error(err, "custom resource migration guard has an invalid start time; retry the Helm upgrade if migration is incomplete", "namespace", namespace, "timestampKey", timestampKey)
		return
	}
	if age := time.Since(startedAt); age > guardDeadline {
		logger.Error(errors.New("migration guard is older than hook deadline"), "custom resource migration guard remains active; retry the Helm upgrade to resume synchronization", "namespace", namespace, "age", age.Round(time.Second).String(), "hookDeadline", guardDeadline.String())
	}
}

func markMigrationStarted(ctx context.Context, client configMapClient, timestampKey string) error {
	guard, err := client.Get(ctx, crdmigration.GuardConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get custom resource migration guard before %s: %w", timestampKey, err)
	}
	if guard.Labels[operatorLabelKey] != operatorLabelValue || guard.Labels[guardLabelKey] != guardLabelValue {
		return fmt.Errorf("ConfigMap %q is not the operator CRD migration guard", crdmigration.GuardConfigMapName)
	}
	if guard.Data == nil {
		guard.Data = map[string]string{}
	}
	guard.Data[timestampKey] = time.Now().UTC().Format(time.RFC3339)
	if _, err := client.Update(ctx, guard, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("record %s in migration guard: %w", timestampKey, err)
	}
	return nil
}

func removeMigrationGuard(ctx context.Context, client configMapClient) error {
	existing, err := client.Get(ctx, crdmigration.GuardConfigMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get custom resource migration guard ConfigMap %q before removal: %w", crdmigration.GuardConfigMapName, err)
	}
	if existing.Labels[operatorLabelKey] != operatorLabelValue || existing.Labels[guardLabelKey] != guardLabelValue {
		return fmt.Errorf("ConfigMap %q is not the operator CRD migration guard; refusing to remove it", crdmigration.GuardConfigMapName)
	}
	if err := client.Delete(ctx, crdmigration.GuardConfigMapName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("remove custom resource migration guard ConfigMap %q: %w", crdmigration.GuardConfigMapName, err)
	}
	return nil
}

func waitForDeploymentReady(ctx context.Context, client deploymentClient, name string, logger logr.Logger) error {
	if name == "" {
		return fmt.Errorf("operator deployment name is required before synchronizing CRDs")
	}
	logger.Info("waiting for the operator rollout before synchronizing CRDs", "deployment", name)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		deployment, err := client.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get operator Deployment %q while waiting for rollout: %w", name, err)
		}
		desired := int32(1)
		if deployment.Spec.Replicas != nil {
			desired = *deployment.Spec.Replicas
		}
		if deployment.Status.ObservedGeneration >= deployment.Generation &&
			deployment.Status.Replicas == desired &&
			deployment.Status.UpdatedReplicas == desired &&
			deployment.Status.AvailableReplicas == desired &&
			deployment.Status.UnavailableReplicas == 0 {
			logger.Info("operator rollout is complete", "deployment", name, "replicas", desired)
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for operator Deployment %q rollout: %w", name, ctx.Err())
		case <-ticker.C:
		}
	}
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
		if desired.Labels == nil {
			desired.Labels = make(map[string]string, 1)
		}
		desired.Labels[operatorLabelKey] = operatorLabelValue
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
	specChanged := !apiequality.Semantic.DeepEqual(existing.Spec, desired.Spec)
	labelChanged := existing.Labels[operatorLabelKey] != operatorLabelValue
	if !specChanged && !labelChanged {
		return nil
	}

	patchObject := map[string]any{
		"metadata": map[string]any{
			"labels": map[string]string{operatorLabelKey: operatorLabelValue},
		},
	}
	if specChanged {
		patchObject["spec"] = desired.Spec
	}
	patch, err := json.Marshal(patchObject)
	if err != nil {
		return fmt.Errorf("encode CRD patch: %w", err)
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
