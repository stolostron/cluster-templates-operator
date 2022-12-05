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

	"github.com/kubernetes-client/go-base/config/api"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/argocd"

	"github.com/stolostron/cluster-templates-operator/clusterprovider"
	"github.com/stolostron/cluster-templates-operator/clustersetup"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ClusterTemplateInstanceReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	EnableHypershift bool
	EnableHive       bool
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=hypershift.openshift.io,resources=hostedclusters;nodepools,verbs=get;list;watch
// +kubebuilder:rbac:groups=hive.openshift.io,resources=clusterclaims;clusterdeployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;roles,verbs=get;list;watch;create;update;delete

func (r *ClusterTemplateInstanceReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	clusterTemplateInstance := &v1alpha1.ClusterTemplateInstance{}
	if err := r.Get(ctx, req.NamespacedName, clusterTemplateInstance); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(
				"clustertemplateinstance not found, aborting reconcile",
				"name",
				req.NamespacedName,
			)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if clusterTemplateInstance.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(
			clusterTemplateInstance,
			v1alpha1.CTIFinalizer,
		) {
			if clusterTemplateInstance.Status.ClusterTemplateSpec != nil {
				app, err := clusterTemplateInstance.GetDay1Application(ctx, r.Client)
				if err != nil {
					if !apierrors.IsNotFound(err) {
						return ctrl.Result{}, err
					}
				}

				if app != nil {
					if err = r.Client.Delete(ctx, app); err != nil {
						return ctrl.Result{}, err
					}
				}

				apps, err := clusterTemplateInstance.GetDay2Applications(ctx, r.Client)
				if err != nil {
					if !apierrors.IsNotFound(err) {
						return ctrl.Result{}, err
					}
				}

				if apps != nil {
					for _, app := range apps.Items {
						if err = r.Client.Delete(ctx, &app); err != nil {
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
					Namespace:     clusterTemplateInstance.Status.ClusterTemplateSpec.ArgoCDNamespace,
				}); err != nil {
					return ctrl.Result{}, err
				}

				for _, secret := range secrets.Items {
					if err := r.Client.Delete(ctx, &secret); err != nil {
						return ctrl.Result{}, err
					}
				}
			}
			controllerutil.RemoveFinalizer(
				clusterTemplateInstance,
				v1alpha1.CTIFinalizer,
			)
			if err := r.Update(ctx, clusterTemplateInstance); err != nil {
				return ctrl.Result{}, err
			}
		}

	}

	if !controllerutil.ContainsFinalizer(
		clusterTemplateInstance,
		v1alpha1.CTIFinalizer,
	) {
		controllerutil.AddFinalizer(clusterTemplateInstance, v1alpha1.CTIFinalizer)
		if err := r.Update(ctx, clusterTemplateInstance); err != nil {
			return ctrl.Result{}, err
		}
	}

	if len(clusterTemplateInstance.Status.Conditions) == 0 {
		SetDefaultConditions(clusterTemplateInstance)
		clusterTemplateInstance.Status.Phase = v1alpha1.PendingPhase
		clusterTemplateInstance.Status.Message = v1alpha1.PendingMessage
	}

	if clusterTemplateInstance.Status.ClusterTemplateSpec == nil {
		clusterTemplate := v1alpha1.ClusterTemplate{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: clusterTemplateInstance.Spec.ClusterTemplateRef}, &clusterTemplate); err != nil {
			clusterTemplateInstance.Status.Phase = v1alpha1.FailedPhase
			errMsg := fmt.Sprintf("failed to fetch ClusterTemplate - %q", err)
			clusterTemplateInstance.Status.Message = errMsg
			if updErr := r.Status().Update(ctx, clusterTemplateInstance); updErr != nil {
				return ctrl.Result{}, fmt.Errorf(
					"failed to update status of clustertemplateinstance %q: %w",
					req.NamespacedName,
					updErr,
				)
			}
			return ctrl.Result{}, err
		}
		clusterTemplateInstance.Status.ClusterTemplateSpec = &clusterTemplate.Spec
		if val, ok := clusterTemplate.Annotations[clusterprovider.ClusterProviderExperimentalAnnotation]; ok {
			if clusterTemplateInstance.Annotations == nil {
				clusterTemplateInstance.Annotations = map[string]string{}
			}
			clusterTemplateInstance.Annotations[clusterprovider.ClusterProviderExperimentalAnnotation] = val
		}
	}

	err := r.reconcile(ctx, clusterTemplateInstance)

	if updErr := r.Status().Update(ctx, clusterTemplateInstance); updErr != nil {
		return ctrl.Result{}, fmt.Errorf(
			"failed to update status of clustertemplateinstance %q: %w",
			req.NamespacedName,
			updErr,
		)
	}

	return ctrl.Result{}, err
}

