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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package provider

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// ProviderRegistration manages the KrknOperatorTargetProvider CR registration and heartbeat
type ProviderRegistration struct {
	client            client.Client
	namespace         string
	providerName      string
	heartbeatInterval time.Duration
	stopCh            chan struct{}
}

// Config holds the configuration for provider registration
type Config struct {
	// ProviderName is the name to register as (e.g., "krkn-operator", "my-custom-operator")
	ProviderName string

	// HeartbeatInterval is the interval at which the provider heartbeat is updated
	// If not set, defaults to 30 seconds
	HeartbeatInterval time.Duration

	// Namespace is the namespace where the provider CR will be created
	Namespace string
}

// NewProviderRegistration creates a new provider registration manager
// Deprecated: Use NewProviderRegistrationWithConfig instead
func NewProviderRegistration(c client.Client, namespace string) *ProviderRegistration {
	return NewProviderRegistrationWithConfig(c, Config{
		ProviderName:      "krkn-operator",
		HeartbeatInterval: 30 * time.Second,
		Namespace:         namespace,
	})
}

// NewProviderRegistrationWithConfig creates a new provider registration manager with custom configuration
func NewProviderRegistrationWithConfig(c client.Client, cfg Config) *ProviderRegistration {
	// Set defaults
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}
	if cfg.ProviderName == "" {
		cfg.ProviderName = "krkn-operator"
	}

	return &ProviderRegistration{
		client:            c,
		namespace:         cfg.Namespace,
		providerName:      cfg.ProviderName,
		heartbeatInterval: cfg.HeartbeatInterval,
		stopCh:            make(chan struct{}),
	}
}

// Start implements manager.Runnable interface
// It ensures the provider CR exists and starts the heartbeat goroutine
func (p *ProviderRegistration) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("provider-registration")
	logger.Info("Starting provider registration", "name", p.providerName, "namespace", p.namespace)

	// Create or update provider CR
	if err := p.ensureProvider(ctx); err != nil {
		logger.Error(err, "Failed to ensure provider CR")
		return err
	}

	logger.Info("Provider CR ensured successfully")

	// Start heartbeat goroutine
	ticker := time.NewTicker(p.heartbeatInterval)
	defer ticker.Stop()

	logger.Info("Starting heartbeat loop", "interval", p.heartbeatInterval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled, deactivating provider")
			// Deactivate provider on shutdown (use background context as ctx is cancelled)
			deactivateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := p.deactivateProvider(deactivateCtx); err != nil {
				logger.Error(err, "Failed to deactivate provider")
			} else {
				logger.Info("Provider deactivated successfully")
			}
			return nil

		case <-ticker.C:
			if err := p.updateHeartbeat(ctx); err != nil {
				logger.Error(err, "Failed to update heartbeat")
				// Continue anyway - don't stop on heartbeat errors
			} else {
				logger.V(1).Info("Heartbeat updated successfully")
			}
		}
	}
}

// ensureProvider creates or updates the KrknOperatorTargetProvider CR
func (p *ProviderRegistration) ensureProvider(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("provider-registration")

	provider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.providerName,
			Namespace: p.namespace,
		},
	}

	result, err := controllerutil.CreateOrUpdate(ctx, p.client, provider, func() error {
		// Set spec fields
		provider.Spec.OperatorName = p.providerName
		provider.Spec.Active = true

		// Initialize status
		if provider.Status.Timestamp.IsZero() {
			provider.Status.Timestamp = metav1.Now()
		}

		return nil
	})

	if err != nil {
		return err
	}

	logger.Info("Provider CR operation completed", "result", result, "name", p.providerName)
	return nil
}

// updateHeartbeat updates the provider's heartbeat timestamp
func (p *ProviderRegistration) updateHeartbeat(ctx context.Context) error {
	var provider krknv1alpha1.KrknOperatorTargetProvider
	if err := p.client.Get(ctx, types.NamespacedName{
		Name:      p.providerName,
		Namespace: p.namespace,
	}, &provider); err != nil {
		return err
	}

	provider.Status.Timestamp = metav1.Now()
	return p.client.Status().Update(ctx, &provider)
}

// deactivateProvider sets the provider's active flag to false
func (p *ProviderRegistration) deactivateProvider(ctx context.Context) error {
	var provider krknv1alpha1.KrknOperatorTargetProvider
	if err := p.client.Get(ctx, types.NamespacedName{
		Name:      p.providerName,
		Namespace: p.namespace,
	}, &provider); err != nil {
		return client.IgnoreNotFound(err)
	}

	provider.Spec.Active = false
	return p.client.Update(ctx, &provider)
}

// NeedLeaderElection implements manager.LeaderElectionRunnable
// Provider registration should only run on the leader
func (p *ProviderRegistration) NeedLeaderElection() bool {
	return true
}

// Ensure ProviderRegistration implements the required interfaces
var _ manager.LeaderElectionRunnable = &ProviderRegistration{}
