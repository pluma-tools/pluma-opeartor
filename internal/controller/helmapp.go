package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	helmaction "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmcli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	operatorv1alpha1 "pluma.io/api/operator/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctllog "sigs.k8s.io/controller-runtime/pkg/log"
)

var settings = newHelmSettings()

func init() {
	log.SetFlags(log.Lshortfile)
}

func debug(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func warning(format string, v ...interface{}) {
	format = fmt.Sprintf("WARNING: %s\n", format)
	fmt.Fprintf(os.Stderr, format, v...)
}

// HelmAppReconciler reconciles a HelmApp object
type HelmAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.HelmApp{}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *HelmAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cLog := ctllog.FromContext(ctx)

	// Fetch the HelmApp instance
	helmApp := &operatorv1alpha1.HelmApp{}
	if err := r.Get(ctx, req.NamespacedName, helmApp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the HelmApp is being deleted
	if !helmApp.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, helmApp)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(helmApp, "helmapp.pluma.io/finalizer") {
		controllerutil.AddFinalizer(helmApp, "helmapp.pluma.io/finalizer")
		if err := r.Update(ctx, helmApp); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Initialize Helm client
	helmCfg, err := newActionConfiguration()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to new Helm action config: %v", err)
	}

	// Get the local kubeconfig
	restClientGetter := genericclioptions.NewConfigFlags(true)
	if err := helmCfg.Init(restClientGetter, helmApp.Namespace, "", debug); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	// Create a map of desired components
	desiredComponents := make(map[string]*operatorv1alpha1.HelmComponent)
	for _, component := range helmApp.Spec.Components {
		desiredComponents[component.Name] = component
	}

	// Process each component
	var componentStatuses []*operatorv1alpha1.HelmComponentStatus
	for _, component := range helmApp.Spec.Components {
		status, err := r.reconcileComponent(ctx, helmApp, component, helmCfg)
		if err != nil {
			cLog.Error(err, fmt.Sprintf("Failed to reconcile component %s", component.Name))
		}
		componentStatuses = append(componentStatuses, status)
	}

	// Uninstall components that are no longer in the spec
	if helmApp.Status != nil {
		for _, existingStatus := range helmApp.Status.Components {
			if _, exists := desiredComponents[existingStatus.Name]; !exists {
				if err := r.uninstallComponent(ctx, existingStatus.Name, helmCfg); err != nil {
					cLog.Error(err, fmt.Sprintf("Failed to uninstall component %s", existingStatus.Name))
					// Update component status with error message
					componentStatuses = append(componentStatuses, &operatorv1alpha1.HelmComponentStatus{
						Name:           existingStatus.Name,
						Status:         helmrelease.StatusFailed.String(),
						Message:        fmt.Sprintf("uninstall %s error: %v", existingStatus.Name, err),
						Version:        existingStatus.Version,
						Resources:      existingStatus.Resources,
						ResourcesTotal: existingStatus.ResourcesTotal,
					})
				} else {
					cLog.Info("Uninstalled component", "component", existingStatus.Name)
				}
			}
		}
	}

	// Update HelmApp status
	if helmApp.Status == nil {
		helmApp.Status = &operatorv1alpha1.HelmAppStatus{}
	}
	helmApp.Status.Components = componentStatuses

	// Calculate overall phase based on component statuses
	overallPhase := calculateOverallPhase(helmApp, componentStatuses)
	helmApp.Status.Phase = overallPhase

	if err := r.Status().Update(ctx, helmApp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update HelmApp status: %w", err)
	}

	// If the phase is FAILED, requeue after 3 minutes
	if overallPhase == operatorv1alpha1.Phase_FAILED {
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil
	}

	return ctrl.Result{}, nil
}

