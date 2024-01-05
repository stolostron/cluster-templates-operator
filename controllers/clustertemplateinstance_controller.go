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
	"strings"
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
	"github.com/stolostron/cluster-templates-operator/utils"

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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	ocmv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argoAppSet "github.com/argoproj/applicationset/pkg/utils"

	consoleV1 "github.com/openshift/api/route/v1"
	restclient "k8s.io/client-go/rest"
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
	cti := &v1alpha1.ClusterTemplateInstance{}
	if err := r.Get(ctx, req.NamespacedName, cti); err != nil {
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

	if len(cti.Status.Conditions) == 0 {
		cti.Status.Phase = v1alpha1.PendingPhase
		cti.Status.Message = v1alpha1.PendingMessage
	}

	if cti.GetDeletionTimestamp() != nil {
		return r.delete(ctx, cti)
	}

	// Check if CTI should be auto-removed:
	requeueAfter, err := r.autoDelete(ctx, cti)
	if err != nil {
		return ctrl.Result{}, err
	}

	var ct client.Object
	if cti.Spec.KubeconfigSecretRef != nil {
		ct = &v1alpha1.ClusterTemplateSetup{}
	} else {
		ct = &v1alpha1.ClusterTemplate{}
	}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: cti.Spec.ClusterTemplateRef}, ct); err != nil {
		cti.Status.Phase = v1alpha1.FailedPhase
		cti.Status.Message = fmt.Sprintf("failed to fetch ClusterTemplate - %q", err)
		if updErr := r.Status().Update(ctx, cti); updErr != nil {
			return ctrl.Result{}, fmt.Errorf(
				"failed to update status of clustertemplateinstance %q: %w",
				req.NamespacedName,
				updErr,
			)
		}

		return ctrl.Result{}, err
	}

	isNsType := false
	if ct, ok := ct.(*v1alpha1.ClusterTemplate); ok {
		isNsType = ct.Spec.Type == v1alpha1.Namespace
	}

	cti.SetDefaultConditions(isNsType)

	if isNsType {
		err = r.reconcileNamespace(ctx, cti, ct.(*v1alpha1.ClusterTemplate))
	} else {
		err = r.reconcileCluster(ctx, cti, ct)
	}

	if updErr := r.Status().Update(ctx, cti); updErr != nil {
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
	cti *v1alpha1.ClusterTemplateInstance,
) (ctrl.Result, error) {
	if len(cti.Finalizers) != 1 || !controllerutil.ContainsFinalizer(
		cti,
		v1alpha1.CTIFinalizer,
	) {
		return ctrl.Result{}, nil
	}
	var clusterTemplate client.Object
	if cti.Spec.KubeconfigSecretRef != nil {
		clusterTemplate = &v1alpha1.ClusterTemplateSetup{}
	} else {
		clusterTemplate = &v1alpha1.ClusterTemplate{}
	}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: cti.Spec.ClusterTemplateRef}, clusterTemplate); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	} else {
		if ct, ok := clusterTemplate.(*v1alpha1.ClusterTemplate); ok {
			err := cti.DeleteDay1Application(ctx, r.Client, ArgoCDNamespace, ct.Spec.ClusterDefinition)
			if err != nil {
				return ctrl.Result{}, err
			}
			err = cti.DeleteDay2Application(ctx, r.Client, ArgoCDNamespace, ct.Spec.ClusterSetup)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			err = cti.DeleteDay2Application(ctx, r.Client, ArgoCDNamespace, clusterTemplate.(*v1alpha1.ClusterTemplateSetup).Spec.ClusterSetup)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// cleanup argocd secrets (ie new cluster)
	ctiNameLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINameLabel,
		selection.Equals,
		[]string{cti.Name},
	)
	ctiNsLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINamespaceLabel,
		selection.Equals,
		[]string{cti.Namespace},
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
		mc, err := ocm.GetManagedCluster(ctx, r.Client, cti)
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
		cti,
		v1alpha1.CTIFinalizer,
	)
	err := r.Update(ctx, cti)
	return ctrl.Result{}, err
}

func getClusterProperties(ct client.Object) (bool, string, []string) {
	var skipClusterRegistration bool
	var clusterDefinition string
	var clusterSetup []string
	switch clusterTemplate := ct.(type) {
	case *v1alpha1.ClusterTemplateSetup:
		skipClusterRegistration = clusterTemplate.Spec.SkipClusterRegistration
		clusterSetup = clusterTemplate.Spec.ClusterSetup
	case *v1alpha1.ClusterTemplate:
		skipClusterRegistration = clusterTemplate.Spec.SkipClusterRegistration
		clusterDefinition = clusterTemplate.Spec.ClusterDefinition
		clusterSetup = clusterTemplate.Spec.ClusterSetup
	}

	return skipClusterRegistration, clusterDefinition, clusterSetup
}

