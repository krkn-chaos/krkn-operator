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

// Package visualize provides Grafana deployment functionality for krkn-visualize
package visualize

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Upstream resource names created by the krkn-visualize deploy image on the
// target cluster. The operator patches these to enable dashboard embedding.
const (
	// UpstreamGrafanaConfigMap is the ConfigMap holding the deployed Grafana's grafana.ini.
	UpstreamGrafanaConfigMap = "sc-grafana-config"
	// UpstreamGrafanaDeployment is the Deployment running the deployed Grafana.
	UpstreamGrafanaDeployment = "krkn-visualize"
	// GrafanaIniKey is the ConfigMap key holding grafana.ini contents.
	GrafanaIniKey = "grafana.ini"
)

// GrafanaDeployer handles Grafana deployment to target clusters
type GrafanaDeployer struct {
	client    client.Client
	namespace string
}

// NewGrafanaDeployer creates a new Grafana deployer
func NewGrafanaDeployer(c client.Client, namespace string) *GrafanaDeployer {
	return &GrafanaDeployer{
		client:    c,
		namespace: namespace,
	}
}

// DeployGrafana deploys Grafana with configured datasources
func (d *GrafanaDeployer) DeployGrafana(ctx context.Context, config DeploymentConfig) (string, error) {
	logger := log.FromContext(ctx).WithName("grafana-deployer")
	logger.Info("Starting Grafana deployment",
		"instance", config.InstanceName,
		"namespace", config.Namespace,
		"targetCluster", config.TargetCluster,
	)

	// Step 1: Create namespace if it doesn't exist
	if err := d.ensureNamespace(ctx, config.Namespace, config.InstanceName); err != nil {
		return "", fmt.Errorf("failed to create namespace: %w", err)
	}

	// Step 2: Create Secret with datasources and credentials
	if err := d.createDatasourcesSecret(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create datasources Secret: %w", err)
	}

	// Step 3: Create Secret with Grafana admin password
	if err := d.createGrafanaSecret(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create Grafana secret: %w", err)
	}

	// Step 4: Create Grafana Deployment
	if err := d.createGrafanaDeployment(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create Grafana deployment: %w", err)
	}

	// Step 5: Create Grafana Service
	if err := d.createGrafanaService(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create Grafana service: %w", err)
	}

	// Step 6: Deploy the image renderer as a separate service used by Grafana.
	if err := d.createGrafanaRenderer(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create Grafana renderer: %w", err)
	}

	// Step 7: Create OpenShift Route for external access
	if err := CreateGrafanaRoute(ctx, d.client, config.Namespace, config.InstanceName); err != nil {
		return "", fmt.Errorf("failed to create Grafana route: %w", err)
	}

	logger.Info("Route created successfully, waiting for host assignment...")
	grafanaURL, err := GetRouteURL(ctx, d.client, config.Namespace, ResourceName(config.InstanceName, "grafana"))
	if err != nil {
		return "", fmt.Errorf("failed to get Grafana route host: %w", err)
	}
	logger.Info("Route URL retrieved successfully", "url", grafanaURL)

	logger.Info("Grafana deployment completed successfully",
		"instance", config.InstanceName,
		"namespace", config.Namespace,
		"url", grafanaURL,
	)

	return grafanaURL, nil
}

// ensureNamespace creates the target namespace if it doesn't exist
func (d *GrafanaDeployer) ensureNamespace(ctx context.Context, namespace, instanceName string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      instanceName,
			},
		},
	}

	err := d.client.Create(ctx, ns)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createDatasourcesSecret creates a Secret with Elasticsearch and Prometheus datasources.