func calculateOverallPhase(helmApp *operatorv1alpha1.HelmApp, componentStatuses []*operatorv1alpha1.HelmComponentStatus) operatorv1alpha1.Phase {
	if len(componentStatuses) == 0 {
		return operatorv1alpha1.Phase_UNKNOWN
	}
	if !helmApp.ObjectMeta.DeletionTimestamp.IsZero() {
		return operatorv1alpha1.Phase_DELETING
	}

	hasFailure := false
	allDeployed := true

	for _, status := range componentStatuses {
		switch status.Status {
		case helmrelease.StatusFailed.String():
			hasFailure = true
		case helmrelease.StatusDeployed.String():
			// Do nothing, it's good
		default:
			allDeployed = false
		}
	}

	if hasFailure {
		return operatorv1alpha1.Phase_FAILED
	}
	if allDeployed {
		return operatorv1alpha1.Phase_SUCCEEDED
	}
	return operatorv1alpha1.Phase_RECONCILING
}

func (r *HelmAppReconciler) reconcileDelete(ctx context.Context, helmApp *operatorv1alpha1.HelmApp) (ctrl.Result, error) {
	cLog := ctllog.FromContext(ctx)

	// Initialize Helm client
	helmCfg, err := newActionConfiguration()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to new Helm action config: %v", err)
	}

	// Get the local kubeconfig
	restClientGetter := genericclioptions.NewConfigFlags(true)
	if err := helmCfg.Init(restClientGetter, helmApp.Namespace, "", debug); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	// Uninstall all components in reverse order
	allComponentsUninstalled := true
	if helmApp.Status != nil && len(helmApp.Status.Components) > 0 {
		for i := len(helmApp.Status.Components) - 1; i >= 0; i-- {
			component := helmApp.Status.Components[i]
			if err := r.uninstallComponent(ctx, component.Name, helmCfg); err != nil {
				cLog.Error(err, fmt.Sprintf("Failed to uninstall component %s during deletion", component.Name))
				allComponentsUninstalled = false

				err = fmt.Errorf("uninstall %s error: %v", component.Name, err)
				// Update component status with error message
				helmApp.Status.Components[i].Status = helmrelease.StatusFailed.String()
				helmApp.Status.Components[i].Message = err.Error()
			} else {
				cLog.Info("Uninstalled component during deletion", "component", component.Name)
				// Remove the uninstalled component from the status
				helmApp.Status.Components = append(helmApp.Status.Components[:i], helmApp.Status.Components[i+1:]...)
			}
		}
	}

	// Update HelmApp status
	helmApp.Status.Phase = calculateOverallPhase(helmApp, helmApp.Status.Components)
	if err := r.Status().Update(ctx, helmApp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update HelmApp status: %w", err)
	}

	// Remove finalizer only if all components are uninstalled
	if allComponentsUninstalled {
		controllerutil.RemoveFinalizer(helmApp, "helmapp.pluma.io/finalizer")
		if err := r.Update(ctx, helmApp); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Requeue if not all components are uninstalled
	return ctrl.Result{Requeue: true}, nil
}

func newActionConfiguration() (*helmaction.Configuration, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
	}

	// Create a new registry client
	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("initializing new helm registry client: %s", err)
	}
	return &helmaction.Configuration{
		RegistryClient: registryClient,
	}, nil
}

func newHelmSettings() *helmcli.EnvSettings {
	helmSettings := helmcli.New()
	helmSettings.Debug = true

	return helmSettings
}