func (r *ClusterTemplateInstanceReconciler) reconcileNamespace(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	ct *v1alpha1.ClusterTemplate,
) error {
	if err := r.reconcileEnvironmentCreate(ctx, cti, ct.Spec.ClusterDefinition, ct.Spec.TargetCluster, cti.Namespace+"-"+cti.Name); err != nil {
		return cti.SetErrorPhase(v1alpha1.EnvironmentDefinitionFailedPhase, "failed to create namespace definition", err)
	}

	if err := r.reconcileEnvironmentStatus(ctx, cti, true); err != nil {
		return cti.SetErrorPhase(v1alpha1.EnvironmentInstallFailedPhase, "failed to get namespace status", err)
	}

	if err := r.reconcileUserAccount(ctx, cti); err != nil {
		return cti.SetErrorPhase(v1alpha1.EnvironmentAccountFailedPhase, "failed to create user account", err)
	}

	if err := r.reconcileNamespaceRBAC(ctx, cti); err != nil {
		return cti.SetErrorPhase(v1alpha1.EnvironmentRBACFailedPhase, "failed to create namespace rbac", err)
	}

	if err := r.reconcileEnvironmentSetupCreate(ctx, cti, ct.Spec.ClusterSetup, true); err != nil {
		return cti.SetErrorPhase(v1alpha1.EnvironmentSetupCreateFailedPhase, "failed to create namespace setup", err)
	}

	if err := r.reconcileEnvironmentSetup(ctx, cti, ct.Spec.ClusterSetup); err != nil {
		return cti.SetErrorPhase(v1alpha1.EnvironmentSetupFailedPhase, "failed to reconcile namespace setup", err)
	}

	if err := r.reconcileAppLinks(ctx, cti); err != nil {
		return cti.SetErrorPhase(v1alpha1.CredentialsFailedPhase, "failed to reconcile app links", err)
	}

	if err := r.reconcileNamespaceCredentials(ctx, cti); err != nil {
		return cti.SetErrorPhase(v1alpha1.CredentialsFailedPhase, "failed to reconcile namespace credentials", err)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileAppLinks(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
) error {
	if !cti.PhaseCanExecute(
		v1alpha1.EnvironmentSetupSucceeded,
		v1alpha1.AppLinksCollected,
	) {
		return nil
	}

	k8sClient, err := r.getClientForCluster(ctx, cti)
	if err != nil {
		cti.SetAppLinksCollectedCondition(
			metav1.ConditionFalse,
			v1alpha1.AppLinksFailed,
			"Failed to create target cluster client - "+err.Error(),
		)
		return err
	}
	apps, err := cti.GetDay2Applications(ctx, k8sClient, ArgoCDNamespace)
	if err != nil {
		cti.SetAppLinksCollectedCondition(
			metav1.ConditionFalse,
			v1alpha1.AppLinksFailed,
			"Failed to get day2 apps - "+err.Error(),
		)
		return err
	}
	links := []string{}
	for _, app := range apps.Items {
		for _, item := range app.Status.Resources {
			if item.Kind == "Route" {
				route := consoleV1.Route{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: item.Name, Namespace: item.Namespace}, &route); err != nil {
					cti.SetAppLinksCollectedCondition(
						metav1.ConditionFalse,
						v1alpha1.AppLinksFailed,
						"Failed to get route - "+err.Error(),
					)
					return err
				}
				links = append(links, route.Spec.Host)
			}
		}
	}
	cti.Status.AppLinks = links

	cti.SetAppLinksCollectedCondition(
		metav1.ConditionTrue,
		v1alpha1.AppLinksSucceeded,
		"App links collected",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileUserAccount(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
) error {
	if !cti.PhaseCanExecute(
		v1alpha1.EnvironmentInstallSucceeded,
		v1alpha1.NamespaceAccountCreated,
	) {
		return nil
	}

	app, err := cti.GetDay1Application(ctx, r.Client, ArgoCDNamespace)
	if err != nil {
		cti.SetEnvironmentAccountCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentAccountFailed,
			"Failed to fetch day1 app - "+err.Error(),
		)
		return err
	}

	namespace := ""
	for _, resource := range app.Status.Resources {
		if resource.Kind == "Namespace" {
			namespace = resource.Name
		}
	}

	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-sa",
			Namespace: namespace,
		},
	}

	client, err := r.getClientForCluster(ctx, cti)
	if err != nil {
		cti.SetEnvironmentAccountCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentAccountFailed,
			"Failed to create k8s client for target cluster - "+err.Error(),
		)
		return nil
	}

	if err := utils.EnsureResourceExists(ctx, client, &sa, false); err != nil {
		cti.SetEnvironmentAccountCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentAccountFailed,
			"Failed to create namespace account - "+err.Error(),
		)
		return err
	}

	cti.SetEnvironmentAccountCondition(
		metav1.ConditionTrue,
		v1alpha1.EnvironmentAccountCreated,
		"Account created",
	)

	return nil
}

