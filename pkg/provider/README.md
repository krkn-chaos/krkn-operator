# Provider Package

This package provides reusable functionality for Kubernetes operators that implement the Krkn operator target provider pattern.

## Overview

The provider package includes two main components:

1. **Provider Registration** - Manages the lifecycle of a `KrknOperatorTargetProvider` CR:
   - Creating and registering the provider CR
   - Sending periodic heartbeat updates
   - Deactivating the provider on shutdown
   - Leader election support

2. **Provider Configuration** - Manages provider configuration schemas:
   - Creating config requests
   - Contributing configuration data
   - JSON schema validation

## Usage

### Basic Usage (Default Configuration)

```go
import (
    "sigs.k8s.io/controller-runtime/pkg/manager"
    "github.com/krkn-chaos/krkn-operator/pkg/provider"
)

func main() {
    // ... setup manager ...

    // Create provider registration with defaults
    // Provider name: "krkn-operator"
    // Heartbeat interval: 30 seconds
    providerReg := provider.NewProviderRegistration(mgr.GetClient(), namespace)

    // Add to manager
    if err := mgr.Add(providerReg); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Configuration

```go
import (
    "time"
    "sigs.k8s.io/controller-runtime/pkg/manager"
    "github.com/krkn-chaos/krkn-operator/pkg/provider"
)

