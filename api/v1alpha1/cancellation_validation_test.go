package v1alpha1

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	celvalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"sigs.k8s.io/yaml"
)

func TestRunCancellationValidation(t *testing.T) {
	requested, notRequested := true, false
	tests := []struct {
		name       string
		create     bool
		oldFlag    *bool
		newFlag    *bool
		removeSpec bool
		update     bool
		wantErr    bool
	}{
		{name: "create without cancellation", create: true},
		{name: "create already cancelled", create: true, newFlag: &requested},
		{name: "omitted flag can request cancellation", newFlag: &requested},
		{name: "false flag can request cancellation", oldFlag: &notRequested, newFlag: &requested},
		{name: "false flag can be omitted", oldFlag: &notRequested},
		{name: "repeated cancellation is idempotent", oldFlag: &requested, newFlag: &requested},
		{name: "cancellation cannot become false", oldFlag: &requested, newFlag: &notRequested, wantErr: true},
		{name: "cancellation field cannot be removed", oldFlag: &requested, wantErr: true},
		{name: "cancelled spec cannot be removed", oldFlag: &requested, removeSpec: true, wantErr: true},
		{name: "other updates preserve cancellation", oldFlag: &requested, newFlag: &requested, update: true},
	}

	for _, file := range []string{
		"krkn.krkn-chaos.dev_krknairuns.yaml",
		"krkn.krkn-chaos.dev_krknscenarioruns.yaml",
	} {
		content, err := os.ReadFile(filepath.Join("..", "..", "config", "crd", "bases", file))
		if err != nil {
			t.Fatal(err)
		}
		var crd apiextensionsv1.CustomResourceDefinition
		if err := yaml.Unmarshal(content, &crd); err != nil {
			t.Fatal(err)
		}
		for _, version := range crd.Spec.Versions {
			if !version.Served {
				continue
			}
			t.Run(crd.Spec.Names.Kind+"/"+version.Name, func(t *testing.T) {
				if version.Schema == nil || version.Schema.OpenAPIV3Schema == nil {
					t.Fatal("served CRD version has no structural schema")
				}
				var properties apiextensions.JSONSchemaProps
				if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(version.Schema.OpenAPIV3Schema, &properties, nil); err != nil {
					t.Fatal(err)
				}
				structural, err := structuralschema.NewStructural(&properties)
				if err != nil {
					t.Fatal(err)
				}
				validator := celvalidation.NewValidator(structural, true, celconfig.PerCallLimit)
				if validator == nil {
					t.Fatal("served CRD version has no CEL validator")
				}
				for _, test := range tests {
					t.Run(test.name, func(t *testing.T) {
						var previous any
						if !test.create {
							previous = cancellationValidationObject(crd.Spec.Group, version.Name, crd.Spec.Names.Kind, test.oldFlag)
						}
						next := cancellationValidationObject(crd.Spec.Group, version.Name, crd.Spec.Names.Kind, test.newFlag)
						if test.removeSpec {
							delete(next, "spec")
						}
						if test.update {
							next["spec"].(map[string]any)["ownerUserId"] = "updated-owner"
							next["status"] = map[string]any{"phase": "Cancelled"}
						}
						errors, _ := validator.Validate(context.Background(), field.NewPath("resource"), structural, next, previous, celconfig.RuntimeCELCostBudget)
						if (len(errors) != 0) != test.wantErr {
							t.Fatalf("validation errors = %v, want rejection = %v", errors, test.wantErr)
						}
					})
				}
			})
		}
	}
}

func cancellationValidationObject(group, version, kind string, requested *bool) map[string]any {
	spec := map[string]any{
		"targetRequestId": "target-request",
		"targetClusters":  map[string]any{"provider": []any{"cluster"}},
	}
	if kind == "KrknAIRun" {
		spec["configMapName"] = "run-config"
	} else {
		spec["scenario"] = map[string]any{"name": "pod-scenarios", "private": false}
	}
	if requested != nil {
		spec["cancelRequested"] = *requested
	}
	return map[string]any{
		"apiVersion": group + "/" + version,
		"kind":       kind,
		"metadata":   map[string]any{"name": "cancellation-validation"},
		"spec":       spec,
	}
}
