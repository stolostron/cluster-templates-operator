/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubernetes-client/go-base/config/api"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/argocd"
	ocm "github.com/stolostron/cluster-templates-operator/ocm"

	"github.com/stolostron/cluster-templates-operator/clusterprovider"
	"github.com/stolostron/cluster-templates-operator/clustersetup"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/v1beta1"
	agent "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	ocmv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	CTIlog = logf.Log.WithName("cti-controller")
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type Clock interface {
	Now() time.Time
}

type ClusterTemplateInstanceReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	EnableHypershift     bool
	EnableHive           bool
	EnableManagedCluster bool
	EnableKlusterlet     bool
	Clock
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplatesetup/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplatesetup,verbs=get;list;watch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplatequotas,verbs=get;list;watch
// +kubebuilder:rbac:groups=hypershift.openshift.io,resources=hostedclusters;nodepools,verbs=get;list;watch
// +kubebuilder:rbac:groups=hive.openshift.io,resources=clusterclaims;clusterdeployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;roles,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=managedclusters,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=managedclustersets/join,verbs=create
// +kubebuilder:rbac:groups=register.open-cluster-management.io,resources=managedclusters/accept,verbs=update
// +kubebuilder:rbac:groups=agent.open-cluster-management.io,resources=klusterletaddonconfigs,verbs=get;list;watch;create;delete

func (r *ClusterTemplateInstanceReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	clusterTemplateInstance := &v1alpha1.ClusterTemplateInstance{}
	if err := r.Get(ctx, req.NamespacedName, clusterTemplateInstance); err != nil {
		if apierrors.IsNotFound(err) {
			CTIlog.Info(
				"clustertemplateinstance not found, aborting reconcile",
				"name",
				req.NamespacedName,
			)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if len(clusterTemplateInstance.Status.Conditions) == 0 {
		clusterTemplateInstance.Status.Phase = v1alpha1.PendingPhase
		clusterTemplateInstance.Status.Message = v1alpha1.PendingMessage
	}
	clusterTemplateInstance.SetDefaultConditions()

	if clusterTemplateInstance.GetDeletionTimestamp() != nil {
		return r.delete(ctx, clusterTemplateInstance)
	}

	// Check if CTI should be auto-removed:
	requeueAfter, err := r.autoDelete(ctx, clusterTemplateInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	var clusterTemplate client.Object
	if clusterTemplateInstance.Spec.KubeconfigSecretRef != nil {
		clusterTemplate = &v1alpha1.ClusterTemplateSetup{}
	} else {
		clusterTemplate = &v1alpha1.ClusterTemplate{}
	}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: clusterTemplateInstance.Spec.ClusterTemplateRef}, clusterTemplate); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.FailedPhase
		clusterTemplateInstance.Status.Message = fmt.Sprintf("failed to fetch ClusterTemplate - %q", err)
		if updErr := r.Status().Update(ctx, clusterTemplateInstance); updErr != nil {
			return ctrl.Result{}, fmt.Errorf(
				"failed to update status of clustertemplateinstance %q: %w",
				req.NamespacedName,
				updErr,
			)
		}

		return ctrl.Result{}, err
	}

	err = r.reconcile(ctx, clusterTemplateInstance, clusterTemplate)
	if updErr := r.Status().Update(ctx, clusterTemplateInstance); updErr != nil {
		return ctrl.Result{}, fmt.Errorf(
			"failed to update status of clustertemplateinstance %q: %w",
			req.NamespacedName,
			updErr,
		)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if requeueAfter != nil {
		return ctrl.Result{RequeueAfter: *requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

// Return the amount of time the reconcile should re-queued to check the delete time,
// if time already passed remove the CTI.
func (r *ClusterTemplateInstanceReconciler) autoDelete(ctx context.Context, cti *v1alpha1.ClusterTemplateInstance) (*time.Duration, error) {
	ctqList := &v1alpha1.ClusterTemplateQuotaList{}
	if err := r.List(ctx, ctqList); err != nil {
		return nil, err
	}

	var deleteAfter *time.Duration
	for _, ctq := range ctqList.Items {
		for _, allowedTemplate := range ctq.Spec.AllowedTemplates {
			if cti.Spec.ClusterTemplateRef == allowedTemplate.Name {
				if allowedTemplate.DeleteAfter != nil {
					deleteAfter = &allowedTemplate.DeleteAfter.Duration
				}
				break
			}
		}
	}
	if deleteAfter == nil {
		return nil, nil
	}

	now := r.Now()
	createTimestamp := cti.CreationTimestamp
	if now.After(createTimestamp.Add(*deleteAfter)) {
		CTIlog.Info("Removing CTI as time to live expired", "name", cti.Name)
		if err := r.Delete(ctx, cti); err != nil {
			return nil, err
		}
	} else {
		requeueAfter := createTimestamp.Add(*deleteAfter).Sub(now)
		return &requeueAfter, nil
	}

	return nil, nil
}

func (r *ClusterTemplateInstanceReconciler) delete(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) (ctrl.Result, error) {
	if len(clusterTemplateInstance.Finalizers) != 1 || !controllerutil.ContainsFinalizer(
		clusterTemplateInstance,
		v1alpha1.CTIFinalizer,
	) {
		return ctrl.Result{}, nil
	}
	var clusterTemplate client.Object
	if clusterTemplateInstance.Spec.KubeconfigSecretRef != nil {
		clusterTemplate = &v1alpha1.ClusterTemplateSetup{}
	} else {
		clusterTemplate = &v1alpha1.ClusterTemplate{}
	}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: clusterTemplateInstance.Spec.ClusterTemplateRef}, clusterTemplate); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	} else {
		if ct, ok := clusterTemplate.(*v1alpha1.ClusterTemplate); ok {
			err := clusterTemplateInstance.DeleteDay1Application(ctx, r.Client, ArgoCDNamespace, ct.Spec.ClusterDefinition)
			if err != nil {
				return ctrl.Result{}, err
			}
			err = clusterTemplateInstance.DeleteDay2Application(ctx, r.Client, ArgoCDNamespace, ct.Spec.ClusterSetup)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			err = clusterTemplateInstance.DeleteDay2Application(ctx, r.Client, ArgoCDNamespace, clusterTemplate.(*v1alpha1.ClusterTemplateSetup).Spec.ClusterSetup)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// cleanup argocd secrets (ie new cluster)
	ctiNameLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINameLabel,
		selection.Equals,
		[]string{clusterTemplateInstance.Name},
	)
	ctiNsLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINamespaceLabel,
		selection.Equals,
		[]string{clusterTemplateInstance.Namespace},
	)
	selector := labels.NewSelector().Add(*ctiNameLabelReq, *ctiNsLabelReq)
	secrets := &corev1.SecretList{}
	if err := r.Client.List(ctx, secrets, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     ArgoCDNamespace,
	}); err != nil {
		return ctrl.Result{}, err
	}

	for _, secret := range secrets.Items {
		if err := r.Client.Delete(ctx, &secret); err != nil {
			return ctrl.Result{}, err
		}
	}

	if r.EnableManagedCluster {
		mc, err := ocm.GetManagedCluster(ctx, r.Client, clusterTemplateInstance)
		if err != nil {
			_, ok := err.(*ocm.MCNotFoundError)
			if !ok {
				return ctrl.Result{}, err
			}
		}
		if mc != nil {
			if r.EnableKlusterlet {
				klusterlet := &agent.KlusterletAddonConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mc.Name,
						Namespace: mc.Name,
					},
				}
				if err := r.Client.Delete(ctx, klusterlet); err != nil && !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
			importSecret := &corev1.Secret{
				ObjectMeta: ocm.GetImportSecretMeta(mc.Name),
			}
			if err := r.Client.Delete(ctx, importSecret); err != nil {
				if !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
			if err := r.Client.Delete(ctx, mc); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	controllerutil.RemoveFinalizer(
		clusterTemplateInstance,
		v1alpha1.CTIFinalizer,
	)
	err := r.Update(ctx, clusterTemplateInstance)
	return ctrl.Result{}, err
}

func getClusterProperties(clusterTemplate client.Object) (bool, string, []string) {
	var skipClusterRegistration bool
	var clusterDefinition string
	var clusterSetup []string
	switch clusterTemplate.(type) {
	case *v1alpha1.ClusterTemplateSetup:
		skipClusterRegistration = clusterTemplate.(*v1alpha1.ClusterTemplateSetup).Spec.SkipClusterRegistration
		clusterSetup = clusterTemplate.(*v1alpha1.ClusterTemplateSetup).Spec.ClusterSetup
	case *v1alpha1.ClusterTemplate:
		skipClusterRegistration = clusterTemplate.(*v1alpha1.ClusterTemplate).Spec.SkipClusterRegistration
		clusterDefinition = clusterTemplate.(*v1alpha1.ClusterTemplate).Spec.ClusterDefinition
		clusterSetup = clusterTemplate.(*v1alpha1.ClusterTemplate).Spec.ClusterSetup
	}

	return skipClusterRegistration, clusterDefinition, clusterSetup
}

func (r *ClusterTemplateInstanceReconciler) reconcile(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterTemplate client.Object,
) error {
	skipClusterRegistration, clusterDefinition, clusterSetup := getClusterProperties(clusterTemplate)
	if clusterTemplateInstance.Spec.KubeconfigSecretRef == nil {
		if err := r.reconcileClusterCreate(ctx, clusterTemplateInstance, clusterDefinition); err != nil {
			clusterTemplateInstance.Status.Phase = v1alpha1.ClusterDefinitionFailedPhase
			errMsg := fmt.Sprintf("failed to create cluster definition - %q", err)
			clusterTemplateInstance.Status.Message = errMsg
			return fmt.Errorf(errMsg)
		}
		if err := r.reconcileClusterStatus(
			ctx,
			clusterTemplateInstance,
		); err != nil {
			clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallFailedPhase
			errMsg := fmt.Sprintf("failed to reconcile cluster status - %q", err)
			clusterTemplateInstance.Status.Message = errMsg
			return fmt.Errorf(errMsg)
		}
	} else {
		secret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: *clusterTemplateInstance.Spec.KubeconfigSecretRef, Namespace: clusterTemplateInstance.Namespace}, secret); err != nil {
			return err
		}
		kubeconfig, okKubeconfig := secret.Data["kubeconfig"]
		if !okKubeconfig {
			return fmt.Errorf("kubeconfig not found in secret %q", *clusterTemplateInstance.Spec.KubeconfigSecretRef)
		}
		password := secret.Data["password"]
		if err := clusterprovider.CreateClusterSecrets(
			ctx,
			r.Client,
			kubeconfig,
			[]byte("kubeadmin"),
			password,
			*clusterTemplateInstance,
		); err != nil {
			return err
		}
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionTrue,
			v1alpha1.ClusterInstalled,
			"Cluster defined via secret",
		)
	}

	//ACM integration

	if err := r.reconcileCreateManagedCluster(ctx, clusterTemplateInstance, skipClusterRegistration, clusterTemplate.GetLabels()); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.ManagedClusterFailedPhase
		errMsg := fmt.Sprintf("failed to create ManagedCluster - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileImportManagedCluster(ctx, clusterTemplateInstance, skipClusterRegistration); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.ManagedClusterImportFailedPhase
		errMsg := fmt.Sprintf("failed to import ManagedCluster - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileCreateKlusterlet(ctx, clusterTemplateInstance, skipClusterRegistration); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.KlusterletCreateFailedPhase
		errMsg := fmt.Sprintf("failed to create Klusterlet - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileConsoleURL(ctx, clusterTemplateInstance, skipClusterRegistration); err != nil {
		return fmt.Errorf("failed to retrieve Console URL - %q", err)
	}

	//

	if err := r.reconcileAddClusterToArgo(ctx, clusterTemplateInstance, skipClusterRegistration); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.ArgoClusterFailedPhase
		errMsg := fmt.Sprintf("failed to add cluster to argo - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileClusterSetupCreate(ctx, clusterTemplateInstance, clusterSetup); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterSetupCreateFailedPhase
		errMsg := fmt.Sprintf("failed to create cluster setup - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileClusterSetup(ctx, clusterTemplateInstance, clusterSetup); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterSetupFailedPhase
		errMsg := fmt.Sprintf("failed to reconcile cluster setup - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileClusterCredentials(ctx, clusterTemplateInstance); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf("failed to reconcile cluster credentials - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCreate(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterDefinition string,
) error {
	clusterDefinitionCreatedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterDefinitionCreated),
	)

	if clusterDefinitionCreatedCondition.Status == metav1.ConditionFalse {
		if err := clusterTemplateInstance.CreateDay1Application(ctx, r.Client, ArgoCDNamespace, ArgoCDNamespace == defaultArgoCDNs, clusterDefinition); err != nil {
			clusterTemplateInstance.SetClusterDefinitionCreatedCondition(
				metav1.ConditionFalse,
				v1alpha1.ClusterDefinitionFailed,
				fmt.Sprintf("Failed to create application - %q", err),
			)
			return err
		}
		clusterTemplateInstance.SetClusterDefinitionCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.ApplicationCreated,
			"Application created",
		)
	}
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterStatus(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	CTIlog.Info(
		"Reconcile instance status",
		"name",
		clusterTemplateInstance.Namespace+"/"+clusterTemplateInstance.Name,
	)
	appCreatedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterDefinitionCreated),
	)
	if appCreatedCondition.Status == metav1.ConditionFalse {
		return nil
	}

	CTIlog.Info(
		"Fetch day1 argo application",
		"name",
		clusterTemplateInstance.Namespace+"/"+clusterTemplateInstance.Name,
	)
	application, err := clusterTemplateInstance.GetDay1Application(ctx, r.Client, ArgoCDNamespace)

	if err != nil {
		failedMsg := fmt.Sprintf("Failed to fetch application - %q", err)
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ApplicationFetchFailed,
			failedMsg,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallFailedPhase
		clusterTemplateInstance.Status.Message = failedMsg
		return err
	}

	appHealth, msg := argocd.GetApplicationHealth(application, false)
	if appHealth == argocd.ApplicationSyncRunning {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterInstalling,
			msg,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallingPhase
		clusterTemplateInstance.Status.Message = msg
		return nil
	}

	if appHealth == argocd.ApplicationDegraded {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ApplicationDegraded,
			msg,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallFailedPhase
		clusterTemplateInstance.Status.Message = msg
		return nil
	}

	if appHealth == argocd.ApplicationError {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ApplicationError,
			msg,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallFailedPhase
		clusterTemplateInstance.Status.Message = msg
		return nil
	}

	clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallingPhase
	clusterTemplateInstance.Status.Message = "Cluster is installing"
	if _, ok := clusterTemplateInstance.Annotations[v1alpha1.ClusterProviderExperimentalAnnotation]; ok {
		CTIlog.Info("Experimental provider specified", "name", clusterTemplateInstance.Name)
		return nil
	}

	provider := clusterprovider.GetClusterProvider(*application)

	if provider == nil {
		msg := "Unknown cluster provider - only Hive and Hypershift clusters are recognized"
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterProviderDetectionFailed,
			msg,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallFailedPhase
		clusterTemplateInstance.Status.Message = msg
		return nil
	}

	ready, status, err := provider.GetClusterStatus(ctx, r.Client, *clusterTemplateInstance)
	CTIlog.Info(
		"Instance status - "+status,
		"name",
		clusterTemplateInstance.Namespace+"/"+clusterTemplateInstance.Name,
	)
	if err != nil {
		msg := fmt.Sprintf("Failed to detect cluster status - %q", err)
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterStatusFailed,
			msg,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallFailedPhase
		clusterTemplateInstance.Status.Message = msg
		return err
	}

	if ready {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionTrue,
			v1alpha1.ClusterInstalled,
			status,
		)
	} else {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterInstalling,
			status,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallingPhase
		clusterTemplateInstance.Status.Message = "Cluster is installing"
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCredentials(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	clusterSetupSucceededCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterSetupSucceeded),
	)

	if clusterSetupSucceededCondition.Status == metav1.ConditionFalse {
		return nil
	}

	if clusterTemplateInstance.Status.APIserverURL == "" {
		kubeconfigSecret := corev1.Secret{}

		if err := r.Client.Get(
			ctx,
			client.ObjectKey{
				Name:      clusterTemplateInstance.GetKubeconfigRef(),
				Namespace: clusterTemplateInstance.Namespace,
			},
			&kubeconfigSecret,
		); err != nil {
			return err
		}

		kubeconfig := api.Config{}
		if err := yaml.Unmarshal(kubeconfigSecret.Data["kubeconfig"], &kubeconfig); err != nil {
			return err
		}
		clusterTemplateInstance.Status.APIserverURL = kubeconfig.Clusters[0].Cluster.Server
	}

	clusterTemplateInstance.Status.AdminPassword = &corev1.LocalObjectReference{
		Name: clusterTemplateInstance.GetKubeadminPassRef(),
	}
	clusterTemplateInstance.Status.Kubeconfig = &corev1.LocalObjectReference{
		Name: clusterTemplateInstance.GetKubeconfigRef(),
	}

	// Set the cluster setup secrets
	clusterSetupSecrets := &corev1.SecretList{}
	req, _ := labels.NewRequirement(
		v1alpha1.CTISetupSecretLabel,
		selection.Exists,
		[]string{},
	)
	ctiNameLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINameLabel,
		selection.Equals,
		[]string{clusterTemplateInstance.Name},
	)
	ctiNsLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINamespaceLabel,
		selection.Equals,
		[]string{clusterTemplateInstance.Namespace},
	)
	selector := labels.NewSelector().Add(*req, *ctiNameLabelReq, *ctiNsLabelReq)
	if err := r.Client.List(ctx, clusterSetupSecrets, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return err
	}
	for _, secret := range clusterSetupSecrets.Items {
		if !clusterTemplateInstance.ContainsSetupSecret(secret.Name) {
			clusterTemplateInstance.Status.ClusterSetupSecrets = append(
				clusterTemplateInstance.Status.ClusterSetupSecrets,
				corev1.LocalObjectReference{Name: secret.Name},
			)
		}
	}

	if err := r.ReconcileDynamicRoles(ctx, r.Client, clusterTemplateInstance); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf(
			"failed to reconcile role and role-bindings for users with cluster-templates-role - %q",
			err,
		)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	clusterTemplateInstance.Status.Phase = v1alpha1.ReadyPhase
	clusterTemplateInstance.Status.Message = "Cluster is ready"

	return nil
}

