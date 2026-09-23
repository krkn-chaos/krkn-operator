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

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/cloudcreds"
)

// TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef verifies the
// end-to-end secrecy contract without a live cluster:
//
//  1. Console-style (or malicious) run payload includes cloudCredentialRef AND
//     plaintext cloud env vars.
//  2. PostScenarioRun persists CloudCredentialRef on the CRD and strips cloud
//     secrets from Spec.Environment.
//  3. The same InjectCredentials call the controller uses produces Pod env vars
//     via SecretKeyRef (never plaintext values from the request).
func TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(t *testing.T) {
	const (
		targetRequestID = "contract-target"
		clusterName     = "contract-cluster"
		credName        = "aws-contract"
		namespace       = "default"
	)
	kubeconfig := "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd"

	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	managedClustersJSON, _ := json.Marshal(map[string]map[string]map[string]string{
		"krkn-operator-acm": {
			clusterName: {"kubeconfig": kubeconfig},
		},
	})

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: targetRequestID, Namespace: namespace},
		Data:       map[string][]byte{"managed-clusters": managedClustersJSON},
	}
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: targetRequestID, Namespace: namespace},
		Spec:       krknv1alpha1.KrknTargetRequestSpec{UUID: "contract-uuid"},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"krkn-operator": {{
					ClusterName:   clusterName,
					ClusterAPIURL: "https://" + clusterName + ".example.com:6443",
				}},
			},
		},
	}

	awsCred := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credName,
			Namespace: namespace,
			Labels:    cloudcreds.BuildLabels(cloudcreds.ProviderAWS, nil, true),
		},
		Data: map[string][]byte{
			cloudcreds.SecretKeyAWSAccessKeyID:     []byte("AKIACONTRACT"),
			cloudcreds.SecretKeyAWSSecretAccessKey: []byte("from-secret-not-request"),
			cloudcreds.SecretKeyAWSDefaultRegion:   []byte("us-east-1"),
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(targetSecret, targetRequest, awsCred).
		Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), namespace, "localhost:50051")

	// Payload mirrors a console run that selected a saved credential, plus a
	// direct-API attempt to also smuggle plaintext cloud secrets.
	reqBody := map[string]interface{}{
		"targetRequestID": targetRequestID,
		"targetClusters": map[string][]string{
			"krkn-operator": {clusterName},
		},
		"scenario": map[string]interface{}{
			"name":    "node-scenarios",
			"private": false,
		},
		"cloudCredentialRef": credName,
		"environment": map[string]string{
			"TIMEOUT":               "300",
			"AWS_SECRET_ACCESS_KEY": "plaintext-should-be-stripped",
			"AWS_ACCESS_KEY_ID":     "plaintext-aki",
			"CLOUD_TYPE":            "aws",
			"NODE_NAME":             "worker-1",
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, ScenariosRunPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createResp ScenarioRunCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if createResp.ScenarioRunName == "" {
		t.Fatal("expected ScenarioRunName in response")
	}

	// --- Step 2: CRD must reference the credential and omit cloud secrets ---
	var scenarioRun krknv1alpha1.KrknScenarioRun
	if err := fakeClient.Get(req.Context(), k8stypes.NamespacedName{
		Name:      createResp.ScenarioRunName,
		Namespace: namespace,
	}, &scenarioRun); err != nil {
		t.Fatalf("get scenario run CR: %v", err)
	}

	if scenarioRun.Spec.CloudCredentialRef != credName {
		t.Fatalf("CloudCredentialRef = %q, want %q", scenarioRun.Spec.CloudCredentialRef, credName)
	}

	for _, forbidden := range []string{
		"AWS_SECRET_ACCESS_KEY",
		"AWS_ACCESS_KEY_ID",
		"AWS_DEFAULT_REGION",
		"CLOUD_TYPE",
	} {
		if _, ok := scenarioRun.Spec.Environment[forbidden]; ok {
			t.Errorf("CRD Environment must not contain stripped cloud key %q (got %q)",
				forbidden, scenarioRun.Spec.Environment[forbidden])
		}
	}
	if scenarioRun.Spec.Environment["TIMEOUT"] != "300" {
		t.Errorf("non-cloud env TIMEOUT = %q, want 300", scenarioRun.Spec.Environment["TIMEOUT"])
	}
	if scenarioRun.Spec.Environment["NODE_NAME"] != "worker-1" {
		t.Errorf("non-cloud env NODE_NAME = %q, want worker-1", scenarioRun.Spec.Environment["NODE_NAME"])
	}

	// --- Step 3: Controller injection path → Pod env uses SecretKeyRef ---
	// Mirrors prepareJobResources: start from CR Environment, then append
	// InjectCredentials output (same helper the reconciler calls).
	podEnv := make([]corev1.EnvVar, 0, len(scenarioRun.Spec.Environment)+8)
	for k, v := range scenarioRun.Spec.Environment {
		podEnv = append(podEnv, corev1.EnvVar{Name: k, Value: v})
	}

	var credSecret corev1.Secret
	if err := fakeClient.Get(req.Context(), k8stypes.NamespacedName{
		Name:      scenarioRun.Spec.CloudCredentialRef,
		Namespace: namespace,
	}, &credSecret); err != nil {
		t.Fatalf("load credential secret for injection: %v", err)
	}

	credEnvVars, _, _, err := cloudcreds.InjectCredentials(&credSecret)
	if err != nil {
		t.Fatalf("InjectCredentials: %v", err)
	}
	podEnv = append(podEnv, credEnvVars...)

	byName := map[string]corev1.EnvVar{}
	for _, env := range podEnv {
		byName[env.Name] = env
	}

	if env, ok := byName["TIMEOUT"]; !ok || env.Value != "300" || env.ValueFrom != nil {
		t.Errorf("TIMEOUT should remain a plaintext non-cloud value, got %+v", env)
	}

	secretRefKeys := map[string]string{
		"AWS_ACCESS_KEY_ID":     cloudcreds.SecretKeyAWSAccessKeyID,
		"AWS_SECRET_ACCESS_KEY": cloudcreds.SecretKeyAWSSecretAccessKey,
		"AWS_DEFAULT_REGION":    cloudcreds.SecretKeyAWSDefaultRegion,
	}
	for name, wantKey := range secretRefKeys {
		env, ok := byName[name]
		if !ok {
			t.Fatalf("missing injected env %s", name)
		}
		if env.Value != "" {
			t.Errorf("%s must not be plaintext on the Pod; Value=%q", name, env.Value)
		}
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			t.Fatalf("%s must use SecretKeyRef", name)
		}
		if env.ValueFrom.SecretKeyRef.Name != credName {
			t.Errorf("%s SecretKeyRef.Name = %q, want %q", name, env.ValueFrom.SecretKeyRef.Name, credName)
		}
		if env.ValueFrom.SecretKeyRef.Key != wantKey {
			t.Errorf("%s SecretKeyRef.Key = %q, want %q", name, env.ValueFrom.SecretKeyRef.Key, wantKey)
		}
	}

	if env, ok := byName["CLOUD_TYPE"]; !ok || env.Value != "aws" {
		t.Errorf("CLOUD_TYPE = %+v, want Value=aws from injection", byName["CLOUD_TYPE"])
	}
}

