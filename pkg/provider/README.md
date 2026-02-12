# Provider Registration Package

This package provides reusable provider registration functionality for Kubernetes operators that implement the Krkn operator target provider pattern.

## Overview

The provider registration package manages the lifecycle of a `KrknOperatorTargetProvider` Custom Resource (CR), including:
- Creating and registering the provider CR
- Sending periodic heartbeat updates
- Deactivating the provider on shutdown
- Leader election support

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

### Functions

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

### Interfaces

`ProviderRegistration` implements:
- `manager.Runnable` - Can be added to a controller-runtime manager
- `manager.LeaderElectionRunnable` - Only runs on leader pod

## License

Copyright 2025. Licensed under the Apache License, Version 2.0.