func (d *GrafanaDeployer) createDatasourcesSecret(ctx context.Context, config DeploymentConfig) error {
	datasourcesYAML := "apiVersion: 1\ndatasources:\n"

	// Add Elasticsearch datasource if configured
	if config.ElasticsearchURL != "" {
		datasourcesYAML += fmt.Sprintf(`  - name: Elasticsearch
    type: elasticsearch
    access: proxy
    url: %s
    database: "krkn-*"
    isDefault: true
    jsonData:
      esVersion: "7.10.0"
      timeField: "@timestamp"
      logMessageField: "message"
      logLevelField: "level"
`, config.ElasticsearchURL)

		// Add basic auth if credentials provided
		if config.ElasticsearchUsername != "" {
			datasourcesYAML += "    basicAuth: true\n"
			datasourcesYAML += fmt.Sprintf("    basicAuthUser: %s\n", config.ElasticsearchUsername)
			datasourcesYAML += "    secureJsonData:\n"
			datasourcesYAML += fmt.Sprintf("      basicAuthPassword: %s\n", config.ElasticsearchPassword)
		}
	}

	// Add Prometheus datasource if configured
	if config.PrometheusURL != "" {
		datasourcesYAML += fmt.Sprintf(`  - name: Prometheus
    type: prometheus
    access: proxy
    url: %s
`, config.PrometheusURL)

		if config.PrometheusBearerToken != "" {
			datasourcesYAML += "    jsonData:\n"
			datasourcesYAML += "      httpHeaderName1: \"Authorization\"\n"
			datasourcesYAML += "    secureJsonData:\n"
			datasourcesYAML += fmt.Sprintf("      httpHeaderValue1: \"Bearer %s\"\n", config.PrometheusBearerToken)
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceName(config.InstanceName, "grafana-datasources"),
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"datasources.yaml": []byte(datasourcesYAML),
		},
	}

	err := d.client.Create(ctx, secret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createGrafanaSecret creates Secret with Grafana admin credentials
func (d *GrafanaDeployer) createGrafanaSecret(ctx context.Context, config DeploymentConfig) error {
	rendererTokenBytes := make([]byte, 32)
	if _, err := rand.Read(rendererTokenBytes); err != nil {
		return fmt.Errorf("failed to generate Grafana renderer token: %w", err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceName(config.InstanceName, "grafana-admin"),
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"admin-user":     "admin",
			"admin-password": config.GrafanaPassword,
			"renderer-token": hex.EncodeToString(rendererTokenBytes),
		},
	}

	err := d.client.Create(ctx, secret)
	if err == nil {
		return nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return err
	}

	// Add the token to existing installations too. Updating the deployment
	// templates below then rolls both Grafana and the renderer onto this token.
	existing := &corev1.Secret{}
	if err := d.client.Get(ctx, client.ObjectKeyFromObject(secret), existing); err != nil {
		return fmt.Errorf("failed to get existing Grafana secret: %w", err)
	}
	if len(existing.Data["renderer-token"]) == 0 {
		if existing.Data == nil {
			existing.Data = make(map[string][]byte)
		}
		existing.Data["renderer-token"] = []byte(secret.StringData["renderer-token"])
		if err := d.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to add renderer token to existing Grafana secret: %w", err)
		}
	}
	return nil
}

func (d *GrafanaDeployer) createOrUpdateDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	if err := d.client.Create(ctx, deployment); err == nil {
		return nil
	} else if !apierrors.IsAlreadyExists(err) {
		return err
	}

	existing := &appsv1.Deployment{}
	key := client.ObjectKeyFromObject(deployment)
	if err := d.client.Get(ctx, key, existing); err != nil {
		return fmt.Errorf("failed to get existing deployment %s: %w", deployment.Name, err)
	}
	existing.Spec.Template = deployment.Spec.Template
	if err := d.client.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update deployment %s: %w", deployment.Name, err)
	}
	return nil
}

// createGrafanaDeployment creates the Grafana Deployment
func (d *GrafanaDeployer) createGrafanaDeployment(ctx context.Context, config DeploymentConfig) error {
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceName(config.InstanceName, "grafana"),
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
				"app":          ResourceName(config.InstanceName, "grafana"),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": ResourceName(config.InstanceName, "grafana"),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":          ResourceName(config.InstanceName, "grafana"),
						LabelManagedBy: ManagedByValue,
						LabelComponent: ComponentValue,
						LabelName:      config.InstanceName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "grafana",
							Image: "grafana/grafana:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 3000,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "GF_RENDERING_SERVER_URL",
									Value: fmt.Sprintf("http://%s:8081/render", ResourceName(config.InstanceName, "grafana-renderer")),
								},
								{
									Name:  "GF_RENDERING_CALLBACK_URL",
									Value: fmt.Sprintf("http://%s:3000/", ResourceName(config.InstanceName, "grafana")),
								},
								{
									Name: "GF_SECURITY_ADMIN_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: ResourceName(config.InstanceName, "grafana-admin"),
											},
											Key: "admin-user",
										},
									},
								},
								{
									Name: "GF_SECURITY_ADMIN_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: ResourceName(config.InstanceName, "grafana-admin"),
											},
											Key: "admin-password",
										},
									},
								},
								{
									Name: "GF_RENDERING_RENDERER_TOKEN",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: ResourceName(config.InstanceName, "grafana-admin"),
											},
											Key: "renderer-token",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datasources",
									MountPath: "/etc/grafana/provisioning/datasources",
								},
								{
									Name:      "grafana-storage",
									MountPath: "/var/lib/grafana",
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/api/health",
										Port: intstr.FromInt(3000),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/api/health",
										Port: intstr.FromInt(3000),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       5,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
							},
						},
						{
							Name:  "dashboard-syncer",
							Image: "quay.io/krkn-chaos/visualize-syncer:opensearch-latest",
							Env: []corev1.EnvVar{
								{
									Name: "GRAFANA_ADMIN_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: ResourceName(config.InstanceName, "grafana-admin")},
											Key:                  "admin-password",
										},
									},
								},
								{
									Name: "GRAFANA_URL",
									// #nosec G101 -- $(GRAFANA_ADMIN_PASSWORD) is a Kubernetes env
									// var reference resolved from the admin Secret at pod start,
									// not a hardcoded credential.
									Value: "http://admin:$(GRAFANA_ADMIN_PASSWORD)@localhost:3000",
								},
								{
									Name:  "INPUT_DIRS",
									Value: "/visualize/rendered/Chaos:/visualize/rendered/k8s",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "datasources",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: ResourceName(config.InstanceName, "grafana-datasources"),
								},
							},
						},
						{
							Name: "grafana-storage",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	if err := d.createOrUpdateDeployment(ctx, deployment); err != nil {
		return err
	}

	return nil
}

