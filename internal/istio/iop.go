package istio

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	v1alpha12 "istio.io/api/operator/v1alpha1"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/imdario/mergo"
	structpb2 "google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"pluma.io/pluma-opeartor/config"
	"pluma.io/pluma-opeartor/internal/pkg/constants"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"pluma.io/api/operator/v1alpha1"
)

// IstioOperatorReconciler reconciles a IstioOperator object
type IstioOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config config.Config
}

// SetupWithManager sets up the controller with the Manager.
func (r *IstioOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.IstioOperator{}).
		Complete(r)
}

func (r *IstioOperatorReconciler) reconcileDelete(ctx context.Context, iop *operatorv1alpha1.IstioOperator) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Attempt to remove the HelmApp
	hApp := &v1alpha1.HelmApp{}
	err := r.Get(ctx, client.ObjectKey{Namespace: iop.GetNamespace(), Name: iop.GetName()}, hApp)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to get HelmApp: %w", err)
		}
		// HelmApp not found, proceed with finalizer removal
	} else if hApp.GetName() == iop.GetName() && hApp.Labels != nil && hApp.Labels[constants.ManagedLabel] == constants.ManagedLabelValue {
		// HelmApp found, attempt to delete it
		if err := r.Delete(ctx, hApp); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete HelmApp: %w", err)
		}
		log.Info("HelmApp deleted successfully", "HelmApp", hApp.Name)
	}

	// Remove the finalizer from the IstioOperator
	controllerutil.RemoveFinalizer(iop, constants.IOPFinalizer)
	if err := r.Update(ctx, iop); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finalizer removed successfully", "IstioOperator", iop.Name)

	return ctrl.Result{}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IstioOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the IstioOperator instance
	iop := &operatorv1alpha1.IstioOperator{}
	if err := r.Get(ctx, req.NamespacedName, iop); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if object is being deleted
	if !iop.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, iop)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(iop, constants.IOPFinalizer) {
		controllerutil.AddFinalizer(iop, constants.IOPFinalizer)
		if err := r.Update(ctx, iop); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert IstioOperator to HelmApp
	helmApp, err := r.convertIopToHelmApp(iop)
	if err != nil {
		log.Error(err, "Failed to convert IstioOperator to HelmApp")
		return ctrl.Result{}, err
	}

	// Create or update the HelmApp
	if err := r.createOrUpdateHelmApp(ctx, helmApp); err != nil {
		log.Error(err, "Failed to create or update HelmApp")
		return ctrl.Result{}, err
	}

	status := r.calculateOverallPhase(ctx, iop)
	iop.Status = &v1alpha12.InstallStatus{
		Status: status,
	}
	if err := r.Status().Update(ctx, iop); err != nil {
		return ctrl.Result{RequeueAfter: serverFailedAfter}, fmt.Errorf("failed to update iop status: %w", err)
	}

	switch status {
	case v1alpha12.InstallStatus_ERROR:
		return ctrl.Result{RequeueAfter: failedAfter}, nil
	case v1alpha12.InstallStatus_RECONCILING, v1alpha12.InstallStatus_NONE:
		return ctrl.Result{RequeueAfter: reconcileAfter}, nil
	default:
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

const (
	failedAfter       = 90 * time.Second
	serverFailedAfter = 60 * time.Second
	reconcileAfter    = 20 * time.Second
)

func (r *IstioOperatorReconciler) calculateOverallPhase(ctx context.Context, iop *operatorv1alpha1.IstioOperator) v1alpha12.InstallStatus_Status {
	if iop == nil {
		return v1alpha12.InstallStatus_NONE
	}
	helmApp := &v1alpha1.HelmApp{}
	err := r.Get(ctx, client.ObjectKey{Namespace: iop.GetNamespace(), Name: iop.GetName()}, helmApp)
	if err != nil {
		if errors.IsNotFound(err) {
			return v1alpha12.InstallStatus_RECONCILING
		}
		return v1alpha12.InstallStatus_NONE
	}

	phase := v1alpha1.Phase_UNKNOWN
	if helmApp.Status != nil {
		phase = helmApp.Status.GetPhase()
	}
	switch phase {
	case v1alpha1.Phase_UNKNOWN:
		return v1alpha12.InstallStatus_NONE
	case v1alpha1.Phase_SUCCEEDED:
		return v1alpha12.InstallStatus_HEALTHY
	case v1alpha1.Phase_FAILED:
		return v1alpha12.InstallStatus_ERROR
	default:
		return v1alpha12.InstallStatus_RECONCILING
	}
}

func structToMap(in any) map[string]interface{} {
	var res map[string]interface{}
	inStr, err := json.Marshal(in)
	if err != nil {
		fmt.Errorf("failed to marshal input to JSON: %w", err)
		return nil
	}

	if err := json.Unmarshal(inStr, &res); err != nil {
		fmt.Errorf("failed to unmarshal JSON to map: %w", err)
		return nil
	}
	return res
}

func (r *IstioOperatorReconciler) convertIopToHelmApp(in *operatorv1alpha1.IstioOperator) (*v1alpha1.HelmApp, error) {
	if in == nil || in.Spec == nil {
		return nil, fmt.Errorf("iop must required")
	}

	buildName := func(p string) string {
		return fmt.Sprintf("iop-%s-%s", in.GetName(), p)
	}
	// todo: add default config
	version := "1.21.1"
	if in.Spec.GetTag().GetStringValue() != "" {
		version = in.Spec.GetTag().GetStringValue()
	}

	// init component template
	base := &v1alpha1.HelmComponent{
		Name:    buildName("base"),
		Chart:   "base",
		Version: version,
	}
	istiodComponent := &v1alpha1.HelmComponent{
		Name:    buildName("istiod"),
		Chart:   "istiod",
		Version: version,
	}
	ingressGateway := v1alpha1.HelmComponent{
		// name need to custom defined
		Chart:   "gateway",
		Version: version,
	}
	cni := &v1alpha1.HelmComponent{}
	ztunnel := &v1alpha1.HelmComponent{}

	// merge iop profile
	iop, err := r.mergeIOPWithProfile(in)
	if err != nil {
		return nil, err
	}

	// parse global values
	var globalValues *structpb.Struct

	if iop.Spec.Values == nil {
		iop.Spec.Values = &structpb.Struct{Fields: make(map[string]*structpb.Value, 0)}
	}
	if global := iop.Spec.GetValues().GetFields()["global"]; global != nil {
		gv := global.GetStructValue()
		// render hub & tag
		if h := iop.Spec.GetHub(); h != "" {
			// Set hub in global structure
			gv.Fields["hub"] = structpb2.NewStringValue(h)
		}
	}
	if rv := iop.Spec.GetRevision(); rv != "" {
		iop.Spec.Values.Fields["revision"] = structpb2.NewStringValue(rv)
	}
	globalValues = iop.Spec.GetValues()

	// parse components
	components := make([]*v1alpha1.HelmComponent, 0)
	if iop.Spec.GetComponents() != nil {
		// base
		if iop.Spec.GetComponents().GetBase().GetEnabled().GetValue() {
			components = append(components, base)
		}

		// istiod
		if iop.Spec.GetComponents().GetPilot().GetEnabled().GetValue() {
			// Merge component-specific values
			componentValues := make(map[string]interface{})

			// mesh config
			if mc := iop.Spec.GetMeshConfig(); mc != nil {
				componentValues["meshConfig"] = structToMap(mc)
			}
			if mc := iop.Spec.GetValues().GetFields()["meshConfig"]; mc != nil {
				if componentValues["meshConfig"] == nil {
					componentValues["meshConfig"] = mc.GetStructValue().AsMap()
				} else {
					if cMC, ok := componentValues["meshConfig"].(map[string]any); ok {
						for k, v := range mc.GetStructValue().AsMap() {
							cMC[k] = v
						}
					}
				}
			}

			// sidecarInjectorWebhook
			if sw := iop.Spec.GetValues().GetFields()["sidecarInjectorWebhook"]; sw != nil {
				componentValues["sidecarInjectorWebhook"] = sw.GetStructValue().AsMap()
			}

			// Merge values from iop.Spec.Values.Pilot
			if pilot := iop.Spec.GetValues().GetFields()["pilot"]; pilot != nil {
				componentValues["pilot"] = pilot.GetStructValue().AsMap()
			}

			// Merge component-specific values
			if iopK8s := iop.Spec.GetComponents().GetPilot().GetK8S(); iopK8s != nil {
				if iopK8s.Affinity != nil {

					componentValues["affinity"] = structToMap(iopK8s.Affinity)

				}
				if iopK8s.Resources != nil {
					if componentValues["pilot"] == nil {
						componentValues["pilot"] = map[string]interface{}{
							"resources": iopK8s.Resources,
						}
					} else {
						pilot, ok := componentValues["pilot"].(map[string]any)
						if !ok {
							return nil, fmt.Errorf("componentValues['pilot'] is not a map[string]any")
						}
						pilot["resources"] = structToMap(iopK8s.Resources)
						componentValues["pilot"] = pilot
					}
				}
			}
			// Convert componentValues to structpb.Struct
			componentValuesStruct, err := structpb2.NewStruct(componentValues)
			if err != nil {
				return nil, fmt.Errorf("failed to convert component values to struct: %w", err)
			}
			istiodComponent.ComponentValues = componentValuesStruct

			components = append(components, istiodComponent)
		}

		buildGw := func(gw *v1alpha12.GatewaySpec) error {
			if !gw.GetEnabled().GetValue() {
				return nil
			}
			if gw.GetName() == "" {
				// must have name
				return nil
			}
			newGw := ingressGateway
			newGw.Name = gw.GetName()

			// Merge component-specific values
			componentValues := make(map[string]interface{})
			if gw.K8S != nil {
				if gw.K8S.Affinity != nil {
					componentValues["affinity"] = structToMap(gw.K8S.Affinity)
				}
				if gw.K8S.Resources != nil {
					componentValues["resources"] = structToMap(gw.K8S.Resources)
				}
			}

			// Merge values from iop.Spec.Values.Gateways
			if gateways := iop.Spec.GetValues().GetFields()["gateways"]; gateways != nil {
				if ingressGw := gateways.GetStructValue().GetFields()["istio-ingressgateway"]; ingressGw != nil {
					if autoscaleEnabled := ingressGw.GetStructValue().GetFields()["autoscaleEnabled"]; autoscaleEnabled != nil {
						componentValues["autoscaling"] = map[string]interface{}{
							"enabled": autoscaleEnabled.GetBoolValue(),
						}
						if autoscaleMin := ingressGw.GetStructValue().GetFields()["autoscaleMin"]; autoscaleMin != nil {
							if _, ok := componentValues["autoscaling"]; !ok {
								componentValues["autoscaling"] = make(map[string]interface{})
							}
							componentValues["autoscaling"].(map[string]interface{})["minReplicas"] = autoscaleMin.GetNumberValue()
						}
					}
				}
			}

			// Convert componentValues to structpb.Struct
			componentValuesStruct, err := structpb2.NewStruct(componentValues)
			if err != nil {
				return fmt.Errorf("failed to convert component values to struct: %w", err)
			}
			newGw.ComponentValues = componentValuesStruct
			components = append(components, &newGw)
			return nil
		}
		// ingress
		for _, gw := range iop.Spec.GetComponents().GetIngressGateways() {
			if err := buildGw(gw); err != nil {
				return nil, err
			}
		}
		for _, gw := range iop.Spec.GetComponents().GetEgressGateways() {
			if err := buildGw(gw); err != nil {
				return nil, err
			}
		}

		// cni
		if iop.Spec.GetComponents().GetCni().GetEnabled().GetValue() {
			// todo ambient
			components = append(components, cni)
		}

		// ztunnel
		if iop.Spec.GetComponents().GetZtunnel().GetEnabled().GetValue() {
			// todo ambient
			components = append(components, ztunnel)
		}
	}

	repo := "https://istio-release.storage.googleapis.com/charts"
	happ := &v1alpha1.HelmApp{
		ObjectMeta: v1.ObjectMeta{
			Name:      iop.GetName(),
			Namespace: iop.GetNamespace(),
			Labels: map[string]string{
				constants.ManagedLabel: constants.ManagedLabelValue,
				// default
				constants.AllowForceUpgradeLabel: "true",
				constants.SourceFromIOP:          fmt.Sprintf("%s", in.GetName()),
			},
		},
		Spec: &v1alpha1.HelmAppSpec{
			Components:   components,
			GlobalValues: globalValues,
			Repo: &v1alpha1.HelmRepo{
				Name: "istio",
				Url:  repo,
			},
		},
	}

	if iop.GetLabels() != nil {
		if v, ok := iop.GetLabels()[constants.AllowForceUpgradeLabel]; ok {
			happ.Labels[constants.AllowForceUpgradeLabel] = v
		}
	}

	wantYAML, err := yaml.Marshal(happ)
	if err != nil {
		fmt.Sprintf("Failed to marshal to YAML: %v", err)
	}
	fmt.Sprintf(" marshal want to YAML: %s", wantYAML)

	return happ, nil
}

func (r *IstioOperatorReconciler) createOrUpdateHelmApp(ctx context.Context, helmApp *v1alpha1.HelmApp) error {
	log := log.FromContext(ctx)

	// Check if the HelmApp already exists
	existingHelmApp := &v1alpha1.HelmApp{}
	err := r.Get(ctx, client.ObjectKey{Namespace: helmApp.Namespace, Name: helmApp.Name}, existingHelmApp)
	if err != nil {
		if errors.IsNotFound(err) {
			// HelmApp doesn't exist, create it
			log.Info("Creating new HelmApp", "namespace", helmApp.Namespace, "name", helmApp.Name)
			if err := r.Create(ctx, helmApp); err != nil {
				return fmt.Errorf("failed to create HelmApp: %w", err)
			}
			return nil
		}
		// Error reading the object - requeue the request
		return fmt.Errorf("failed to get HelmApp: %w", err)
	}

	managed := false
	if existingHelmApp.Labels != nil && existingHelmApp.Labels[constants.ManagedLabel] == constants.ManagedLabelValue {
		managed = true
	}
	// HelmApp exists, check if update is needed
	if managed && (!reflect.DeepEqual(existingHelmApp.Labels, helmApp.Labels) ||
		!reflect.DeepEqual(existingHelmApp.Spec, helmApp.Spec)) {
		log.Info("Updating existing HelmApp", "namespace", helmApp.Namespace, "name", helmApp.Name)
		existingHelmApp.Labels = helmApp.Labels
		existingHelmApp.Spec = helmApp.Spec
		if err := r.Update(ctx, existingHelmApp); err != nil {
			return fmt.Errorf("failed to update HelmApp: %w", err)
		}
	} else {
		log.Info("No changes detected, skipping update", "namespace", helmApp.Namespace, "name", helmApp.Name)
	}

	return nil
}

func (r *IstioOperatorReconciler) mergeIOPWithProfile(iop *operatorv1alpha1.IstioOperator) (*operatorv1alpha1.IstioOperator, error) {
	if iop == nil || iop.Spec == nil {
		return nil, fmt.Errorf("input IstioOperator is nil or has nil Spec")
	}

	profileName := "default"
	if iop.Spec.Profile != "" {
		profileName = iop.Spec.Profile
	}

	// Read the profile file
	profilePath := fmt.Sprintf("%s/%s.yaml", r.Config.ProfilesDir, profileName)
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file %s: %w", profilePath, err)
	}

	// Unmarshal the profile into an IstioOperator
	profileIOP := &operatorv1alpha1.IstioOperator{}
	if err := yaml.Unmarshal(profileData, profileIOP); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile data: %w", err)
	}

	// Merge the profile IOP with the input IOP
	mergedIOP := profileIOP.DeepCopy()
	if err := mergo.Merge(iop, mergedIOP); err != nil {
		return nil, fmt.Errorf("failed to merge IOPs: %w", err)
	}

	// todo remove: Test
	wantYAML, err := yaml.Marshal(iop)
	if err != nil {
		fmt.Sprintf("Failed to marshal to YAML: %v", err)
	}
	fmt.Sprintf(" marshal want to YAML: %s", wantYAML)

	return iop.DeepCopy(), nil
}
