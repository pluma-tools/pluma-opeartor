# IstioOperator 迁移到 Pluma 安装模式

## 背景

因为 Istio 社区废弃了 Istio Operator，因此，需要维护一个维护 Istio 安装的 Operator 需要寻找另外的出路。

下面描述如何通过 Pluma Operator 来替换 Istio Operator。

在这里如何将现有的 Istio 网格升级为 Pluma Operator 管理的网格非常重要，我们尽量支持无缝迁移，
但是作为生产环境的网格，为了考虑起稳定性，也允许一定程度的手动升级，这篇文章让我带大家来了解如何将网格迁移为 Pluma 安装模式。

## 原理

这里简要介绍一下如何支持 IstioOperator CRD 来安装 Istio 的原理。

通过将 IstioOperator CRD 转换成 HelmApp CRD，然后 Pluma 将通过标准 Helm 分部分安装 IstioOperator。

## 迁移

### 大致步骤

1. ‼️ 非常重要：停用 Istio Operator 组件（原生 Istio 模式）
2. 检查 IstioOperator CRD 规范
3. 明确迁移变化范围
4. 开启 IOP 强制迁移，添加 label `action.pluma.io/allow-froce-upgrade: true`
5. 手动修改（可选）
6. 安装 Pluma
7. 确认组件升级安装

### IstioOperator CRD 规范

#### 数组对象需要定义完整

components 部分 ingressGateways 和 egressGateways 类似的 嵌套了对象数组，在模版渲染时无法根据对象去深度渲染，
因此需要保证其配置的完整性。

举例如下：

```yaml
# 模版
components:
  base:
    enabled: true
  pilot:
    enabled: true
  # Istio Gateway feature
  ingressGateways:
    - name: istio-ingressgateway
      enabled: true
  egressGateways:
    - name: istio-egressgateway
      enabled: false
```

```yaml
# 自定义
components:
  pilot:
    k8s:
      resources:
        limits:
          cpu: 1500m
          memory: 1500Mi
        requests:
          cpu: 200m
          memory: 200Mi
  ingressGateways:
    - name: istio-ingressgateway
```

最后渲染结果为：

```yaml
  components:
    base:
      enabled: true
    pilot:
      enabled: true
      k8s:
        resources:
          limits:
            cpu: 1500m
            memory: 1500Mi
          requests:
            cpu: 200m
            memory: 200Mi
    ingressGateways:
      - name: istio-ingressgateway
        enabled: false # bool 规范，默认为 false， 这时网关将不会安装
    egressGateways:
      - name: istio-egressgateway
        enabled: false
```

### 迁移变化范围

#### Istiod

Istiod 升级后没有太大变化，主要是 label 和 annotation 的变化

```bash
$ diff old-istiod.yaml new-istiod.yaml
>     meta.helm.sh/release-name: iop-nicole-dmesh-mspider-mcpc-istiod
>     meta.helm.sh/release-namespace: istio-system
8c10,11
<     install.operator.istio.io/owning-resource: nicole-dmesh-mspider-mcpc
---
>     app.kubernetes.io/managed-by: Helm
>     install.operator.istio.io/owning-resource: unknown
15c18
<     release: istio
---
>     release: iop-nicole-dmesh-mspider-mcpc-istiod
222a226,228
>   annotations:
>     meta.helm.sh/release-name: iop-nicole-dmesh-mspider-mcpc-istiod
>     meta.helm.sh/release-namespace: istio-system
224c230,231
<     install.operator.istio.io/owning-resource: nicole-dmesh-mspider-mcpc
---
>     app.kubernetes.io/managed-by: Helm
>     install.operator.istio.io/owning-resource: unknown
230c237
<     release: istio
---
>     release: iop-nicole-dmesh-mspider-mcpc-istiod
```


#### 默认 Ingress 网关

注意：**网关将会重启**
新架构下的网关采用了标准的 Helm Gateway 模版，因此需要发生了变化，变化如下：

```bash
$ diff old-ingressgateway.yaml new-ingressgateway.yaml

5c5,7
<     deployment.kubernetes.io/revision: "1"
---
>     deployment.kubernetes.io/revision: "2"
>     meta.helm.sh/release-name: istio-ingressgateway
>     meta.helm.sh/release-namespace: istio-system
7a10,13
>     app.kubernetes.io/managed-by: Helm
>     app.kubernetes.io/name: istio-ingressgateway
>     app.kubernetes.io/version: 1.22.4
>     helm.sh/chart: gateway-1.22.4
33a40
>         inject.istio.io/templates: gateway
38c45
<         sidecar.istio.io/inject: "false"
---
>         sidecar.istio.io/inject: "true"
51c58
<         sidecar.istio.io/inject: "false"
---
>         sidecar.istio.io/inject: "true"
64,127c71
<           env:
<             - name: PILOT_CERT_PROVIDER
<               value: istiod
<             - name: CA_ADDR
<               value: istiod.istio-system.svc:15012
<             - name: NODE_NAME
<               valueFrom:
<                 fieldRef:
<                   apiVersion: v1
<                   fieldPath: spec.nodeName
<             - name: POD_NAME
<               valueFrom:
<                 fieldRef:
<                   apiVersion: v1
<                   fieldPath: metadata.name
<             - name: POD_NAMESPACE
<               valueFrom:
<                 fieldRef:
<                   apiVersion: v1
<                   fieldPath: metadata.namespace
<             - name: INSTANCE_IP
<               valueFrom:
<                 fieldRef:
<                   apiVersion: v1
<                   fieldPath: status.podIP
<             - name: HOST_IP
<               valueFrom:
<                 fieldRef:
<                   apiVersion: v1
<                   fieldPath: status.hostIP
<             - name: ISTIO_CPU_LIMIT
<               valueFrom:
<                 resourceFieldRef:
<                   divisor: "0"
<                   resource: limits.cpu
<             - name: SERVICE_ACCOUNT
<               valueFrom:
<                 fieldRef:
<                   apiVersion: v1
<                   fieldPath: spec.serviceAccountName
<             - name: ISTIO_META_WORKLOAD_NAME
<               value: istio-ingressgateway
<             - name: ISTIO_META_OWNER
<               value: kubernetes://apis/apps/v1/namespaces/istio-system/deployments/istio-ingressgateway
<             - name: ISTIO_META_MESH_ID
<               value: nicole-dmesh
<             - name: TRUST_DOMAIN
<               value: cluster.local
<             - name: ISTIO_META_UNPRIVILEGED_POD
<               value: "true"
<             - name: ISTIO_META_DNS_AUTO_ALLOCATE
<               value: "true"
<             - name: ISTIO_META_DNS_CAPTURE
<               value: "true"
<             - name: WASM_INSECURE_REGISTRIES
<               value: '*'
<             - name: ISTIO_META_CLUSTER_ID
<               value: nicole-k2-v28-a25
<             - name: ISTIO_META_NODE_NAME
<               valueFrom:
<                 fieldRef:
<                   apiVersion: v1
<                   fieldPath: spec.nodeName
<           image: release-ci.daocloud.io/mspider/proxyv2:1.22.4
---
>           image: auto
163a108,110
>             runAsGroup: 1337
>             runAsNonRoot: true
>             runAsUser: 1337
199,200c146,150
<       serviceAccount: istio-ingressgateway-service-account
<       serviceAccountName: istio-ingressgateway-service-account
---
>         sysctls:
>           - name: net.ipv4.ip_unprivileged_port_start
>             value: "0"
>       serviceAccount: istio-ingressgateway
>       serviceAccountName: istio-ingressgateway
```

#### 网关网关，单独的 IstioOperator 自定义的网关

TODO
