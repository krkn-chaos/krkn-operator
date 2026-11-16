package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// KrknAIRunPodConfig holds installation-wide limits for new run Pods.
type KrknAIRunPodConfig struct {
	OrchestratorResources   corev1.ResourceRequirements
	ResultUploaderResources corev1.ResourceRequirements
	OutputSizeLimit         resource.Quantity
	TmpSizeLimit            resource.Quantity
	UploadStateSizeLimit    resource.Quantity
}

// KrknAIRunPodConfigFromEnv reads bounded defaults and optional overrides.
func KrknAIRunPodConfigFromEnv(getenv func(string) string) (KrknAIRunPodConfig, error) {
	cfg := KrknAIRunPodConfig{
		OrchestratorResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("100m"),
				corev1.ResourceMemory:           resource.MustParse("256Mi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("500m"),
				corev1.ResourceMemory:           resource.MustParse("512Mi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
		},
		ResultUploaderResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("50m"),
				corev1.ResourceMemory:           resource.MustParse("128Mi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("250m"),
				corev1.ResourceMemory:           resource.MustParse("512Mi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("256Mi"),
			},
		},
		OutputSizeLimit:      resource.MustParse("512Mi"),
		TmpSizeLimit:         resource.MustParse("128Mi"),
		UploadStateSizeLimit: resource.MustParse("16Mi"),
	}
	if getenv == nil {
		return KrknAIRunPodConfig{}, fmt.Errorf("read Krkn-AI run configuration: getenv is nil")
	}
	for _, setting := range []struct {
		name      string
		resources *corev1.ResourceRequirements
	}{
		{"KRKNAI_ORCHESTRATOR_RESOURCES", &cfg.OrchestratorResources},
		{"KRKNAI_RESULT_UPLOADER_RESOURCES", &cfg.ResultUploaderResources},
	} {
		if value := strings.TrimSpace(getenv(setting.name)); value != "" {
			if value[0] != '{' {
				return KrknAIRunPodConfig{}, fmt.Errorf("%s must be a JSON object", setting.name)
			}
			decoder := json.NewDecoder(strings.NewReader(value))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(setting.resources); err != nil {
				return KrknAIRunPodConfig{}, fmt.Errorf("parse %s: %w", setting.name, err)
			}
			var extra any
			if err := decoder.Decode(&extra); err != io.EOF {
				return KrknAIRunPodConfig{}, fmt.Errorf("%s must contain one JSON object", setting.name)
			}
		}
		for _, name := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceEphemeralStorage} {
			request := setting.resources.Requests[name]
			limit := setting.resources.Limits[name]
			if request.Sign() <= 0 || limit.Sign() <= 0 {
				return KrknAIRunPodConfig{}, fmt.Errorf("%s requests.%s and limits.%s must be greater than zero", setting.name, name, name)
			}
			if request.Cmp(limit) > 0 {
				return KrknAIRunPodConfig{}, fmt.Errorf("%s requests.%s must not exceed limits.%s", setting.name, name, name)
			}
		}
		for _, quantities := range []corev1.ResourceList{setting.resources.Requests, setting.resources.Limits} {
			for name := range quantities {
				if name != corev1.ResourceCPU && name != corev1.ResourceMemory && name != corev1.ResourceEphemeralStorage {
					return KrknAIRunPodConfig{}, fmt.Errorf("%s has unsupported resource %q", setting.name, name)
				}
			}
		}
	}
	for _, setting := range []struct {
		name     string
		quantity *resource.Quantity
	}{
		{"KRKNAI_OUTPUT_SIZE_LIMIT", &cfg.OutputSizeLimit},
		{"KRKNAI_TMP_SIZE_LIMIT", &cfg.TmpSizeLimit},
		{"KRKNAI_UPLOAD_STATE_SIZE_LIMIT", &cfg.UploadStateSizeLimit},
	} {
		if value := getenv(setting.name); value != "" {
			quantity, err := resource.ParseQuantity(value)
			if err != nil {
				return KrknAIRunPodConfig{}, fmt.Errorf("parse %s: %w", setting.name, err)
			}
			*setting.quantity = quantity
		}
		if setting.quantity.Sign() <= 0 {
			return KrknAIRunPodConfig{}, fmt.Errorf("%s must be greater than zero", setting.name)
		}
	}
	return cfg, nil
}
