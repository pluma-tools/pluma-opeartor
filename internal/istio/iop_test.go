package istio

import (
	"testing"

	"pluma.io/pluma-opeartor/config"

	"github.com/google/go-cmp/cmp"
	operatorv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

func Test_mergeIOPWithProfile(t *testing.T) {
	type args struct {
		iop string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Merge with default profile",
			args: args{
				iop: `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: nicole-dmesh-mspider-mcpc
  namespace: istio-system
spec:
  profile: default
  hub: release-ci.daocloud.io/mspider
  tag: "1.22.2"
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
      enabled: true
  meshConfig:
    defaultConfig:
      extraStatTags:
      - destination_mesh_id
      - source_mesh_id
      proxyMetadata:
        ISTIO_META_DNS_AUTO_ALLOCATE: "true"
        ISTIO_META_DNS_CAPTURE: "true"
        WASM_INSECURE_REGISTRIES: '*'
      tracing:
        sampling: 100
    enableTracing: true
    extensionProviders:
    - name: otel
      opentelemetry:
        port: 4317
        service: insight-agent-opentelemetry-collector.insight-system.svc.cluster.local
  values:
    gateways:
      istio-ingressgateway:
        autoscaleEnabled: true
        autoscaleMin: 1
    global:
      istioNamespace: istio-system
      meshID: nicole-dmesh
      multiCluster:
        clusterName: nicole-c1-k25-a24
`,
			},
			want: `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: nicole-dmesh-mspider-mcpc
  namespace: istio-system
spec:
  hub: release-ci.daocloud.io/mspider
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
      enabled: true
    egressGateways:
    - name: istio-egressgateway
      enabled: false
  meshConfig:
    defaultConfig:
      extraStatTags:
      - destination_mesh_id
      - source_mesh_id
      proxyMetadata:
        ISTIO_META_DNS_AUTO_ALLOCATE: "true"
        ISTIO_META_DNS_CAPTURE: "true"
        WASM_INSECURE_REGISTRIES: '*'
      tracing:
        sampling: 100
    enableTracing: true
    extensionProviders:
    - name: otel
      opentelemetry:
        port: 4317
        service: insight-agent-opentelemetry-collector.insight-system.svc.cluster.local
  profile: default
  tag: "1.22.2"
  values:
    defaultRevision: ""
    global:
      configValidation: true
      istioNamespace: istio-system
      meshID: nicole-dmesh
      multiCluster:
        clusterName: nicole-c1-k25-a24
    gateways:
      istio-egressgateway: {}
      istio-ingressgateway:
        autoscaleEnabled: true
        autoscaleMin: 1
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &IstioOperatorReconciler{
				Config: config.Config{
					ProfilesDir: "../../istio/profiles",
				},
			}
			var iop operatorv1alpha1.IstioOperator
			err := yaml.Unmarshal([]byte(tt.args.iop), &iop)
			if err != nil {
				t.Fatalf("Failed to unmarshal input YAML: %v", err)
			}
			got, err := r.mergeIOPWithProfile(&iop)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeIOPWithProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Convert the got IstioOperator to YAML for comparison
			gotYAML, err := yaml.Marshal(got)
			if err != nil {
				t.Fatalf("Failed to marshal got IstioOperator to YAML: %v", err)
			}

			// Convert the want YAML to IstioOperator and back to YAML for normalization
			var wantIOP operatorv1alpha1.IstioOperator
			err = yaml.Unmarshal([]byte(tt.want), &wantIOP)
			if err != nil {
				t.Fatalf("Failed to unmarshal want YAML: %v", err)
			}
			wantYAML, err := yaml.Marshal(&wantIOP)
			if err != nil {
				t.Fatalf("Failed to marshal want IstioOperator to YAML: %v", err)
			}

			if diff := cmp.Diff(string(gotYAML), string(wantYAML)); diff != "" {
				t.Errorf("mergeIOPWithProfile() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
