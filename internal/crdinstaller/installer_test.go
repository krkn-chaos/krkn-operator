package crdinstaller

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(t *testing.T) {
	directory := t.TempDir()
	updated := testCRD("widgets", "Widget")
	updated.Spec.Names.ShortNames = []string{"widget"}
	writeCRD(t, directory, "widgets.yaml", updated)
	created := testCRD("gadgets", "Gadget")
	writeCRD(t, directory, "gadgets.yaml", created)

	existing := updated.DeepCopy()
	existing.Spec.Names.ShortNames = []string{"old"}
	clientset := fake.NewSimpleClientset(existing)
	client := clientset.ApiextensionsV1().CustomResourceDefinitions()

	names, err := applyCRDs(context.Background(), client, directory)
	if err != nil {
		t.Fatalf("apply CRDs: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 synchronized CRDs, got %v", names)
	}

	gotUpdated, err := client.Get(context.Background(), updated.Name, v1.GetOptions{})
	if err != nil {
		t.Fatalf("get updated CRD: %v", err)
	}
	if len(gotUpdated.Spec.Names.ShortNames) != 1 || gotUpdated.Spec.Names.ShortNames[0] != "widget" {
		t.Fatalf("updated CRD short names = %v, want [widget]", gotUpdated.Spec.Names.ShortNames)
	}
	if _, err := client.Get(context.Background(), created.Name, v1.GetOptions{}); err != nil {
		t.Fatalf("get newly created CRD: %v", err)
	}

	var creates, patches int
	for _, action := range clientset.Actions() {
		switch action.GetVerb() {
		case "create":
			creates++
		case "patch":
			patches++
		}
	}
	if creates != 1 || patches != 1 {
		t.Fatalf("got %d creates and %d patches, want 1 of each", creates, patches)
	}
}

func TestApplyCRDsDoesNotPatchUnchangedSpecs(t *testing.T) {
	directory := t.TempDir()
	desired := testCRD("widgets", "Widget")
	writeCRD(t, directory, "widgets.yaml", desired)

	clientset := fake.NewSimpleClientset(desired)
	client := clientset.ApiextensionsV1().CustomResourceDefinitions()
	if _, err := applyCRDs(context.Background(), client, directory); err != nil {
		t.Fatalf("apply CRDs: %v", err)
	}
	for _, action := range clientset.Actions() {
		if action.GetVerb() == "patch" || action.GetVerb() == "create" {
			t.Fatalf("unchanged CRD caused %s action", action.GetVerb())
		}
	}
}

func TestApplyCRDsRejectsNonCRDManifest(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "invalid.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: not-a-crd\n"), 0600); err != nil {
		t.Fatalf("write invalid manifest: %v", err)
	}

	clientset := fake.NewSimpleClientset()
	client := clientset.ApiextensionsV1().CustomResourceDefinitions()
	if _, err := applyCRDs(context.Background(), client, directory); err == nil {
		t.Fatal("expected invalid manifest to be rejected")
	}
}

func TestWaitForEstablished(t *testing.T) {
	crd := testCRD("widgets", "Widget")
	crd.Status.Conditions = []apiextensionsv1.CustomResourceDefinitionCondition{{
		Type:   apiextensionsv1.Established,
		Status: apiextensionsv1.ConditionTrue,
	}}
	clientset := fake.NewSimpleClientset(crd)
	client := clientset.ApiextensionsV1().CustomResourceDefinitions()

	if err := waitForEstablished(context.Background(), client, []string{crd.Name}); err != nil {
		t.Fatalf("wait for CRD establishment: %v", err)
	}
}

func TestWaitForEstablishedReturnsRejectedNames(t *testing.T) {
	crd := testCRD("widgets", "Widget")
	crd.Status.Conditions = []apiextensionsv1.CustomResourceDefinitionCondition{{
		Type:    apiextensionsv1.NamesAccepted,
		Status:  apiextensionsv1.ConditionFalse,
		Reason:  "Conflict",
		Message: "plural name is already claimed",
	}}
	clientset := fake.NewSimpleClientset(crd)
	client := clientset.ApiextensionsV1().CustomResourceDefinitions()

	if err := waitForEstablished(context.Background(), client, []string{crd.Name}); err == nil {
		t.Fatal("expected rejected CRD names to return an error")
	}
}

func testCRD(plural, kind string) *apiextensionsv1.CustomResourceDefinition {
	singular := plural[:len(plural)-1]
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: v1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: v1.ObjectMeta{Name: plural + ".example.dev"},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.dev",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   plural,
				Singular: singular,
				Kind:     kind,
				ListKind: kind + "List",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    "v1",
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"},
				},
			}},
		},
	}
}

func writeCRD(t *testing.T, directory, name string, crd *apiextensionsv1.CustomResourceDefinition) {
	t.Helper()
	data, err := yaml.Marshal(crd)
	if err != nil {
		t.Fatalf("marshal CRD fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
		t.Fatalf("write CRD fixture: %v", err)
	}
}
