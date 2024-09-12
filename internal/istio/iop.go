package istio

//
//import (
//	"context"
//	"fmt"
//	"log"
//
//	structpb "github.com/golang/protobuf/ptypes/struct"
//	iopv1alpha1 "istio.io/istio/operator/pkg/apis"
//	"k8s.io/apimachinery/pkg/runtime"
//	"k8s.io/client-go/tools/record"
//	ctrl "sigs.k8s.io/controller-runtime"
//	"sigs.k8s.io/controller-runtime/pkg/client"
//)
//
//// IOPReconciler reconciles a IstioOperator object
//type IOPReconciler struct {
//	client.Client
//	Scheme   *runtime.Scheme
//	Recorder record.EventRecorder
//}
//
//// SetupWithManager sets up the controller with the Manager.
//func (r *IOPReconciler) SetupWithManager(mgr ctrl.Manager) error {
//	return ctrl.NewControllerManagedBy(mgr).
//		For(&iopv1alpha1.IstioOperator{}).
//		Complete(r)
//}
//
//// Reconcile is part of the main kubernetes reconciliation loop which aims to
//// move the current state of the cluster closer to the desired state.
//func (r *IOPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//	log.Printf("Reconciling IstioOperator %s/%s", req.Namespace, req.Name)
//
//	// Fetch the IstioOperator instance
//	instance := &iopv1alpha1.IstioOperator{}
//	err := r.Get(ctx, req.NamespacedName, instance)
//	if err != nil {
//		return ctrl.Result{}, client.IgnoreNotFound(err)
//	}
//
//	// Convert IstioOperator to Helm values
//	helmValues, err := convertToHelmValues(instance)
//	if err != nil {
//		return ctrl.Result{}, fmt.Errorf("failed to convert IstioOperator to Helm values: %w", err)
//	}
//
//	// Initialize HelmInstaller
//	helmInstaller, err := NewHelmInstaller("") // Assuming kubeconfig is not needed or handled elsewhere
//	if err != nil {
//		return ctrl.Result{}, fmt.Errorf("failed to initialize HelmInstaller: %w", err)
//	}
//
//	// Install or upgrade Helm release
//	releaseName := fmt.Sprintf("istio-%s", instance.Name)
//	chartName := "istio/base" // Update this with the actual chart name for Istio
//	chartVersion := ""        // Specify a version if needed, or leave empty for latest
//
//	err = helmInstaller.InstallComponent(releaseName, chartName, chartVersion, helmValues)
//	if err != nil {
//		return ctrl.Result{}, fmt.Errorf("failed to install/upgrade Helm release: %w", err)
//	}
//
//	log.Printf("Successfully reconciled IstioOperator %s/%s", req.Namespace, req.Name)
//	return ctrl.Result{}, nil
//}
//
//// convertToHelmValues converts an IstioOperator spec to Helm values
////
///*
//iop sample：
//spec:
//  components:
//    ingressGateways:
//    - k8s:
//        resources:
//          limits:
//            cpu: 1000m
//            memory: 900Mi
//          requests:
//            cpu: 50m
//            memory: 50Mi
//      name: istio-ingressgateway
//    pilot:
//      k8s:
//        resources:
//          limits:
//            cpu: 1500m
//            memory: 1500Mi
//          requests:
//            cpu: 200m
//            memory: 200Mi
//  hub: release-ci.daocloud.io/mspider
//  meshConfig:
//    defaultConfig:
//      extraStatTags:
//      - destination_mesh_id
//      - source_mesh_id
//      proxyMetadata:
//        ISTIO_META_DNS_AUTO_ALLOCATE: "true"
//        ISTIO_META_DNS_CAPTURE: "true"
//        WASM_INSECURE_REGISTRIES: '*'
//      tracing:
//        sampling: 100
//    enableTracing: true
//    extensionProviders:
//    - name: otel
//      opentelemetry:
//        port: 4317
//        service: insight-agent-opentelemetry-collector.insight-system.svc.cluster.local
//  namespace: istio-system
//  profile: default
//  tag: 1.21.1-mspider
//  values:
//    gateways:
//      istio-ingressgateway:
//        autoscaleEnabled: true
//        autoscaleMin: 1
//      securityContext:
//        sysctls: []
//    global:
//      istioNamespace: istio-system
//      meshID: mspider-dedicated
//      multiCluster:
//        clusterName: mspider-dedicated
//      network: internal-net
//      proxy:
//        logLevel: warning
//        resources:
//          limits:
//            cpu: 600m
//            memory: 200Mi
//          requests:
//            cpu: 100m
//            memory: 20Mi
//    meshConfig:
//      enableTracing: true
//      outboundTrafficPolicy:
//        mode: ALLOW_ANY
//    pilot:
//      autoscaleEnabled: true
//      autoscaleMin: 1
//      replicaCount: 1
//    sidecarInjectorWebhook:
//      enableNamespacesByDefault: false
//
//Convert to helm values:
//global:
//  hub: release-ci.daocloud.io/mspider
//  tag: 1.21.1-mspider
//  istioNamespace: istio-system
//
//  meshID: mspider-dedicated
//  multiCluster:
//    clusterName: mspider-dedicated
//
//
//  proxy:
//    logLevel: warning
//    resources:
//      limits:
//        cpu: 600m
//        memory: 512Mi
//      requests:
//        cpu: 100m
//        memory: 128Mi
//
//  sidecarInjectorWebhook:
//    enableNamespacesByDefault: false
//
//base:
//  enabled: true
//# Overrides for the `istiod-remote` dep
//
//
//istiod:
//  enabled: true
//	resources:
//		limits:
//			cpu: 1500m
//			memory: 1500Mi
//		requests:
//			cpu: 200m
//			memory: 200Mi
//  autoscaleEnabled: true
//  autoscaleMin: 1
//  replicaCount: 1
//  sidecarInjectorWebhook:
//    enableNamespacesByDefault: false
//  meshConfig:
//    outboundTrafficPolicy:
//      mode: ALLOW_ANY
//    defaultConfig:
//      extraStatTags:
//        - destination_mesh_id
//        - source_mesh_id
//      proxyMetadata:
//        ISTIO_META_DNS_AUTO_ALLOCATE: 'true'
//        ISTIO_META_DNS_CAPTURE: 'true'
//        WASM_INSECURE_REGISTRIES: '*'
//      tracing:
//        sampling: 100
//    enableTracing: true
//    extensionProviders:
//      - name: otel
//        opentelemetry:
//          port: 4317
//          service: >-
//            insight-agent-opentelemetry-collector.insight-system.svc.cluster.local
//
//istio-ingress:
//  enabled: true
//  gateways:
//    istio-ingressgateway:
//      name: istio-ingressgateway
//      labels:
//        app: istio-ingressgateway
//        istio: ingressgateway
//			resources:
//				limits:
//					cpu: 1000m
//					memory: 900Mi
//				requests:
//					cpu: 50m
//					memory: 50Mi
//
//
//# Overrides for the `ztunnel` dep
//ztunnel:
//  enabled: false
//  profile: ambient
//
//# Overrides for the `cni` dep
//cni:
//  enabled: false
//  profile: ambient
//
//*/
//func convertToHelmValues(iop *iopv1alpha1.IstioOperator) (map[string]interface{}, error) {
//	// Start with default values
//	values := map[string]interface{}{
//		"global": map[string]interface{}{},
//		"base": map[string]interface{}{
//			"enabled": true,
//		},
//		"istiod": map[string]interface{}{
//			"enabled": false,
//		},
//		"istio-ingress": map[string]interface{}{
//			"enabled": true,
//		},
//		"ztunnel": map[string]interface{}{
//			"enabled": false,
//		},
//		"cni": map[string]interface{}{
//			"enabled": false,
//		},
//	}
//
//	// Merge IstioOperator spec into values
//	if err := mergeValues(values, iop); err != nil {
//		return nil, fmt.Errorf("failed to merge IstioOperator values: %w", err)
//	}
//
//	return values, nil
//}
//
//func mergeValues(values map[string]interface{}, iop *iopv1alpha1.IstioOperator) error {
//	// Merge global values
//	global := values["global"].(map[string]interface{})
//
//	// First, copy spec.values.global to helm global
//	if iop.Spec.Values != nil && iop.Spec.Values.Fields["global"] != nil {
//		globalStruct := iop.Spec.Values.Fields["global"].GetStructValue()
//		if err := mergeStructPB(global, globalStruct); err != nil {
//			return fmt.Errorf("failed to merge spec.values.global: %w", err)
//		}
//	}
//
//	// Then, set other global values
//	global["hub"] = iop.Spec.Hub
//	global["tag"] = iop.Spec.Tag
//	global["istioNamespace"] = iop.Namespace
//
//	// Merge meshConfig
//	if iop.Spec.MeshConfig != nil {
//		values["istiod"].(map[string]interface{})["meshConfig"] = iop.Spec.MeshConfig
//	}
//
//	// Merge components
//	if iop.Spec.Components != nil {
//		if iop.Spec.Components.Pilot != nil && iop.Spec.Components.Pilot.K8S != nil {
//			values["istiod"].(map[string]interface{})["resources"] = iop.Spec.Components.Pilot.K8S.Resources
//		}
//		if len(iop.Spec.Components.IngressGateways) > 0 {
//			gateways := make(map[string]interface{})
//			for _, gw := range iop.Spec.Components.IngressGateways {
//				gateway := map[string]interface{}{
//					"name": gw.Name,
//				}
//				if gw.K8S != nil && gw.K8S.Resources != nil {
//					gateway["resources"] = gw.K8S.Resources
//				}
//				gateways[gw.Name] = gateway
//			}
//			values["istio-ingress"].(map[string]interface{})["gateways"] = gateways
//		}
//	}
//
//	// Merge additional values from IstioOperator
//	if iop.Spec.Values != nil {
//		if err := mergeStructPB(values, iop.Spec.Values); err != nil {
//			return fmt.Errorf("failed to merge additional values: %w", err)
//		}
//	}
//
//	return nil
//}
//
//func mergeStructPB(dst map[string]interface{}, src *structpb.Struct) error {
//	for k, v := range src.Fields {
//		switch v.Kind.(type) {
//		case *structpb.Value_StructValue:
//			if _, ok := dst[k]; !ok {
//				dst[k] = make(map[string]interface{})
//			}
//			if dstMap, ok := dst[k].(map[string]interface{}); ok {
//				if err := mergeStructPB(dstMap, v.GetStructValue()); err != nil {
//					return err
//				}
//			} else {
//				return fmt.Errorf("cannot merge struct into non-map value for key %s", k)
//			}
//		default:
//			dst[k] = v.AsInterface()
//		}
//	}
//	return nil
//}
