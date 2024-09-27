# Pluma Operator

Helm operator and Istio operator

Pluma Operator is a Kubernetes operator that provides advanced component management capabilities using Helm charts. It offers continuous lifecycle management for installed components and supports the conversion of Istio Custom Resource Definitions (CRDs) into HelmApp resources for streamlined Istio installation.

## Key Capabilities

1. **Helm Integration**: Utilizes Helm charts for efficient and standardized component deployment.
2. **Lifecycle Management**: Provides ongoing maintenance and updates for installed components.
3. **Istio Support**: Converts Istio CRDs to HelmApp resources, enabling suite-based Istio installation.
4. **Kubernetes Native**: Seamlessly integrates with Kubernetes environments for streamlined operations.

## install

To install the Pluma Operator using Helm, execute the following command. This command will perform an upgrade if the Pluma Operator is already installed or install it if it’s not present. It will also automatically create the `pluma-system` namespace if it doesn't exist.

```bash
helm upgrade --install pluma-operator ./manifests/pluma --create-namespace --namespace pluma-system
```

## Getting Started

### Install Istio

Use IstioOperator CRD

#### Istio Mesh Demo

```yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: nicole-dmesh-mspider-mcpc
  namespace: istio-system
spec:
  components:
    ingressGateways:
      - enabled: true
        k8s:
          resources:
            limits:
              cpu: 1000m
              memory: 900Mi
            requests:
              cpu: 50m
              memory: 50Mi
        name: istio-ingressgateway
    pilot:
      k8s:
        resources:
          limits:
            cpu: 1500m
            memory: 1500Mi
          requests:
            cpu: 200m
            memory: 200Mi
  hub: release-ci.daocloud.io/mspider
  meshConfig:
    defaultConfig:
      extraStatTags:
        - destination_mesh_id
        - source_mesh_id
      proxyMetadata:
        ISTIO_META_DNS_AUTO_ALLOCATE: 'true'
        ISTIO_META_DNS_CAPTURE: 'true'
        WASM_INSECURE_REGISTRIES: '*'
      tracing:
        sampling: 100
    enableTracing: true
    extensionProviders:
      - name: otel
        opentelemetry:
          port: 4317
          service: >-
            insight-agent-opentelemetry-collector.insight-system.svc.cluster.local
  namespace: istio-system
  profile: default
  tag: 1.21.1
  values:
    gateways:
      istio-ingressgateway:
        autoscaleEnabled: true
        autoscaleMin: 1
      securityContext:
        sysctls: []
    global:
      istioNamespace: istio-system
      meshID: mspider-dedicated
      multiCluster:
        clusterName: mspider-dedicated
      network: internal-net
      proxy:
        logLevel: warning
        resources:
          limits:
            cpu: 600m
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 20Mi
    meshConfig:
      enableTracing: true
      outboundTrafficPolicy:
        mode: ALLOW_ANY
    pilot:
      autoscaleEnabled: true
      autoscaleMin: 1
      replicaCount: 1
    sidecarInjectorWebhook:
      enableNamespacesByDefault: false

```

#### Istio Gateway Demo

```yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: test-gw
  namespace: default
spec:
  components:
    ingressGateways:
      - enabled: true
        k8s:
          replicaCount: 1
          resources:
            limits:
              cpu: 600m
              memory: 200Mi
            requests:
              cpu: 200m
              memory: 200Mi
          service:
            ports:
              - name: http-0
                port: 80
                protocol: TCP
                targetPort: 8080
            type: NodePort
        label:
          mspider.io/mesh-gateway-name: test-gw
          test-gw: test-gw
        name: test-gw
        namespace: default
  profile: empty
  tag: 1.21.1
  values:
    gateways:
      istio-ingressgateway:
        autoscaleEnabled: false
        injectionTemplate: gateway
    global:
      hub: release-ci.daocloud.io/mspider
```

## Common helm application

```yaml
apiVersion: operator.pluma.io/v1alpha1
kind: HelmApp
metadata:
  name: helm-demo
  namespace: default
spec:
  components:
    - chart: gateway
      componentValues:
        resources:
          limits:
            cpu: 600m
            memory: 200Mi
          requests:
            cpu: 200m
            memory: 200Mi
      name: demo
      version: 1.21.1
  globalValues:
    hub: release-ci.daocloud.io/mspider
  repo:
    name: istio
    url: https://istio-release.storage.googleapis.com/charts    

```

## HelmApp CRD

### Status
```yaml
status:
  components:
  - name: demo
    resources:
    - apiVersion: v1
      kind: ServiceAccount
      name: demo
      namespace: default
    - apiVersion: rbac.authorization.k8s.io/v1
      kind: Role
      name: demo
      namespace: default
    - apiVersion: rbac.authorization.k8s.io/v1
      kind: RoleBinding
      name: demo
      namespace: default
    - apiVersion: v1
      kind: Service
      name: demo
      namespace: default
    - apiVersion: apps/v1
      kind: Deployment
      name: demo
      namespace: default
    - apiVersion: autoscaling/v2
      kind: HorizontalPodAutoscaler
      name: demo
      namespace: default
    resourcesTotal: 6
    status: deployed
    version: "1"
  phase: SUCCEEDED
```