// createGrafanaRenderer deploys Grafana's image renderer as a separate pod.
func (d *GrafanaDeployer) createGrafanaRenderer(ctx context.Context, config DeploymentConfig) error {
	name := ResourceName(config.InstanceName, "grafana-renderer")
	labels := map[string]string{
		LabelManagedBy: ManagedByValue,
		LabelComponent: ComponentValue,
		LabelName:      config.InstanceName,
		"app":          name,
	}
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: config.Namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:  "renderer",
					Image: "grafana/grafana-image-renderer:latest",
					Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8081, Protocol: corev1.ProtocolTCP}},
					Env: []corev1.EnvVar{{
						Name: "AUTH_TOKEN",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: ResourceName(config.InstanceName, "grafana-admin")},
								Key:                  "renderer-token",
							},
						},
					}},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi"), corev1.ResourceCPU: resource.MustParse("100m")},
						Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi"), corev1.ResourceCPU: resource.MustParse("500m")},
					},
				}}},
			},
		},
	}
	if err := d.createOrUpdateDeployment(ctx, deployment); err != nil {
		return err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: config.Namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    []corev1.ServicePort{{Name: "http", Port: 8081, TargetPort: intstr.FromInt(8081), Protocol: corev1.ProtocolTCP}},
			Selector: map[string]string{"app": name},
		},
	}
	if err := d.client.Create(ctx, service); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createGrafanaService creates the Grafana Service
func (d *GrafanaDeployer) createGrafanaService(ctx context.Context, config DeploymentConfig) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceName(config.InstanceName, "grafana"),
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
				"app":          ResourceName(config.InstanceName, "grafana"),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       3000,
					TargetPort: intstr.FromInt(3000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": ResourceName(config.InstanceName, "grafana"),
			},
		},
	}

	err := d.client.Create(ctx, service)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createGrafanaRoute creates an OpenShift Route for external access
func (d *GrafanaDeployer) createGrafanaRoute(ctx context.Context, config DeploymentConfig) (*routev1.Route, error) {
	weight := int32(100)

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceName(config.InstanceName, "grafana"),
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
				"app":          ResourceName(config.InstanceName, "grafana"),
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   ResourceName(config.InstanceName, "grafana"),
				Weight: &weight,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
			},
		},
	}

	err := d.client.Create(ctx, route)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Route already exists, fetch it
			existingRoute := &routev1.Route{}
			if getErr := d.client.Get(ctx, client.ObjectKey{
				Namespace: config.Namespace,
				Name:      ResourceName(config.InstanceName, "grafana"),
			}, existingRoute); getErr == nil {
				return existingRoute, nil
			}
		}
		return nil, err
	}

	// Wait a moment for the route host to be assigned
	if err := d.client.Get(ctx, client.ObjectKey{
		Namespace: config.Namespace,
		Name:      ResourceName(config.InstanceName, "grafana"),
	}, route); err != nil {
		return nil, err
	}

	return route, nil
}

// DeleteGrafana removes resources belonging to one Grafana instance without
// deleting the shared target namespace.
func (d *GrafanaDeployer) DeleteGrafana(ctx context.Context, namespace, instanceName string) error {
	logger := log.FromContext(ctx).WithName("grafana-deployer")
	logger.Info("Deleting Grafana deployment", "namespace", namespace, "instance", instanceName)

	resources := []client.Object{
		&routev1.Route{ObjectMeta: metav1.ObjectMeta{Name: ResourceName(instanceName, "grafana"), Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: ResourceName(instanceName, "grafana"), Namespace: namespace}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: ResourceName(instanceName, "grafana"), Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: ResourceName(instanceName, "grafana-renderer"), Namespace: namespace}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: ResourceName(instanceName, "grafana-renderer"), Namespace: namespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: ResourceName(instanceName, "grafana-datasources"), Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: ResourceName(instanceName, "grafana-admin"), Namespace: namespace}},
	}
	for _, resource := range resources {
		if err := d.client.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete %s: %w", resource.GetName(), err)
		}
	}

	// Delete any remaining pods owned by this instance. Normally removing the
	// Deployment removes its pods, but deleting them explicitly also cleans up
	// orphaned pods left behind by an interrupted deployment.
	if err := d.client.DeleteAllOf(ctx, &corev1.Pod{},
		client.InNamespace(namespace),
		client.MatchingLabels{
			LabelManagedBy: ManagedByValue,
			LabelComponent: ComponentValue,
			LabelName:      instanceName,
		},
	); err != nil {
		return fmt.Errorf("failed to delete Grafana pods for %s: %w", instanceName, err)
	}

	logger.Info("Grafana deployment deleted successfully", "namespace", namespace, "instance", instanceName)
	return nil
}

