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
	"fmt"

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

	// Step 2: Create ConfigMap with datasources
	if err := d.createDatasourcesConfigMap(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create datasources ConfigMap: %w", err)
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

	// Step 6: Create OpenShift Route for external access
	var grafanaURL string
	if err := CreateGrafanaRoute(ctx, d.client, config.Namespace, config.InstanceName); err != nil {
		logger.Error(err, "Failed to create OpenShift Route, falling back to service URL")
		grafanaURL = fmt.Sprintf("http://grafana.%s.svc.cluster.local:3000", config.Namespace)
	} else {
		logger.Info("Route created successfully, waiting for host assignment...")
		// Wait for Route host to be assigned
		routeURL, err := GetRouteURL(ctx, d.client, config.Namespace)
		if err != nil {
			logger.Error(err, "Failed to get Route URL after polling, falling back to service URL",
				"namespace", config.Namespace,
				"routeName", "grafana")
			grafanaURL = fmt.Sprintf("http://grafana.%s.svc.cluster.local:3000", config.Namespace)
		} else {
			grafanaURL = routeURL
			logger.Info("Route URL retrieved successfully", "url", grafanaURL)
		}
	}

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

// createDatasourcesConfigMap creates ConfigMap with Elasticsearch and Prometheus datasources
func (d *GrafanaDeployer) createDatasourcesConfigMap(ctx context.Context, config DeploymentConfig) error {
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

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-datasources",
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
			},
		},
		Data: map[string]string{
			"datasources.yaml": datasourcesYAML,
		},
	}

	err := d.client.Create(ctx, cm)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createGrafanaSecret creates Secret with Grafana admin credentials
func (d *GrafanaDeployer) createGrafanaSecret(ctx context.Context, config DeploymentConfig) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-admin",
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
		},
	}

	err := d.client.Create(ctx, secret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createGrafanaDeployment creates the Grafana Deployment
func (d *GrafanaDeployer) createGrafanaDeployment(ctx context.Context, config DeploymentConfig) error {
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
				"app":          "grafana",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "grafana",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":          "grafana",
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
									Name: "GF_SECURITY_ADMIN_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "grafana-admin",
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
												Name: "grafana-admin",
											},
											Key: "admin-password",
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
					},
					Volumes: []corev1.Volume{
						{
							Name: "datasources",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "grafana-datasources",
									},
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

	err := d.client.Create(ctx, deployment)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createGrafanaService creates the Grafana Service
func (d *GrafanaDeployer) createGrafanaService(ctx context.Context, config DeploymentConfig) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
				"app":          "grafana",
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
				"app": "grafana",
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
			Name:      "grafana",
			Namespace: config.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      config.InstanceName,
				"app":          "grafana",
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   "grafana",
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
				Name:      "grafana",
			}, existingRoute); getErr == nil {
				return existingRoute, nil
			}
		}
		return nil, err
	}

	// Wait a moment for the route host to be assigned
	if err := d.client.Get(ctx, client.ObjectKey{
		Namespace: config.Namespace,
		Name:      "grafana",
	}, route); err != nil {
		return nil, err
	}

	return route, nil
}

// DeleteGrafana removes Grafana deployment
func (d *GrafanaDeployer) DeleteGrafana(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx).WithName("grafana-deployer")
	logger.Info("Deleting Grafana deployment", "namespace", namespace)

	// Delete namespace (cascade deletes all resources)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	if err := d.client.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	logger.Info("Grafana deployment deleted successfully", "namespace", namespace)
	return nil
}
