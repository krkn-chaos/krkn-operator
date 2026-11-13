package controller

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAIRunRejectsNoScenariosBeforeProvisioning(t *testing.T) {
	for _, registryName := range []string{"", "private"} {
		t.Run("registry="+registryName, func(t *testing.T) {
			r, c, run := newAIRunFixture(t)
			run.Spec.RegistryName = registryName
			if err := c.Update(context.Background(), run); err != nil {
				t.Fatal(err)
			}
			var config corev1.ConfigMap
			if err := c.Get(context.Background(), client.ObjectKey{Namespace: run.Namespace, Name: run.Spec.ConfigMapName}, &config); err != nil {
				t.Fatal(err)
			}
			config.Data["krkn-ai.yaml"] = "scenario:\n  pod-scenarios:\n    enable: false\n"
			if err := c.Update(context.Background(), &config); err != nil {
				t.Fatal(err)
			}
			err := r.ensureProvisioned(context.Background(), run)
			var permanent *configurationError
			if !errors.As(err, &permanent) {
				t.Fatalf("empty selection error = %v", err)
			}
			var pods corev1.PodList
			if err := c.List(context.Background(), &pods); err != nil {
				t.Fatal(err)
			}
			var maps corev1.ConfigMapList
			if err := c.List(context.Background(), &maps); err != nil {
				t.Fatal(err)
			}
			if len(pods.Items) != 0 || len(maps.Items) != 1 || run.Status.OrchestratorPodName != "" {
				t.Fatal("empty selection provisioned workload resources")
			}
		})
	}
}