func main() {
    // ... setup manager ...

    // Create provider registration with custom configuration
    providerReg := provider.NewProviderRegistrationWithConfig(mgr.GetClient(), provider.Config{
        ProviderName:      "my-custom-operator",
        HeartbeatInterval: 60 * time.Second,
        Namespace:         namespace,
    })

    // Add to manager
    if err := mgr.Add(providerReg); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration Options

### Config Struct

```go
type Config struct {
    // ProviderName is the name to register as (e.g., "krkn-operator", "my-custom-operator")
    ProviderName string

    // HeartbeatInterval is the interval at which the provider heartbeat is updated
    // If not set, defaults to 30 seconds
    HeartbeatInterval time.Duration

    // Namespace is the namespace where the provider CR will be created
    Namespace string
}
```

### Default Values

- **ProviderName**: `"krkn-operator"`
- **HeartbeatInterval**: `30 * time.Second`

## How It Works

1. **Registration**: On startup, the provider registration creates or updates a `KrknOperatorTargetProvider` CR with `Active: true`

2. **Heartbeat**: Every `HeartbeatInterval`, the provider updates the `Status.Timestamp` field to indicate it's still alive

3. **Deactivation**: On shutdown, the provider sets `Active: false` to signal it's no longer available

4. **Leader Election**: The provider registration only runs on the leader pod (implements `manager.LeaderElectionRunnable`)

## Example Integration

Here's a complete example of integrating provider registration into your operator:

```go
package main

import (
    "os"
    "time"

    ctrl "sigs.k8s.io/controller-runtime"
    "github.com/krkn-chaos/krkn-operator/pkg/provider"
)

func main() {
    // Get operator configuration
    operatorName := os.Getenv("OPERATOR_NAME")
    if operatorName == "" {
        operatorName = "my-operator"
    }

    namespace := os.Getenv("POD_NAMESPACE")
    if namespace == "" {
        namespace = "default"
    }

    // Create manager
    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
        // ... manager options ...
    })
    if err != nil {
        panic(err)
    }

    // Setup provider registration
    providerReg := provider.NewProviderRegistrationWithConfig(mgr.GetClient(), provider.Config{
        ProviderName:      operatorName,
        HeartbeatInterval: 45 * time.Second,
        Namespace:         namespace,
    })

    if err := mgr.Add(providerReg); err != nil {
        panic(err)
    }

    // Start manager
    if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
        panic(err)
    }
}
```

## Requirements

- **Kubernetes API Version**: Requires the `KrknOperatorTargetProvider` CRD to be installed
- **RBAC Permissions**: The operator service account needs permissions to create/update/get `KrknOperatorTargetProvider` resources
- **Controller Runtime**: Compatible with `sigs.k8s.io/controller-runtime` v0.21.0+

## API Reference

### Provider Registration Functions

#### NewProviderRegistration
```go
func NewProviderRegistration(c client.Client, namespace string) *ProviderRegistration
```
Creates a provider registration with default configuration. **Deprecated**: Use `NewProviderRegistrationWithConfig` for custom configuration.

#### NewProviderRegistrationWithConfig
```go
func NewProviderRegistrationWithConfig(c client.Client, cfg Config) *ProviderRegistration
```
Creates a provider registration with custom configuration.

### Provider Configuration Functions

#### CreateProviderConfigRequest
```go
func CreateProviderConfigRequest(
    ctx context.Context,
    c client.Client,
    namespace string,
    name string,
) (string, error)
```
Creates a new KrknOperatorTargetProviderConfig CR and generates a unique UUID. Returns the UUID for tracking the config request.

#### UpdateProviderConfig
```go
func UpdateProviderConfig(
    ctx context.Context,
    c client.Client,
    config *krknv1alpha1.KrknOperatorTargetProviderConfig,
    operatorName string,
    configMapName string,
    jsonSchema string,
) error
```
Updates a KrknOperatorTargetProviderConfig CR with provider configuration data. Takes the CR object directly (already fetched by the reconcile loop). Validates JSON schema before updating.

### Interfaces

`ProviderRegistration` implements:
- `manager.Runnable` - Can be added to a controller-runtime manager
- `manager.LeaderElectionRunnable` - Only runs on leader pod

---

## Provider Configuration

The provider package also provides functions for managing provider configuration schemas through `KrknOperatorTargetProviderConfig` resources.

### Configuration Functions

#### CreateProviderConfigRequest

Creates a new config request and generates a unique UUID.

```go
func CreateProviderConfigRequest(
    ctx context.Context,
    c client.Client,
    namespace string,
    name string,
) (string, error)
```

**Parameters:**
- `ctx` - Context
- `c` - Kubernetes client
- `namespace` - Namespace where the CR will be created
- `name` - Optional name for the CR (if empty, generates "config-" + UUID prefix)

**Returns:**
- `uuid` - The generated UUID for this config request
- `error` - Error if creation fails

**Example:**
```go
import (
    "context"
    "github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// Create a new config request
uuid, err := provider.CreateProviderConfigRequest(
    context.Background(),
    k8sClient,
    "krkn-operator-system",
    "", // auto-generate name
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Config request created with UUID: %s\n", uuid)
```

#### UpdateProviderConfig

Updates a config request with provider configuration data.

```go
func UpdateProviderConfig(
    ctx context.Context,
    c client.Client,
    config *krknv1alpha1.KrknOperatorTargetProviderConfig,
    operatorName string,
    configMapName string,
    jsonSchema string,
) error
```

**Parameters:**
- `ctx` - Context
- `c` - Kubernetes client
- `config` - The KrknOperatorTargetProviderConfig CR object (already fetched by the reconcile loop)
- `operatorName` - Name of the provider contributing the data (e.g., "krkn-operator-acm")
- `configMapName` - Name of the ConfigMap containing the provider's configuration
- `jsonSchema` - JSON schema string for the provider's configuration (must be valid JSON)

**Returns:**
- `error` - Error if update fails or validation fails

**Note:** Provider controllers have already fetched the CR in their reconcile loop, so they simply pass the CR object directly. This avoids redundant fetches.

**Example (in a controller):**
```go
import (
    "context"
    "encoding/json"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "github.com/krkn-chaos/krkn-operator/pkg/provider"
    krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Fetch the config request CR
    var config krknv1alpha1.KrknOperatorTargetProviderConfig
    if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Skip if already contributed
    if _, exists := config.Status.ConfigData["my-operator"]; exists {
        return ctrl.Result{}, nil
    }

    // Define JSON schema for your operator's configuration
    schema := map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "chaos-level": map[string]interface{}{
                "type": "string",
                "enum": []string{"low", "medium", "high"},
            },
        },
    }

    schemaBytes, _ := json.Marshal(schema)

    // Contribute your configuration - pass the CR object directly
    err := provider.UpdateProviderConfig(
        ctx,
        r.Client,
        &config, // Pass the CR we already fetched
        "my-operator",
        "my-operator-config",
        string(schemaBytes),
    )
    if err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

### Configuration Workflow

1. **Client creates request**: Calls `CreateProviderConfigRequest()` and receives a UUID
2. **Providers contribute**: Each operator calls `UpdateProviderConfig()` with its schema
3. **Aggregation**: krkn-operator aggregates all contributions
4. **Completion**: Status becomes "Completed" when all active providers contribute

### Validation

`UpdateProviderConfig` performs the following validations:
- All required parameters are non-empty
- JSON schema (if provided) is valid JSON
- Config request exists with the given UUID

### Usage in Controllers

Operators should implement a controller that:
1. Watches for new `KrknOperatorTargetProviderConfig` CRs
2. Prepares its configuration (creates ConfigMap, generates schema)
3. Calls `UpdateProviderConfig()` to contribute data

See `docs/provider-config-integration.md` for a complete integration guide.

---

## Resource Cleanup

The provider package provides a generic, idempotent function for cleaning up old CRD instances based on their `Created` timestamp.

### CleanupOldResources

Deletes all instances of a specific CRD type in a namespace whose `Created` field is older than a specified number of seconds.

```go
func CleanupOldResources(
    ctx context.Context,
    c client.Client,
    emptyList client.ObjectList,
    namespace string,
    olderThanSeconds int64,
    getCreatedTime CreatedTimeExtractor,
) (int, error)
```

**Key Features:**
- **Idempotent**: Safe for concurrent execution by multiple operators
- **Panic-safe**: Never panics, even if extractor function panics
- **Conflict-tolerant**: Logs warnings for conflicts but doesn't fail
- **Generic**: Works with any CRD type that has a `Created` timestamp

**Parameters:**
- `ctx` - Context for the operation
- `c` - Kubernetes client
- `emptyList` - An empty instance of the list type (e.g., `&krknv1alpha1.KrknOperatorTargetProviderConfigList{}`)
- `namespace` - Namespace to search in
- `olderThanSeconds` - Age threshold in seconds - resources older than this will be deleted
- `getCreatedTime` - Function to extract the `Created` timestamp from an object

**Returns:**
- `deletedCount` - Number of resources successfully deleted
- `error` - Non-nil only for critical errors (listing failures); deletion conflicts are logged but don't cause errors

**Example Usage:**

```go
import (
    "context"
    "github.com/krkn-chaos/krkn-operator/pkg/provider"
    krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

// Clean up KrknOperatorTargetProviderConfig resources older than 1 hour
deletedCount, err := provider.CleanupOldResources(
    ctx,
    k8sClient,
    &krknv1alpha1.KrknOperatorTargetProviderConfigList{},
    "krkn-operator-system",
    3600, // 1 hour in seconds
    func(obj client.Object) *metav1.Time {
        config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
        return config.Status.Created
    },
)
if err != nil {
    log.Error(err, "Failed to cleanup old configs")
}
log.Info("Cleanup completed", "deletedCount", deletedCount)
```

**Example: Periodic Cleanup in a Controller**

```go
import (
    "context"
    "time"
    ctrl "sigs.k8s.io/controller-runtime"
    "github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// Run cleanup every hour using a goroutine
func (r *MyReconciler) setupPeriodicCleanup(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                deletedCount, err := provider.CleanupOldResources(
                    ctx,
                    r.Client,
                    &krknv1alpha1.KrknOperatorTargetProviderConfigList{},
                    r.Namespace,
                    7200, // Delete resources older than 2 hours
                    func(obj client.Object) *metav1.Time {
                        config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
                        return config.Status.Created
                    },
                )
                if err != nil {
                    log.Error(err, "Periodic cleanup failed")
                } else {
                    log.Info("Periodic cleanup completed", "deletedCount", deletedCount)
                }
            }
        }
    }()
}
```

**Safety Guarantees:**
- **No panics**: Function recovers from panics in the extractor function
- **Idempotent**: Can be called by multiple operators concurrently without issues
- **Graceful conflict handling**: Logs warnings for conflicts (e.g., resource already deleted) but continues
- **Resource skipping**: Skips resources without a `Created` timestamp or with recent timestamps

**Use Cases:**
- Clean up completed config requests after a retention period
- Remove old target requests to prevent cluster bloat
- Implement TTL-like behavior for temporary resources
- Periodic garbage collection of stale CRs

---

## License

Copyright 2025. Licensed under the Apache License, Version 2.0.