// TestCloudCredentialContract_GraphRunStripsNodeEnv verifies graph nodes that
// use cloudCredentialRef also have cloud env vars stripped before CR creation.
func TestCloudCredentialContract_GraphRunStripsNodeEnv(t *testing.T) {
	const (
		credName        = "aws-graph-contract"
		namespace       = "default"
		targetRequestID = "graph-contract-target"
		clusterName     = "graph-cluster"
	)

	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	awsCred := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credName,
			Namespace: namespace,
			Labels:    cloudcreds.BuildLabels(cloudcreds.ProviderAWS, nil, true),
		},
		Data: map[string][]byte{
			cloudcreds.SecretKeyAWSAccessKeyID:     []byte("AKIAGRAPH"),
			cloudcreds.SecretKeyAWSSecretAccessKey: []byte("graph-secret"),
			cloudcreds.SecretKeyAWSDefaultRegion:   []byte("us-west-2"),
		},
	}
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: targetRequestID, Namespace: namespace},
		Spec:       krknv1alpha1.KrknTargetRequestSpec{UUID: "graph-uuid"},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"krkn-operator": {{
					ClusterName:   clusterName,
					ClusterAPIURL: "https://" + clusterName + ".example.com:6443",
				}},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(awsCred, targetRequest).
		Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), namespace, "localhost:50051")

	falseVal := false
	reqBody := GraphRunCreateRequest{
		TargetRequestID:    targetRequestID,
		TargetClusters:     map[string][]string{"krkn-operator": {clusterName}},
		CloudCredentialRef: credName,
		Graph: map[string]krknv1alpha1.GraphScenarioNode{
			"n1": {
				Scenario: krknv1alpha1.ScenarioReference{
					Name:    "node-scenarios",
					Private: &falseVal,
				},
				Env: map[string]string{
					"TIMEOUT":               "60",
					"AWS_SECRET_ACCESS_KEY": "plaintext-graph-secret",
					"CLOUD_TYPE":            "aws",
				},
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, GraphRunsPath, bytes.NewReader(body))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()
	handler.CreateGraphRun(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("expected success creating graph run, got %d: %s", w.Code, w.Body.String())
	}

	var list krknv1alpha1.KrknGraphRunList
	if err := fakeClient.List(req.Context(), &list); err != nil {
		t.Fatalf("list graph runs: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 graph run, got %d", len(list.Items))
	}
	graphRun := list.Items[0]
	if graphRun.Spec.CloudCredentialRef != credName {
		t.Fatalf("graph CloudCredentialRef = %q, want %q", graphRun.Spec.CloudCredentialRef, credName)
	}
	node := graphRun.Spec.Graph["n1"]
	if _, ok := node.Env["AWS_SECRET_ACCESS_KEY"]; ok {
		t.Errorf("graph node Env still has AWS_SECRET_ACCESS_KEY=%q", node.Env["AWS_SECRET_ACCESS_KEY"])
	}
	if _, ok := node.Env["CLOUD_TYPE"]; ok {
		t.Errorf("graph node Env still has CLOUD_TYPE=%q", node.Env["CLOUD_TYPE"])
	}
	if node.Env["TIMEOUT"] != "60" {
		t.Errorf("TIMEOUT = %q, want 60", node.Env["TIMEOUT"])
	}
}
