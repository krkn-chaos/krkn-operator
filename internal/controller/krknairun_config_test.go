package controller

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestKrknAIRunPodConfigPreservesBoundsWithPartialOverrides(t *testing.T) {
	values := map[string]string{
		"KRKNAI_ORCHESTRATOR_RESOURCES":    `{"requests":{"memory":"192Mi"},"limits":{"memory":"384Mi"}}`,
		"KRKNAI_RESULT_UPLOADER_RESOURCES": `{}`,
		"KRKNAI_OUTPUT_SIZE_LIMIT":         "64Mi",
		"KRKNAI_TMP_SIZE_LIMIT":            "32Mi",
		"KRKNAI_UPLOAD_STATE_SIZE_LIMIT":   "8Mi",
	}
	cfg, err := KrknAIRunPodConfigFromEnv(func(name string) string { return values[name] })
	if err != nil {
		t.Fatal(err)
	}
	for _, resources := range []corev1.ResourceRequirements{cfg.OrchestratorResources, cfg.ResultUploaderResources} {
		for _, name := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceEphemeralStorage} {
			request, limit := resources.Requests[name], resources.Limits[name]
			if request.Sign() <= 0 || limit.Sign() <= 0 || request.Cmp(limit) > 0 {
				t.Fatalf("partial overrides left %s unbounded or invalid: %+v", name, resources)
			}
		}
	}
	if cfg.OrchestratorResources.Requests.Memory().Cmp(resource.MustParse("192Mi")) != 0 ||
		cfg.OrchestratorResources.Limits.Memory().Cmp(resource.MustParse("384Mi")) != 0 {
		t.Fatalf("memory override was not applied: %+v", cfg.OrchestratorResources)
	}
	for _, check := range []struct {
		got  resource.Quantity
		want string
	}{
		{cfg.OutputSizeLimit, "64Mi"},
		{cfg.TmpSizeLimit, "32Mi"},
		{cfg.UploadStateSizeLimit, "8Mi"},
	} {
		if check.got.Cmp(resource.MustParse(check.want)) != 0 {
			t.Fatalf("size limit = %s, want %s", check.got.String(), check.want)
		}
	}
}

func TestKrknAIRunPodConfigRejectsInvalidResources(t *testing.T) {
	for _, name := range []string{"KRKNAI_ORCHESTRATOR_RESOURCES", "KRKNAI_RESULT_UPLOADER_RESOURCES"} {
		for _, value := range []string{
			`null`, `[]`, `{} {}`, `{"unknown":{}}`,
			`{"requests":null}`, `{"limits":null}`,
			`{"requests":{"cpu":"-1"}}`,
			`{"limits":{"memory":"0"}}`,
			`{"limits":{"ephemeral-storage":"0"}}`,
			`{"requests":{"cpu":"2"},"limits":{"cpu":"1"}}`,
			`{"limits":{"memory":"not-a-quantity"}}`,
			`{"limits":{"memroy":"512Mi"}}`,
		} {
			t.Run(name+"/"+value, func(t *testing.T) {
				_, err := KrknAIRunPodConfigFromEnv(func(key string) string {
					if key == name {
						return value
					}
					return ""
				})
				if err == nil || !strings.Contains(err.Error(), name) {
					t.Fatalf("invalid resource setting must identify %s: %v", name, err)
				}
			})
		}
	}
}

func TestKrknAIRunPodConfigRejectsInvalidVolumeBounds(t *testing.T) {
	for _, name := range []string{"KRKNAI_OUTPUT_SIZE_LIMIT", "KRKNAI_TMP_SIZE_LIMIT", "KRKNAI_UPLOAD_STATE_SIZE_LIMIT"} {
		for _, value := range []string{"0", "-1Gi", "not-a-quantity"} {
			t.Run(name+"/"+value, func(t *testing.T) {
				_, err := KrknAIRunPodConfigFromEnv(func(key string) string {
					if key == name {
						return value
					}
					return ""
				})
				if err == nil || !strings.Contains(err.Error(), name) {
					t.Fatalf("invalid volume bound must identify %s: %v", name, err)
				}
			})
		}
	}
}