func (r *ClusterTemplateInstanceReconciler) getClientForCluster(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
) (client.Client, error) {
	app, err := cti.GetDay1Application(ctx, r.Client, ArgoCDNamespace)
	if err != nil {
		return nil, err
	}

	if app.Spec.Destination.Server == "https://kubernetes.default.svc" {
		return r.Client, nil
	}

	secrets := &corev1.SecretList{}

	ctiNameLabelReq, _ := labels.NewRequirement(
		argoAppSet.ArgoCDSecretTypeLabel,
		selection.Equals,
		[]string{argoAppSet.ArgoCDSecretTypeCluster},
	)

	selector := labels.NewSelector().Add(*ctiNameLabelReq)

	if err := r.Client.List(ctx, secrets, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     ArgoCDNamespace,
	}); err != nil {
		return nil, err
	}

	for _, secret := range secrets.Items {
		server, err := utils.GetValueFromSecret(secret, "server")
		if err != nil {
			return nil, err
		}
		if server == app.Spec.Destination.Server {
			tlsClientConfig := restclient.TLSClientConfig{}
			config, err := utils.GetMapValueFromSecret(secret, "config")
			if err != nil {
				return nil, err
			}

			tlsClientConfig.CAData = []byte(config.TLSClientConfig.CAData)

			restConfig := restclient.Config{
				Host:            app.Spec.Destination.Server,
				TLSClientConfig: tlsClientConfig,
				BearerToken:     config.BearerToken,
			}
			return client.New(&restConfig, client.Options{})
		}
	}
	return nil, nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileNamespaceRBAC(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
) error {
	if !cti.PhaseCanExecute(
		v1alpha1.NamespaceAccountCreated,
		v1alpha1.EnvironmentRBACSucceeded,
	) {
		return nil
	}

	app, err := cti.GetDay1Application(ctx, r.Client, ArgoCDNamespace)
	if err != nil {
		cti.SetEnvironmentRBACCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentRBACFailed,
			"Failed to fetch day1 app - "+err.Error(),
		)
		return err
	}

	namespace := ""
	for _, resource := range app.Status.Resources {
		if resource.Kind == "Namespace" {
			namespace = resource.Name
		}
	}

	user := cti.Annotations[v1alpha1.CTIRequesterAnnotation]

	// TODO webhooks do not work in devmode
	if len(user) == 0 {
		user = "cluster-admin"
	}

	var subj rbacv1.Subject

	if app.Spec.Destination.Server == "https://kubernetes.default.svc" {

		if strings.HasPrefix(user, "system:serviceaccount") {
			parts := strings.Split(user, ":")
			subj = rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      parts[3],
				Namespace: parts[2],
			}
		} else {
			subj = rbacv1.Subject{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     user,
			}
		}
	} else {
		subj = rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      "user-sa",
			Namespace: namespace,
		}
	}

	k8sClient, err := r.getClientForCluster(ctx, cti)
	if err != nil {
		cti.SetEnvironmentRBACCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentRBACFailed,
			"Failed to create client for target cluster - "+err.Error(),
		)
		return err
	}

	for _, resource := range app.Status.Resources {
		if resource.Kind == "Role" && resource.Group == rbacv1.SchemeGroupVersion.Group {
			binding := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "binding-" + resource.Name,
					Namespace: namespace,
				},
				Subjects: []rbacv1.Subject{subj},
				RoleRef: rbacv1.RoleRef{
					APIGroup: resource.Group,
					Kind:     resource.Kind,
					Name:     resource.Name,
				},
			}

			if err := utils.EnsureResourceExists(ctx, k8sClient, binding, false); err != nil {
				cti.SetEnvironmentRBACCondition(
					metav1.ConditionFalse,
					v1alpha1.EnvironmentRBACFailed,
					"Failed to create role biding - "+err.Error(),
				)
				return err
			}
		}
	}

	cti.SetEnvironmentRBACCondition(
		metav1.ConditionTrue,
		v1alpha1.EnvironmentRBACCreated,
		"RBAC created",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileCluster(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	ct client.Object,
) error {
	skipClusterRegistration, clusterDefinition, clusterSetup := getClusterProperties(ct)
	if cti.Spec.KubeconfigSecretRef == nil {
		if err := r.reconcileEnvironmentCreate(ctx, cti, clusterDefinition, "https://kubernetes.default.svc", ""); err != nil {
			return cti.SetErrorPhase(v1alpha1.EnvironmentDefinitionFailedPhase, "failed to create cluster definition", err)
		}
		if err := r.reconcileEnvironmentStatus(ctx, cti, false); err != nil {
			errMsg := fmt.Sprintf("failed to reconcile cluster status - %q", err)
			_, ok := err.(*AppNotFoundError)
			if !ok {
				cti.Status.Phase = v1alpha1.EnvironmentInstallFailedPhase
				cti.Status.Message = errMsg
			}
			return fmt.Errorf(errMsg)
		}
	} else {
		secret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: *cti.Spec.KubeconfigSecretRef, Namespace: cti.Namespace}, secret); err != nil {
			return err
		}
		kubeconfig, okKubeconfig := secret.Data["kubeconfig"]
		if !okKubeconfig {
			return fmt.Errorf("kubeconfig not found in secret %q", *cti.Spec.KubeconfigSecretRef)
		}
		password := secret.Data["password"]
		if err := clusterprovider.CreateClusterSecrets(
			ctx,
			r.Client,
			kubeconfig,
			[]byte("kubeadmin"),
			password,
			*cti,
		); err != nil {
			return err
		}
		cti.SetEnvironmentInstallCondition(
			metav1.ConditionTrue,
			v1alpha1.EnvironmentInstalled,
			"Cluster defined via secret",
		)
	}

	//ACM integration

	if err := r.reconcileCreateManagedCluster(ctx, cti, skipClusterRegistration, ct.GetLabels()); err != nil {
		return cti.SetErrorPhase(v1alpha1.ManagedClusterFailedPhase, "failed to create ManagedCluster", err)
	}

	if err := r.reconcileImportManagedCluster(ctx, cti, skipClusterRegistration); err != nil {
		return cti.SetErrorPhase(v1alpha1.ManagedClusterImportFailedPhase, "failed to import ManagedCluster", err)
	}

	if err := r.reconcileCreateKlusterlet(ctx, cti, skipClusterRegistration); err != nil {
		return cti.SetErrorPhase(v1alpha1.KlusterletCreateFailedPhase, "failed to create Klusterlet", err)
	}

	if err := r.reconcileConsoleURL(ctx, cti, skipClusterRegistration); err != nil {
		return fmt.Errorf("failed to retrieve Console URL - %q", err)
	}

	//

	if err := r.reconcileAddClusterToArgo(ctx, cti, skipClusterRegistration); err != nil {
		errMsg := fmt.Sprintf("failed to add cluster to argo - %q", err)
		_, ok := err.(*clustersetup.LoginError)
		if ok {
			cti.Status.Phase = v1alpha1.ClusterLoginPendingPhase
			cti.Status.Message = "Logging into the new cluster"
		} else {
			cti.Status.Phase = v1alpha1.ArgoClusterFailedPhase
			cti.Status.Message = errMsg
		}
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileEnvironmentSetupCreate(ctx, cti, clusterSetup, false); err != nil {
		return cti.SetErrorPhase(v1alpha1.EnvironmentSetupCreateFailedPhase, "failed to create cluster setup", err)
	}

	if err := r.reconcileEnvironmentSetup(ctx, cti, clusterSetup); err != nil {
		return cti.SetErrorPhase(v1alpha1.EnvironmentSetupFailedPhase, "failed to reconcile cluster setup", err)
	}

	if err := r.reconcileClusterCredentials(ctx, cti); err != nil {
		return cti.SetErrorPhase(v1alpha1.CredentialsFailedPhase, "failed to reconcile cluster credentials", err)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileEnvironmentCreate(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	clusterDefinition string,
	targetCluster string,
	targetNamespace string,
) error {
	environmentDefinitionCreatedCondition := meta.FindStatusCondition(
		cti.Status.Conditions,
		string(v1alpha1.EnvironmentDefinitionCreated),
	)

	if environmentDefinitionCreatedCondition.Status == metav1.ConditionFalse {
		if err := cti.CreateDay1Application(ctx, r.Client, ArgoCDNamespace, ArgoCDNamespace == defaultArgoCDNs, clusterDefinition, targetCluster, targetNamespace); err != nil {
			cti.SetEnvironmentDefinitionCreatedCondition(
				metav1.ConditionFalse,
				v1alpha1.EnvironmentDefinitionFailed,
				fmt.Sprintf("Failed to create application - %q", err),
			)
			return err
		}
		cti.SetEnvironmentDefinitionCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.ApplicationCreated,
			"Application created",
		)
	}
	return nil
}

type AppNotFoundError struct {
	Msg string
}

func (m *AppNotFoundError) Error() string {
	return m.Msg
}

func (r *ClusterTemplateInstanceReconciler) reconcileEnvironmentStatus(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	isNs bool,
) error {
	if !cti.PhaseCanExecute(
		v1alpha1.EnvironmentDefinitionCreated,
		v1alpha1.EnvironmentInstallSucceeded,
	) {
		return nil
	}

	CTIlog.Info(
		"Reconcile environment definition status",
		"name",
		cti.Namespace+"/"+cti.Name,
	)

	application, err := cti.GetDay1Application(ctx, r.Client, ArgoCDNamespace)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return &AppNotFoundError{Msg: err.Error()}
		}
		failedMsg := fmt.Sprintf("Failed to fetch application - %q", err)
		cti.SetEnvironmentInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ApplicationFetchFailed,
			failedMsg,
		)
		return err
	}

	appHealth, msg := argocd.GetApplicationHealth(application, true)
	if appHealth == argocd.ApplicationSyncRunning {
		cti.SetEnvironmentInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentInstalling,
			msg,
		)
		cti.Status.Phase = v1alpha1.EnvironmentInstallingPhase
		cti.Status.Message = msg
		return nil
	}

	if appHealth == argocd.ApplicationDegraded {
		cti.SetEnvironmentInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ApplicationDegraded,
			msg,
		)
		cti.Status.Phase = v1alpha1.EnvironmentInstallFailedPhase
		cti.Status.Message = msg
		return nil
	}

	if appHealth == argocd.ApplicationError {
		cti.SetEnvironmentInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ApplicationError,
			msg,
		)
		cti.Status.Phase = v1alpha1.EnvironmentInstallFailedPhase
		cti.Status.Message = msg
		return nil
	}

	if !isNs {
		cti.Status.Phase = v1alpha1.EnvironmentInstallingPhase
		cti.Status.Message = "Environment is installing"
		if _, ok := cti.Annotations[v1alpha1.ClusterProviderExperimentalAnnotation]; ok {
			CTIlog.Info("Experimental provider specified", "name", cti.Name)
			return nil
		}

		provider := clusterprovider.GetClusterProvider(*application)

		if provider == nil {
			msg := "Unknown cluster provider - only Hive and Hypershift clusters are recognized"
			cti.SetEnvironmentInstallCondition(
				metav1.ConditionFalse,
				v1alpha1.ClusterProviderDetectionFailed,
				msg,
			)
			cti.Status.Phase = v1alpha1.EnvironmentInstallFailedPhase
			cti.Status.Message = msg
			return nil
		}

		ready, status, err := provider.GetClusterStatus(ctx, r.Client, *cti)
		CTIlog.Info(
			"Instance status - "+status,
			"name",
			cti.Namespace+"/"+cti.Name,
		)
		if err != nil {
			msg := fmt.Sprintf("Failed to detect cluster status - %q", err)
			cti.SetEnvironmentInstallCondition(
				metav1.ConditionFalse,
				v1alpha1.ClusterStatusFailed,
				msg,
			)
			cti.Status.Phase = v1alpha1.EnvironmentInstallFailedPhase
			cti.Status.Message = msg
			return err
		}

		if !ready {
			cti.SetEnvironmentInstallCondition(
				metav1.ConditionFalse,
				v1alpha1.EnvironmentInstalling,
				status,
			)
			cti.Status.Phase = v1alpha1.EnvironmentInstallingPhase
			cti.Status.Message = "Environment is installing"
			return nil
		}
	}

	if appHealth == argocd.ApplicationHealthy {
		cti.SetEnvironmentInstallCondition(
			metav1.ConditionTrue,
			v1alpha1.EnvironmentInstalled,
			"Environment is installed",
		)
		cti.Status.Phase = v1alpha1.EnvironmentInstalledPhase
		cti.Status.Message = "Environment is installed"
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileNamespaceCredentials(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
) error {
	if !cti.PhaseCanExecute(
		v1alpha1.AppLinksCollected,
		v1alpha1.NamespaceCredentialsSucceeded,
	) {
		return nil
	}

	app, err := cti.GetDay1Application(ctx, r.Client, ArgoCDNamespace)

	if err != nil {
		cti.SetNamespaceCredentialsCondition(
			metav1.ConditionFalse,
			v1alpha1.NamespaceCredentialsFailed,
			"Failed to list day1 app - "+err.Error(),
		)
		return err
	}

	credentialsFound := true

	// TODO make this work for all target clusters/user types
	if app.Spec.Destination.Server == "https://kubernetes.default.svc" {
		credentialsFound = false
		k8sClient, err := r.getClientForCluster(ctx, cti)
		if err != nil {
			cti.SetNamespaceCredentialsCondition(
				metav1.ConditionFalse,
				v1alpha1.NamespaceCredentialsFailed,
				"Failed to create client for target cluster - "+err.Error(),
			)
			return err
		}
		namespace := ""
		for _, resource := range app.Status.Resources {
			if resource.Kind == "Namespace" {
				namespace = resource.Name
			}
		}

		secrets := corev1.SecretList{}

		if err := k8sClient.List(ctx, &secrets, &client.ListOptions{
			Namespace: namespace,
		}); err != nil {
			cti.SetNamespaceCredentialsCondition(
				metav1.ConditionFalse,
				v1alpha1.NamespaceCredentialsFailed,
				"Failed to list secrets - "+err.Error(),
			)
			return err
		}

		for _, secret := range secrets.Items {
			if secret.Type == corev1.SecretTypeServiceAccountToken && secret.Annotations["kubernetes.io/service-account.name"] == "user-sa" {
				token, err := utils.GetValueFromSecret(secret, "token")
				if err != nil {
					cti.SetNamespaceCredentialsCondition(
						metav1.ConditionFalse,
						v1alpha1.NamespaceCredentialsFailed,
						"Failed to get ServiceAccount token - "+err.Error(),
					)
					return err
				}
				tokenSecret := corev1.Secret{}
				tokenSecret.Name = cti.Name
				tokenSecret.Namespace = cti.Namespace
				tokenSecret.StringData = map[string]string{
					"token": token,
				}
				if err := utils.EnsureResourceExists(ctx, r.Client, &tokenSecret, false); err != nil {
					cti.SetNamespaceCredentialsCondition(
						metav1.ConditionFalse,
						v1alpha1.NamespaceCredentialsFailed,
						"Failed to create token secret - "+err.Error(),
					)
					return err
				}
				cti.Status.AdminPassword = &corev1.LocalObjectReference{
					Name: tokenSecret.Name,
				}
				cti.Status.APIserverURL = app.Spec.Destination.Server
				credentialsFound = true
				break
			}
		}

	}

	if credentialsFound {
		cti.Status.Phase = v1alpha1.ReadyPhase
		cti.Status.Message = "Environment is ready"
		cti.SetNamespaceCredentialsCondition(
			metav1.ConditionTrue,
			v1alpha1.NamespaceCredentialsSuccceeded,
			"Namespace credentials retrieved",
		)
	} else {
		cti.Status.Phase = v1alpha1.EnvironmentCredentialsRunningPhase
		cti.Status.Message = "Creating namespace credentials"
		cti.SetNamespaceCredentialsCondition(
			metav1.ConditionFalse,
			v1alpha1.NamespaceCredentialsPending,
			"Namespace credentials are being created",
		)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCredentials(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
) error {
	clusterSetupSucceededCondition := meta.FindStatusCondition(
		cti.Status.Conditions,
		string(v1alpha1.EnvironmentSetupSucceeded),
	)

	if clusterSetupSucceededCondition.Status == metav1.ConditionFalse {
		return nil
	}

	if cti.Status.APIserverURL == "" {
		kubeconfigSecret := corev1.Secret{}

		if err := r.Client.Get(
			ctx,
			client.ObjectKey{
				Name:      cti.GetKubeconfigRef(),
				Namespace: cti.Namespace,
			},
			&kubeconfigSecret,
		); err != nil {
			return err
		}

		kubeconfig := api.Config{}
		if err := yaml.Unmarshal(kubeconfigSecret.Data["kubeconfig"], &kubeconfig); err != nil {
			return err
		}
		cti.Status.APIserverURL = kubeconfig.Clusters[0].Cluster.Server
	}

	cti.Status.AdminPassword = &corev1.LocalObjectReference{
		Name: cti.GetKubeadminPassRef(),
	}
	cti.Status.Kubeconfig = &corev1.LocalObjectReference{
		Name: cti.GetKubeconfigRef(),
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
		[]string{cti.Name},
	)
	ctiNsLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINamespaceLabel,
		selection.Equals,
		[]string{cti.Namespace},
	)
	selector := labels.NewSelector().Add(*req, *ctiNameLabelReq, *ctiNsLabelReq)
	if err := r.Client.List(ctx, clusterSetupSecrets, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return err
	}
	for _, secret := range clusterSetupSecrets.Items {
		if !cti.ContainsSetupSecret(secret.Name) {
			cti.Status.ClusterSetupSecrets = append(
				cti.Status.ClusterSetupSecrets,
				corev1.LocalObjectReference{Name: secret.Name},
			)
		}
	}

	if err := r.ReconcileDynamicRoles(ctx, r.Client, cti); err != nil {
		cti.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf(
			"failed to reconcile role and role-bindings for users with cluster-templates-role - %q",
			err,
		)
		cti.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	cti.Status.Phase = v1alpha1.ReadyPhase
	cti.Status.Message = "Cluster is ready"

	return nil
}

func (*ClusterTemplateInstanceReconciler) ReconcileDynamicRoles(
	ctx context.Context,
	k8sClient client.Client,
	cti *v1alpha1.ClusterTemplateInstance,
) error {
	roleSubjects, err := cti.GetSubjectsWithClusterTemplateUserRole(
		ctx,
		k8sClient,
	)
	if err != nil {
		cti.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf("failed to get list of users - %q", err)
		cti.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	role, err := cti.CreateDynamicRole(ctx, k8sClient)

	if err != nil {
		cti.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf("failed to create role to access cluster secrets - %q", err)
		cti.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	_, err = cti.CreateDynamicRoleBinding(ctx, k8sClient, role, roleSubjects)
	if err != nil {
		cti.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf(
			"failed to create RoleBinding for %d subjects - %q",
			len(roleSubjects),
			err,
		)
		cti.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileCreateManagedCluster(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
	clusterTemplateLabels map[string]string,
) error {
	if !cti.PhaseCanExecute(
		v1alpha1.EnvironmentInstallSucceeded,
		v1alpha1.ManagedClusterCreated,
	) {
		return nil
	}

	if skipClusterRegistration {
		cti.SetManagedClusterCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCSkipped,
			"ManagedCluster skipped per ClusterTemplate spec",
		)
		return nil
	}

	if !r.EnableManagedCluster {
		cti.SetManagedClusterCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCSkipped,
			"ManagedCluster CRD does not exist, skipping",
		)
		return nil
	}

	CTIlog.Info(
		"Create ManagedCluster for clustertemplateinstance",
		"name",
		cti.Name,
	)

	if err := ocm.CreateManagedCluster(ctx, r.Client, cti, clusterTemplateLabels); err != nil {
		cti.SetManagedClusterCreatedCondition(
			metav1.ConditionFalse,
			v1alpha1.MCFailed,
			"Failed to create MangedCluster",
		)
		return err
	}
	mc, err := ocm.GetManagedCluster(ctx, r.Client, cti)
	if err != nil {
		_, ok := err.(*ocm.MCNotFoundError)
		if !ok {
			return err
		}
	}
	cti.Status.ManagedCluster = corev1.LocalObjectReference{
		Name: mc.Name,
	}
	cti.SetManagedClusterCreatedCondition(
		metav1.ConditionTrue,
		v1alpha1.MCCreated,
		"ManagedCluster created successfully",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileImportManagedCluster(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
) error {
	if !cti.PhaseCanExecute(
		v1alpha1.ManagedClusterCreated,
		v1alpha1.ManagedClusterImported,
	) {
		return nil
	}

	if skipClusterRegistration {
		cti.SetManagedClusterImportedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCImportSkipped,
			"ManagedCluster skipped per ClusterTemplate spec",
		)
		return nil
	}

	if !r.EnableManagedCluster {
		cti.SetManagedClusterImportedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCImportSkipped,
			"ManagedCluster CRD does not exist, skipping",
		)
		return nil
	}

	CTIlog.Info(
		"Import ManagedCluster of clustertemplateinstance",
		"name",
		cti.Name,
	)

	imported, err := ocm.ImportManagedCluster(ctx, r.Client, cti)
	if err != nil {
		cti.SetManagedClusterImportedCondition(
			metav1.ConditionFalse,
			v1alpha1.MCImportFailed,
			"Failed to import ManagedCluster",
		)
		return err
	}
	// TODO mc import err state ?
	if imported {
		cti.SetManagedClusterImportedCondition(
			metav1.ConditionTrue,
			v1alpha1.MCImported,
			"ManagedCluster imported successfully",
		)
	} else {
		cti.Status.Phase = v1alpha1.ManagedClusterImportingPhase
		cti.Status.Message = "ManagedCluster is importing"
		cti.SetManagedClusterImportedCondition(
			metav1.ConditionFalse,
			v1alpha1.MCImporting,
			"ManagedCluster is importing",
		)
	}
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileCreateKlusterlet(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
) error {
	if !cti.PhaseCanExecute(v1alpha1.ManagedClusterImported) {
		return nil
	}

	if !(r.EnableKlusterlet && !skipClusterRegistration) {
		cti.SetKlusterletCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.KlusterletSkipped,
			"KlusterletAddonConfig CRD does not exist, skipping",
		)
		return nil
	}

	CTIlog.Info(
		"Create KlusterletAddonConfig for clustertemplateinstance",
		"name",
		cti.Name,
	)

	if err := ocm.CreateKlusterletAddonConfig(ctx, r.Client, cti); err != nil {
		cti.SetKlusterletCreatedCondition(
			metav1.ConditionFalse,
			v1alpha1.KlusterletFailed,
			fmt.Sprint("Failed to create KlusterletAddonConfig - "+err.Error()),
		)
		return err
	}
	cti.SetKlusterletCreatedCondition(
		metav1.ConditionTrue,
		v1alpha1.KlusterletCreated,
		"KlusterletAddonConfig created successfully",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileConsoleURL(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
) error {
	if !cti.PhaseCanExecute(v1alpha1.ManagedClusterImported) {
		return nil
	}
	if skipClusterRegistration || !r.EnableManagedCluster {
		cti.SetConsoleURLCondition(
			metav1.ConditionTrue,
			v1alpha1.ConsoleURLSkipped,
			"ManagedCluster will not be created, skipping",
		)
		return nil
	}
	mc, err := ocm.GetManagedCluster(ctx, r.Client, cti)
	if err != nil {
		_, ok := err.(*ocm.MCNotFoundError)
		if !ok {
			cti.SetConsoleURLCondition(
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
				cti.Status.ConsoleURL = cc.Value
				cti.SetConsoleURLCondition(
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
	cti *v1alpha1.ClusterTemplateInstance,
	skipClusterRegistration bool,
) error {
	if !cti.PhaseCanExecute(
		v1alpha1.ManagedClusterImported,
		v1alpha1.ArgoClusterAdded,
	) {
		return nil
	}

	if err := clustersetup.AddClusterToArgo(
		ctx,
		r.Client,
		cti,
		clustersetup.GetClientForCluster,
		ArgoCDNamespace,
		r.EnableManagedCluster && !skipClusterRegistration,
		LoginAttemptTimeout.Duration,
	); err != nil {
		_, ok := err.(*clustersetup.LoginError)

		if ok {
			cti.SetArgoClusterAddedCondition(
				metav1.ConditionFalse,
				v1alpha1.ArgoClusterLoginPending,
				fmt.Sprintf("Waiting for login to be successful - %q", err),
			)
			return err
		}

		cti.SetArgoClusterAddedCondition(
			metav1.ConditionFalse,
			v1alpha1.ArgoClusterFailed,
			fmt.Sprintf("Failed to add cluster to argo - %q", err),
		)
		return err
	}
	cti.SetArgoClusterAddedCondition(
		metav1.ConditionTrue,
		v1alpha1.ArgoClusterCreated,
		"Cluster added to argo successfully",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileEnvironmentSetupCreate(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	clusterSetup []string,
	isNs bool,
) error {
	prevPhase := v1alpha1.ArgoClusterAdded
	if isNs {
		prevPhase = v1alpha1.EnvironmentInstallSucceeded
	}

	if !cti.PhaseCanExecute(
		prevPhase,
		v1alpha1.EnvironmentSetupCreated,
	) {
		return nil
	}

	if len(clusterSetup) == 0 {
		cti.SetEnvironmentSetupCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.EnvironmentSetupNotSpecified,
			"No cluster setup specified",
		)
		return nil
	}

	CTIlog.Info(
		"Create cluster setup for clustertemplateinstance",
		"name",
		cti.Name,
	)

	targetServer := ""
	namespace := ""
	if isNs {
		app, err := cti.GetDay1Application(ctx, r.Client, ArgoCDNamespace)
		if err != nil {
			return err
		}
		targetServer = app.Spec.Destination.Server
		for _, resource := range app.Status.Resources {
			if resource.Kind == "Namespace" {
				namespace = resource.Name
			}
		}
	}

	if err := cti.CreateDay2Applications(
		ctx,
		r.Client,
		ArgoCDNamespace,
		clusterSetup,
		isNs,
		namespace,
		targetServer,
	); err != nil {
		cti.SetEnvironmentSetupCreatedCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentSetupCreationFailed,
			fmt.Sprintf("Failed to create cluster setup - %q", err),
		)
		return err
	}
	cti.SetEnvironmentSetupCreatedCondition(
		metav1.ConditionTrue,
		v1alpha1.SetupCreated,
		"Cluster setup created",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileEnvironmentSetup(
	ctx context.Context,
	cti *v1alpha1.ClusterTemplateInstance,
	clusterSetup []string,
) error {

	if !cti.PhaseCanExecute(
		v1alpha1.EnvironmentSetupCreated,
		v1alpha1.EnvironmentSetupSucceeded,
	) {
		return nil
	}

	if len(clusterSetup) == 0 {
		cti.SetEnvironmentSetupSucceededCondition(
			metav1.ConditionTrue,
			v1alpha1.EnvironmentSetupNotDefined,
			"No environment setup defined",
		)
		return nil
	}

	CTIlog.Info(
		"reconcile environment setup for clustertemplateinstance",
		"name",
		cti.Name,
	)
	applications, err := cti.GetDay2Applications(ctx, r.Client, ArgoCDNamespace)

	if err != nil {
		cti.SetEnvironmentSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentSetupFetchFailed,
			fmt.Sprintf("Failed to list setup apps - %q", err),
		)
		return err
	}

	if len(applications.Items) == 0 {
		cti.SetEnvironmentSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentSetupAppsNotFound,
			"Failed to find environment setup apps",
		)
		return fmt.Errorf("failed to find environment setup apps")
	}

	cti.Status.Phase = v1alpha1.EnvironmentSetupRunningPhase
	cti.Status.Message = "Environment setup is running"
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

	cti.Status.ClusterSetup = &clusterSetupStatus

	if allSynced {
		cti.SetEnvironmentSetupSucceededCondition(
			metav1.ConditionTrue,
			v1alpha1.SetupSucceeded,
			"Environment setup succeeded",
		)
	} else if len(errorSetups) > 0 {
		msg := fmt.Sprintf("Following environment setups are in error state - %v", errorSetups)
		cti.SetEnvironmentSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentSetupError,
			msg,
		)
		cti.Status.Phase = v1alpha1.EnvironmentSetupFailedPhase
		cti.Status.Message = msg
	} else if len(degradedSetups) > 0 {
		msg := fmt.Sprintf("Following environment setups are in degraded state - %v", degradedSetups)
		cti.SetEnvironmentSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentSetupDegraded,
			msg,
		)
		cti.Status.Phase = v1alpha1.EnvironmentSetupDegradedPhase
		cti.Status.Message = msg
	} else {
		cti.SetEnvironmentSetupSucceededCondition(
			metav1.ConditionFalse,
			v1alpha1.EnvironmentSetupRunning,
			"Environment setup is running",
		)
		cti.Status.Phase = v1alpha1.EnvironmentSetupRunningPhase
		cti.Status.Message = "Environment setup is running"
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
