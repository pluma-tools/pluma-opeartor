# Pluma Operator

Helm operator and Istio operator

Pluma Operator is a Kubernetes operator that provides advanced component management capabilities using Helm charts. It offers continuous lifecycle management for installed components and supports the conversion of Istio Custom Resource Definitions (CRDs) into HelmApp resources for streamlined Istio installation.

## Features

- Helm-based component installation and management
- Continuous lifecycle management for installed components
- Conversion of Istio CRDs to HelmApp resources
- Simplified, suite-based Istio installation

## Key Capabilities

1. **Helm Integration**: Utilizes Helm charts for efficient and standardized component deployment.
2. **Lifecycle Management**: Provides ongoing maintenance and updates for installed components.
3. **Istio Support**: Converts Istio CRDs to HelmApp resources, enabling suite-based Istio installation.
4. **Kubernetes Native**: Seamlessly integrates with Kubernetes environments for streamlined operations.

## Getting Started

TODO

## Documentation

TODO

## install

To install the Pluma Operator using Helm, execute the following command. This command will perform an upgrade if the Pluma Operator is already installed or install it if it’s not present. It will also automatically create the `pluma-system` namespace if it doesn't exist.

```bash
helm upgrade --install pluma-operator ./manifests/pluma --create-namespace --namespace pluma-system
```

```

```