func (r *ClusterTemplateInstanceReconciler) reconcile(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	if err := r.reconcileClusterCreate(ctx, clusterTemplateInstance); err != nil {
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

	if err := r.reconcileAddClusterToArgo(ctx, clusterTemplateInstance); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.ArgoClusterFailedPhase
		errMsg := fmt.Sprintf("failed to add cluster to argo - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileClusterSetupCreate(ctx, clusterTemplateInstance); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterSetupCreateFailedPhase
		errMsg := fmt.Sprintf("failed to create cluster setup - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileClusterSetup(ctx, clusterTemplateInstance); err != nil {
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
) error {

	clusterDefinitionCreatedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterDefinitionCreated),
	)

	if clusterDefinitionCreatedCondition.Status == metav1.ConditionFalse {
		if err := clusterTemplateInstance.CreateDay1Application(ctx, r.Client); err != nil {
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
	log := ctrl.LoggerFrom(ctx)
	log.Info(
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

	log.Info(
		"Fetch day1 argo application",
		"name",
		clusterTemplateInstance.Namespace+"/"+clusterTemplateInstance.Name,
	)
	application, err := clusterTemplateInstance.GetDay1Application(ctx, r.Client)

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

	appHealth, msg := argocd.GetApplicationHealth(application)
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
	if _, ok := clusterTemplateInstance.Annotations[clusterprovider.ClusterProviderExperimentalAnnotation]; ok {
		log.Info("Experimental provider specified", "name", clusterTemplateInstance.Name)
		return nil
	}

	provider := clusterprovider.GetClusterProvider(*application, log)

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
	log.Info(
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

func (r *ClusterTemplateInstanceReconciler) reconcileAddClusterToArgo(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	installSucceededCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterInstallSucceeded),
	)

	if installSucceededCondition.Status == metav1.ConditionFalse {
		return nil
	}

	argoClusterAddedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ArgoClusterAdded),
	)

	if argoClusterAddedCondition.Status == metav1.ConditionTrue {
		return nil
	}

	if err := clustersetup.AddClusterToArgo(
		ctx,
		r.Client,
		clusterTemplateInstance,
		clustersetup.GetClientForCluster,
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
) error {
	log := ctrl.LoggerFrom(ctx)

	argoClusterAddedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ArgoClusterAdded),
	)

	if argoClusterAddedCondition.Status == metav1.ConditionFalse {
		return nil
	}

	if len(clusterTemplateInstance.Status.ClusterTemplateSpec.ClusterSetup) == 0 {
		clusterTemplateInstance.SetClusterSetupCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.ClusterSetupNotSpecified,
			"No cluster setup specified",
		)
		return nil
	}

	clusterSetupCreatedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterSetupCreated),
	)

	if clusterSetupCreatedCondition.Status == metav1.ConditionTrue {
		return nil
	}

	log.Info(
		"Create cluster setup for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)
	if err := clusterTemplateInstance.CreateDay2Applications(
		ctx,
		r.Client,
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
) error {
	log := ctrl.LoggerFrom(ctx)

	clusterSetupCreatedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterSetupCreated),
	)

	if clusterSetupCreatedCondition.Status == metav1.ConditionFalse {
		return nil
	}

	clusterSetupSucceededCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterSetupSucceeded),
	)

	if clusterSetupSucceededCondition.Status == metav1.ConditionTrue {
		return nil
	}

	if len(clusterTemplateInstance.Status.ClusterTemplateSpec.ClusterSetup) == 0 {
		clusterTemplateInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionTrue,
			v1alpha1.ClusterSetupNotDefined,
			"No cluster setup defined",
		)
		return nil
	}

	log.Info(
		"reconcile cluster setup for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)
	applications, err := clusterTemplateInstance.GetDay2Applications(ctx, r.Client)

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
		status, msg := argocd.GetApplicationHealth(&app)

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

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	mapApplicationToInstance := func(app client.Object) []reconcile.Request {
		reply := []reconcile.Request{}
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
			reply = append(reply, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}})
		}
		return reply
	}

	mapResourceToInstance := func(res client.Object) []reconcile.Request {
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
					if res.GetObjectKind().GroupVersionKind().Kind == argoRes.Kind &&
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

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterTemplateInstance{}).
		Watches(
			&source.Kind{Type: &argo.Application{}},
			handler.EnqueueRequestsFromMapFunc(mapApplicationToInstance))

	if r.EnableHive {
		builder = builder.Watches(
			&source.Kind{Type: &hivev1.ClusterClaim{}},
			handler.EnqueueRequestsFromMapFunc(mapResourceToInstance)).
			Watches(
				&source.Kind{Type: &hivev1.ClusterDeployment{}},
				handler.EnqueueRequestsFromMapFunc(mapResourceToInstance))
	}

	if r.EnableHypershift {
		builder = builder.Watches(
			&source.Kind{Type: &hypershiftv1alpha1.HostedCluster{}},
			handler.EnqueueRequestsFromMapFunc(mapResourceToInstance)).
			Watches(
				&source.Kind{Type: &hypershiftv1alpha1.NodePool{}},
				handler.EnqueueRequestsFromMapFunc(mapResourceToInstance))
	}
	return builder.Complete(r)
}

func SetDefaultConditions(clusterInstance *v1alpha1.ClusterTemplateInstance) {
	clusterInstance.SetClusterDefinitionCreatedCondition(
		metav1.ConditionFalse,
		v1alpha1.ClusterDefinitionPending,
		"Pending",
	)
	clusterInstance.SetClusterInstallCondition(
		metav1.ConditionFalse,
		v1alpha1.ClusterDefinitionNotCreated,
		"Waiting for cluster definition to be created",
	)
	clusterInstance.SetArgoClusterAddedCondition(
		metav1.ConditionFalse,
		v1alpha1.ArgoClusterPending,
		"Waiting for cluster to be ready",
	)
	clusterInstance.SetClusterSetupCreatedCondition(
		metav1.ConditionFalse,
		v1alpha1.ClusterNotInstalled,
		"Waiting for cluster to be ready",
	)
	clusterInstance.SetClusterSetupSucceededCondition(
		metav1.ConditionFalse,
		v1alpha1.ClusterSetupNotCreated,
		"Waiting for cluster setup to be created",
	)
}