func (*ClusterTemplateInstanceReconciler) ReconcileDynamicRoles(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	roleSubjects, err := clusterTemplateInstance.GetSubjectsWithClusterTemplateUserRole(
		ctx,
		k8sClient,
	)
	if err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf("failed to get list of users - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	role, err := clusterTemplateInstance.CreateDynamicRole(ctx, k8sClient)

	if err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf("failed to create role to access cluster secrets - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	_, err = clusterTemplateInstance.CreateDynamicRoleBinding(ctx, k8sClient, role, roleSubjects)
	if err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf(
			"failed to create RoleBinding for %d subjects - %q",
			len(roleSubjects),
			err,
		)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileCreateManagedCluster(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
	clusterTemplateLabels map[string]string,
) error {
	if !clusterTemplateInstance.PhaseCanExecute(
		v1alpha1.ClusterInstallSucceeded,
		v1alpha1.ManagedClusterCreated,
	) {
		return nil
	}

	if skipClusterRegistration {
		clusterTemplateInstance.SetManagedClusterCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCSkipped,
			"ManagedCluster skipped per ClusterTemplate spec",
		)
		return nil
	}

	if !r.EnableManagedCluster {
		clusterTemplateInstance.SetManagedClusterCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCSkipped,
			"ManagedCluster CRD does not exist, skipping",
		)
		return nil
	}

	CTIlog.Info(
		"Create ManagedCluster for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)

	if err := ocm.CreateManagedCluster(ctx, r.Client, clusterTemplateInstance, clusterTemplateLabels); err != nil {
		clusterTemplateInstance.SetManagedClusterCreatedCondition(
			metav1.ConditionFalse,
			v1alpha1.MCFailed,
			"Failed to create MangedCluster",
		)
		return err
	}
	mc, err := ocm.GetManagedCluster(ctx, r.Client, clusterTemplateInstance)
	if err != nil {
		_, ok := err.(*ocm.MCNotFoundError)
		if !ok {
			return err
		}
	}
	clusterTemplateInstance.Status.ManagedCluster = corev1.LocalObjectReference{
		Name: mc.Name,
	}
	clusterTemplateInstance.SetManagedClusterCreatedCondition(
		metav1.ConditionTrue,
		v1alpha1.MCCreated,
		"ManagedCluster created successfully",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileImportManagedCluster(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
) error {
	if !clusterTemplateInstance.PhaseCanExecute(
		v1alpha1.ManagedClusterCreated,
		v1alpha1.ManagedClusterImported,
	) {
		return nil
	}

	if skipClusterRegistration {
		clusterTemplateInstance.SetManagedClusterImportedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCImportSkipped,
			"ManagedCluster skipped per ClusterTemplate spec",
		)
		return nil
	}

	if !r.EnableManagedCluster {
		clusterTemplateInstance.SetManagedClusterImportedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCImportSkipped,
			"ManagedCluster CRD does not exist, skipping",
		)
		return nil
	}

	CTIlog.Info(
		"Import ManagedCluster of clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)

	imported, err := ocm.ImportManagedCluster(ctx, r.Client, clusterTemplateInstance)
	if err != nil {
		clusterTemplateInstance.SetManagedClusterImportedCondition(
			metav1.ConditionFalse,
			v1alpha1.MCImportFailed,
			"Failed to import ManagedCluster",
		)
		return err
	}
	// TODO mc import err state ?
	if imported {
		clusterTemplateInstance.SetManagedClusterImportedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCImported,
			"ManagedCluster imported successfully",
		)
	} else {
		clusterTemplateInstance.Status.Phase = v1alpha1.ManagedClusterImportingPhase
		clusterTemplateInstance.Status.Message = "ManagedCluster is importing"
		clusterTemplateInstance.SetManagedClusterImportedCondition(
			metav1.ConditionFalse,
			v1alpha1.MCImporting,
			"ManagedCluster is importing",
		)
	}
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileCreateKlusterlet(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
) error {
	if !clusterTemplateInstance.PhaseCanExecute(v1alpha1.ManagedClusterImported) {
		return nil
	}

	if !(r.EnableKlusterlet && !skipClusterRegistration) {
		clusterTemplateInstance.SetKlusterletCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.KlusterletSkipped,
			"KlusterletAddonConfig CRD does not exist, skipping",
		)
		return nil
	}

	CTIlog.Info(
		"Create KlusterletAddonConfig for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)

	if err := ocm.CreateKlusterletAddonConfig(ctx, r.Client, clusterTemplateInstance); err != nil {
		clusterTemplateInstance.SetKlusterletCreatedCondition(
			metav1.ConditionFalse,
			v1alpha1.KlusterletFailed,
			fmt.Sprint("Failed to create KlusterletAddonConfig - "+err.Error()),
		)
		return err
	}
	clusterTemplateInstance.SetKlusterletCreatedCondition(
		metav1.ConditionTrue,
		v1alpha1.KlusterletCreated,
		"KlusterletAddonConfig created successfully",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileConsoleURL(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
) error {
	if !clusterTemplateInstance.PhaseCanExecute(v1alpha1.ManagedClusterImported) {
		return nil
	}
	if skipClusterRegistration || !r.EnableManagedCluster {
		clusterTemplateInstance.SetConsoleURLCondition(
			metav1.ConditionTrue,
			v1alpha1.ConsoleURLSkipped,
			"ManagedCluster will not be created, skipping",
		)
		return nil
	}
	mc, err := ocm.GetManagedCluster(ctx, r.Client, clusterTemplateInstance)
	if err != nil {
		_, ok := err.(*ocm.MCNotFoundError)
		if !ok {
			clusterTemplateInstance.SetConsoleURLCondition(
				metav1.ConditionFalse,
				v1alpha1.ConsoleURLFailed,
				err.Error(),
			)
		}
		return err
	}
	if mc.Status.ClusterClaims != nil {
		for _, cc := range mc.Status.ClusterClaims {
			if cc.Name == "consoleurl.cluster.open-cluster-management.io" {
				clusterTemplateInstance.Status.ConsoleURL = cc.Value
				clusterTemplateInstance.SetConsoleURLCondition(
					metav1.ConditionTrue,
					v1alpha1.ConsoleURLSucceeded,
					"Console URL retrieved",
				)
			}
		}
	}
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileAddClusterToArgo(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
) error {
	if !clusterTemplateInstance.PhaseCanExecute(
		v1alpha1.ManagedClusterImported,
		v1alpha1.ArgoClusterAdded,
	) {
		return nil
	}

	if err := clustersetup.AddClusterToArgo(
		ctx,
		r.Client,
		clusterTemplateInstance,
		clustersetup.GetClientForCluster,
		ArgoCDNamespace,
		r.EnableManagedCluster && !skipClusterRegistration,
	); err != nil {
		clusterTemplateInstance.SetArgoClusterAddedCondition(
			metav1.ConditionFalse,
			v1alpha1.ArgoClusterFailed,
			fmt.Sprintf("Failed to add cluster to argo - %q", err),
		)
		return err
	}
	clusterTemplateInstance.SetArgoClusterAddedCondition(
		metav1.ConditionTrue,
		v1alpha1.ArgoClusterCreated,
		"Cluster added to argo successfully",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterSetupCreate(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterSetup []string,
) error {

	if !clusterTemplateInstance.PhaseCanExecute(
		v1alpha1.ArgoClusterAdded,
		v1alpha1.ClusterSetupCreated,
	) {
		return nil
	}

	if len(clusterSetup) == 0 {
		clusterTemplateInstance.SetClusterSetupCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.ClusterSetupNotSpecified,
			"No cluster setup specified",
		)
		return nil
	}

	CTIlog.Info(
		"Create cluster setup for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)
	if err := clusterTemplateInstance.CreateDay2Applications(
		ctx,
		r.Client,
		ArgoCDNamespace,
		clusterSetup,
	); err != nil {
		clusterTemplateInstance.SetClusterSetupCreatedCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterSetupCreationFailed,
			fmt.Sprintf("Failed to create cluster setup - %q", err),
		)
		return err
	}
	clusterTemplateInstance.SetClusterSetupCreatedCondition(
		metav1.ConditionTrue,
		v1alpha1.SetupCreated,
		"Cluster setup created",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterSetup(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterSetup []string,
) error {

	if !clusterTemplateInstance.PhaseCanExecute(
		v1alpha1.ClusterSetupCreated,
		v1alpha1.ClusterSetupSucceeded,
	) {
		return nil
	}

	if len(clusterSetup) == 0 {
		clusterTemplateInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionTrue,
			v1alpha1.ClusterSetupNotDefined,
			"No cluster setup defined",
		)
		return nil
	}

	CTIlog.Info(
		"reconcile cluster setup for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)
	applications, err := clusterTemplateInstance.GetDay2Applications(ctx, r.Client, ArgoCDNamespace)

	if err != nil {
		clusterTemplateInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterSetupFetchFailed,
			fmt.Sprintf("Failed to list setup apps - %q", err),
		)
		return err
	}

	if len(applications.Items) == 0 {
		clusterTemplateInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterSetupAppsNotFound,
			"Failed to find cluster setup apps",
		)
		return fmt.Errorf("failed to find cluster setup apps")
	}

	clusterTemplateInstance.Status.Phase = v1alpha1.ClusterSetupRunningPhase
	clusterTemplateInstance.Status.Message = "Cluster setup is running"
	clusterSetupStatus := []v1alpha1.ClusterSetupStatus{}
	allSynced := true
	errorSetups := []string{}
	degradedSetups := []string{}
	for _, app := range applications.Items {
		setupName := app.Labels[v1alpha1.CTISetupLabel]
		status, msg := argocd.GetApplicationHealth(&app, true)

		clusterSetupStatus = append(clusterSetupStatus, v1alpha1.ClusterSetupStatus{
			Name:    setupName,
			Status:  status,
			Message: msg,
		})

		if status != argocd.ApplicationHealthy {
			allSynced = false
		}

		if status == argocd.ApplicationError {
			errorSetups = append(errorSetups, setupName)
		}

		if status == argocd.ApplicationDegraded {
			degradedSetups = append(degradedSetups, setupName)
		}
	}

	clusterTemplateInstance.Status.ClusterSetup = &clusterSetupStatus

	if allSynced {
		clusterTemplateInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionTrue,
			v1alpha1.SetupSucceeded,
			"Cluster setup succeeded",
		)
	} else if len(errorSetups) > 0 {
		msg := fmt.Sprintf("Following cluster setups are in error state - %v", errorSetups)
		clusterTemplateInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterSetupError,
			msg,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterSetupErrorPhase
		clusterTemplateInstance.Status.Message = msg
	} else if len(degradedSetups) > 0 {
		msg := fmt.Sprintf("Following cluster setups are in degraded state - %v", degradedSetups)
		clusterTemplateInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterSetupDegraded,
			msg,
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterSetupDegradedPhase
		clusterTemplateInstance.Status.Message = msg
	} else {
		clusterTemplateInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterSetupRunning,
			"Cluster setup is running",
		)
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterSetupRunningPhase
		clusterTemplateInstance.Status.Message = "Cluster setup is running"
	}

	return nil
}

func StartCTIController(
	mgr ctrl.Manager,
	enableHypershift bool,
	enableHive bool,
	enableManagedCluster bool,
	enableKlusterlet bool,
) context.CancelFunc {
	ctiReconciller := &ClusterTemplateInstanceReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		EnableHypershift:     enableHypershift,
		EnableHive:           enableHive,
		EnableManagedCluster: enableManagedCluster,
		EnableKlusterlet:     enableKlusterlet,
	}
	if ctiReconciller.Clock == nil {
		ctiReconciller.Clock = realClock{}
	}
	ctiController, err := controller.NewUnmanaged("cti-controller", mgr, controller.Options{
		Reconciler: ctiReconciller,
	})

	if err != nil {
		CTIlog.Error(err, "unable to create cti-controller")
		os.Exit(1)
	}

	ctiReconciller.SetupWatches(ctiController)

	ctx, cancel := context.WithCancel(context.Background())

	// Start our controller in a goroutine so that we do not block.
	go func() {
		// Block until our controller manager is elected leader. We presume our
		// entire process will terminate if we lose leadership, so we don't need
		// to handle that.
		<-mgr.Elected()

		// Start our controller. This will block until the context is
		// closed, or the controller returns an error.
		if err := ctiController.Start(ctx); err != nil {
			CTIlog.Error(err, "cannot run cti-controller")
		}
	}()

	return cancel
}

func (r *ClusterTemplateInstanceReconciler) SetupWatches(ctrl controller.Controller) {
	ctrl.Watch(
		&source.Kind{Type: &v1alpha1.ClusterTemplateInstance{}},
		&handler.EnqueueRequestForObject{},
	)
	ctrl.Watch(
		&source.Kind{Type: &argo.Application{}},
		handler.EnqueueRequestsFromMapFunc(MapObjToInstance),
	)

	if r.EnableHive {
		ctrl.Watch(
			&source.Kind{Type: &hivev1.ClusterClaim{}},
			handler.EnqueueRequestsFromMapFunc(
				r.MapArgoResourceToInstance(v1alpha1.ClusterClaimGVK),
			),
		)

		ctrl.Watch(
			&source.Kind{Type: &hivev1.ClusterDeployment{}},
			handler.EnqueueRequestsFromMapFunc(
				r.MapArgoResourceToInstance(v1alpha1.ClusterDeploymentGVK),
			),
		)
	}

	if r.EnableHypershift {
		ctrl.Watch(
			&source.Kind{Type: &hypershiftv1beta1.HostedCluster{}},
			handler.EnqueueRequestsFromMapFunc(
				r.MapArgoResourceToInstance(v1alpha1.HostedClusterGVK),
			),
		)
		ctrl.Watch(
			&source.Kind{Type: &hypershiftv1beta1.NodePool{}},
			handler.EnqueueRequestsFromMapFunc(r.MapArgoResourceToInstance(v1alpha1.NodePoolGVK)))
	}

	if r.EnableManagedCluster {
		ctrl.Watch(
			&source.Kind{Type: &ocmv1.ManagedCluster{}},
			handler.EnqueueRequestsFromMapFunc(MapObjToInstance),
		)
	}
}

func (r *ClusterTemplateInstanceReconciler) MapArgoResourceToInstance(
	resourceGVK schema.GroupVersionResource,
) func(res client.Object) []reconcile.Request {
	return func(res client.Object) []reconcile.Request {
		reply := []reconcile.Request{}
		apps := &argo.ApplicationList{}

		ctiNameLabelReq, _ := labels.NewRequirement(
			v1alpha1.CTINameLabel,
			selection.Exists,
			[]string{},
		)
		ctiNsLabelReq, _ := labels.NewRequirement(
			v1alpha1.CTINamespaceLabel,
			selection.Exists,
			[]string{},
		)
		ctiSetupReq, _ := labels.NewRequirement(
			v1alpha1.CTISetupLabel,
			selection.DoesNotExist,
			[]string{},
		)
		selector := labels.NewSelector().Add(*ctiNameLabelReq, *ctiNsLabelReq, *ctiSetupReq)

		if err := r.Client.List(context.TODO(), apps, &client.ListOptions{
			LabelSelector: selector,
		}); err != nil {
			return reply
		}

		for _, app := range apps.Items {
			name := ""
			namespace := ""
			for key, val := range app.GetLabels() {
				if key == v1alpha1.CTINameLabel {
					name = val
				}
				if key == v1alpha1.CTINamespaceLabel {
					namespace = val
				}
			}
			if name != "" && namespace != "" {
				for _, argoRes := range app.Status.Resources {
					if resourceGVK.Resource == argoRes.Kind &&
						resourceGVK.Group == argoRes.Group &&
						resourceGVK.Version == argoRes.Version &&
						res.GetNamespace() == argoRes.Namespace &&
						res.GetName() == argoRes.Name {
						reply = append(
							reply,
							reconcile.Request{NamespacedName: types.NamespacedName{
								Namespace: namespace,
								Name:      name,
							}},
						)
					}
				}
			}
		}
		return reply
	}
}

func MapObjToInstance(obj client.Object) []reconcile.Request {
	reply := []reconcile.Request{}
	name := ""
	namespace := ""
	for key, val := range obj.GetLabels() {
		if key == v1alpha1.CTINameLabel {
			name = val
		}
		if key == v1alpha1.CTINamespaceLabel {
			namespace = val
		}
	}
	if name != "" && namespace != "" {
		reply = append(reply, reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}})
	}
	return reply
}