// DeleteNamespace removes the namespace after the caller has confirmed that
// no other krkn-visualize instances are using it.
func (d *GrafanaDeployer) DeleteNamespace(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{}
	if err := d.client.Get(ctx, client.ObjectKey{Name: namespace}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get namespace %s: %w", namespace, err)
	}

	if err := d.client.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace %s: %w", namespace, err)
	}
	return nil
}

// EnableGrafanaEmbedding configures a Grafana instance deployed by the
// upstream krkn-visualize deploy image to allow dashboard embedding.
//
// Grafana denies framing by default (X-Frame-Options: deny), which blocks the
// console's Visualize page from showing dashboards in an iframe. Setting
// allow_embedding = true in the [security] section of grafana.ini lifts that
// restriction; anonymous Viewer auth is already enabled by the upstream
// template, so embedded dashboards remain read-only.
//
// The change is applied by patching the sc-grafana-config ConfigMap and
// restarting the krkn-visualize Deployment so the running Grafana reloads
// its configuration. The operation is idempotent: if the setting is already
// present with the requested value, nothing is updated.
func EnableGrafanaEmbedding(ctx context.Context, c client.Client, namespace string) error {
	logger := log.FromContext(ctx).WithName("grafana-embedder")

	configMap := &corev1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: UpstreamGrafanaConfigMap}, configMap); err != nil {
		return fmt.Errorf("failed to get Grafana config ConfigMap: %w", err)
	}

	ini, ok := configMap.Data[GrafanaIniKey]
	if !ok {
		return fmt.Errorf("ConfigMap %s does not contain key %s", configMap.Name, GrafanaIniKey)
	}

	updated, changed := ensureIniSetting(ini, "security", "allow_embedding", "true")
	if !changed {
		logger.Info("Grafana embedding already enabled", "namespace", namespace)
		return nil
	}

	configMap.Data[GrafanaIniKey] = updated
	if err := c.Update(ctx, configMap); err != nil {
		return fmt.Errorf("failed to update Grafana config ConfigMap: %w", err)
	}

	// Restart the Grafana deployment so the new grafana.ini is picked up.
	deployment := &appsv1.Deployment{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: UpstreamGrafanaDeployment}, deployment); err != nil {
		return fmt.Errorf("failed to get Grafana deployment: %w", err)
	}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	if err := c.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to restart Grafana deployment: %w", err)
	}

	logger.Info("Enabled Grafana embedding", "namespace", namespace, "configMap", configMap.Name)
	return nil
}

// ensureIniSetting sets key = value under the given INI section of ini and
// reports whether the content changed. A missing section is appended. Values
// in comments (';' or '#' prefixes) are ignored when matching keys. If the
// key already exists with the requested value the input is returned unchanged.
func ensureIniSetting(ini, section, key, value string) (string, bool) {
	lines := strings.Split(ini, "\n")
	sectionHeader := "[" + section + "]"

	keyLine := key + " = " + value

	inSection := false
	sectionSeen := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if inSection {
				// We just left the target section without finding the key:
				// insert it as the last line of the section.
				updated := append(lines[:i], append([]string{keyLine}, lines[i:]...)...)
				return strings.Join(updated, "\n"), true
			}
			inSection = trimmed == sectionHeader
			if inSection {
				sectionSeen = true
			}
			continue
		}
		if !inSection || trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		name := trimmed
		if idx := strings.Index(trimmed, "="); idx >= 0 {
			name = strings.TrimSpace(trimmed[:idx])
		}
		if name == key {
			if trimmed == keyLine {
				return ini, false
			}
			lines[i] = keyLine
			return strings.Join(lines, "\n"), true
		}
	}

	if inSection {
		// Target section is the last one in the file.
		return strings.TrimRight(ini, "\n") + "\n" + keyLine + "\n", true
	}
	if !sectionSeen {
		// Section does not exist at all; append it.
		return strings.TrimRight(ini, "\n") + "\n\n" + sectionHeader + "\n" + keyLine + "\n", true
	}
	return ini, false
}