func (r *HelmAppReconciler) reconcileComponent(ctx context.Context, helmApp *operatorv1alpha1.HelmApp, component *operatorv1alpha1.HelmComponent,
	helmCfg *helmaction.Configuration) (*operatorv1alpha1.HelmComponentStatus, error) {
	cLog := ctllog.FromContext(ctx)

	values := MergeValues(helmApp.Spec.GlobalValues.AsMap(), component.ComponentValues.AsMap())

	// Create a new install action
	install := helmaction.NewInstall(helmCfg)
	install.Namespace = helmApp.Namespace
	install.ReleaseName = component.Name
	install.Version = component.Version
	install.RepoURL = helmApp.Spec.Repo.Url
	install.ChartPathOptions.RepoURL = helmApp.Spec.Repo.Url

	// Locate the chart
	cp, err := install.ChartPathOptions.LocateChart(component.Chart, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	// Load Chart
	chart, err := loader.Load(cp)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	// Install or upgrade the release
	var release *helmrelease.Release
	mErrs := &multierror.Error{}

	histClient := helmaction.NewHistory(helmCfg)
	histClient.Max = 1
	history, err := histClient.Run(component.Name)
	switch {
	case errors.Is(err, driver.ErrReleaseNotFound):
		// Release doesn't exist, install it
		release, err = install.Run(chart, values)
		if err != nil {
			cLog.Error(err, "failed to install release")
			multierror.Append(mErrs, fmt.Errorf("failed to install release: %v", err))
		}
		cLog.Info("Installed release", "component", component.Name)
	case err == nil:
		// Release exists, check if update is needed
		if len(history) > 0 && !hasConfigChanged(history[len(history)-1], values, component.Version) {
			cLog.Info("No changes detected, skipping upgrade", "component", component.Name)
			release = history[0]
		} else {
			// Upgrade the release
			upgrade := helmaction.NewUpgrade(helmCfg)
			upgrade.Namespace = helmApp.Namespace
			upgrade.RepoURL = helmApp.Spec.Repo.Url
			upgrade.Version = component.Version
			release, err = upgrade.Run(component.Name, chart, values)
			if err != nil {
				cLog.Error(err, "failed to upgrade release")
				multierror.Append(mErrs, fmt.Errorf("failed to upgrade release: %v", err))
			}
			cLog.Info("Upgraded release", "component", component.Name)
		}
	default:
		cLog.Error(err, "helm releases history")
		multierror.Append(mErrs, fmt.Errorf("helm releases history: %v", err))
	}

	version := "unknown"
	status := "unknown"
	msg := ""
	var resourcesStatus []*operatorv1alpha1.HelmResourceStatus
	resourcesTotal := 0
	if release != nil {
		version = strconv.Itoa(release.Version)
		status = release.Info.Status.String()

		// Parse the release manifest to get resource statuses
		resources, err := resource.NewBuilder(helmCfg.RESTClientGetter).
			Unstructured().
			Stream(bytes.NewBufferString(release.Manifest), "").
			Do().Infos()
		if err != nil {
			cLog.Error(err, "failed to parse release manifest")
		} else {
			resourcesTotal = len(resources)
			for _, r := range resources {
				resourceStatus := &operatorv1alpha1.HelmResourceStatus{
					ApiVersion: r.Mapping.GroupVersionKind.GroupVersion().String(),
					Kind:       r.Mapping.GroupVersionKind.Kind,
					Name:       r.Name,
					Namespace:  r.Namespace,
				}
				resourcesStatus = append(resourcesStatus, resourceStatus)
			}
		}
	}
	if mErrs.ErrorOrNil() != nil {
		msg = mErrs.Error()
	}
	// Create component status
	componentStatus := &operatorv1alpha1.HelmComponentStatus{
		Name:           component.GetName(),
		Version:        version,
		Status:         status,
		Message:        msg,
		Resources:      resourcesStatus,
		ResourcesTotal: int32(resourcesTotal),
	}

	return componentStatus, mErrs.ErrorOrNil()
}

func (r *HelmAppReconciler) uninstallComponent(ctx context.Context, componentName string, helmCfg *helmaction.Configuration) error {
	cLog := ctllog.FromContext(ctx)

	uninstall := helmaction.NewUninstall(helmCfg)
	_, err := uninstall.Run(componentName)
	if err != nil && !errors.Is(err, driver.ErrReleaseNotFound) {
		cLog.Error(err, fmt.Sprintf("Failed to uninstall component %s", componentName))
		return err
	}

	return nil
}

// MergeValues merges two maps of values, with values from b taking precedence.
func MergeValues(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]any); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]any); ok {
					out[k] = MergeValues(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func hasConfigChanged(release *helmrelease.Release, newValues map[string]any, newVersion string) bool {
	if release.Chart.Metadata.Version != newVersion {
		return true
	}
	return !reflect.DeepEqual(release.Config, newValues)
}
