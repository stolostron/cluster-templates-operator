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

package v1alpha1

import (
	"context"

	"gopkg.in/yaml.v3"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/kubernetes-client/go-base/config/api"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CTIClusterTargetVar     = "${new_cluster}"
	CTIInstanceNamespaceVar = "${instance_ns}"
)

func (i *ClusterTemplateInstance) GetKubeadminPassRef() string {
	return i.Name + "-admin-password"
}

func (i *ClusterTemplateInstance) GetKubeconfigRef() string {
	return i.Name + "-admin-kubeconfig"
}

func (i *ClusterTemplateInstance) GetOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		Kind:       "ClusterTemplateInstance",
		APIVersion: APIVersion,
		Name:       i.Name,
		UID:        i.UID,
	}
}

func (i *ClusterTemplateInstance) GetDay1Application(
	ctx context.Context,
	k8sClient client.Client,
) (*argo.Application, error) {
	apps := &argo.ApplicationList{}

	ctiNameLabelReq, _ := labels.NewRequirement(
		CTINameLabel,
		selection.Equals,
		[]string{i.Name},
	)
	ctiNsLabelReq, _ := labels.NewRequirement(
		CTINamespaceLabel,
		selection.Equals,
		[]string{i.Namespace},
	)
	ctiSetupReq, _ := labels.NewRequirement(
		CTISetupLabel,
		selection.DoesNotExist,
		[]string{},
	)
	selector := labels.NewSelector().Add(*ctiNameLabelReq, *ctiNsLabelReq, *ctiSetupReq)

	if err := k8sClient.List(ctx, apps, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     ArgoNamespace,
	}); err != nil {
		return nil, err
	}

	if len(apps.Items) == 0 {

		err := apierrors.NewNotFound(schema.GroupResource{
			Group:    argo.ApplicationSchemaGroupVersionKind.Group,
			Resource: argo.ApplicationSchemaGroupVersionKind.Kind,
		}, i.Namespace+"/"+i.Name)
		return nil, err
	}
	return &apps.Items[0], nil
}

func (i *ClusterTemplateInstance) CreateDay1Application(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplate ClusterTemplate,
) error {
	argoApp, err := i.GetDay1Application(ctx, k8sClient)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if argoApp != nil {
		return nil
	}

	params, err := i.GetHelmParameters(clusterTemplate, "")

	if err != nil {
		return err
	}

	appSpec := clusterTemplate.Spec.ClusterDefinition

	if len(params) > 0 {
		if appSpec.Source.Helm == nil {
			appSpec.Source.Helm = &argo.ApplicationSourceHelm{}
		}
		appSpec.Source.Helm.Parameters = params
	}

	if appSpec.Destination.Namespace == CTIInstanceNamespaceVar {
		appSpec.Destination.Namespace = i.Namespace
	}

	argoApp = &argo.Application{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: i.Name + "-",
			Namespace:    ArgoNamespace,
			Finalizers: []string{
				argo.ResourcesFinalizerName,
			},
			Labels: map[string]string{
				CTINameLabel:      i.Name,
				CTINamespaceLabel: i.Namespace,
			},
		},
		Spec: appSpec,
	}
	return k8sClient.Create(ctx, argoApp)
}

func (i *ClusterTemplateInstance) GetDay2Applications(
	ctx context.Context,
	k8sClient client.Client,
) (*argo.ApplicationList, error) {
	applications := &argo.ApplicationList{}

	ctiNameLabelReq, _ := labels.NewRequirement(
		CTINameLabel,
		selection.Equals,
		[]string{i.Name},
	)
	ctiNsLabelReq, _ := labels.NewRequirement(
		CTINamespaceLabel,
		selection.Equals,
		[]string{i.Namespace},
	)
	applicationLabelReq, _ := labels.NewRequirement(
		CTISetupLabel,
		selection.Exists,
		[]string{},
	)
	selector := labels.NewSelector().Add(*ctiNameLabelReq, *ctiNsLabelReq, *applicationLabelReq)

	err := k8sClient.List(
		ctx,
		applications,
		&client.ListOptions{
			LabelSelector: selector,
			Namespace:     ArgoNamespace,
		},
	)
	return applications, err
}

func (i *ClusterTemplateInstance) CreateDay2Applications(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplate ClusterTemplate,
) error {
	log := ctrl.LoggerFrom(ctx)
	apps, err := i.GetDay2Applications(ctx, k8sClient)

	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	log.Info("Create day2 applications")

	kubeconfigSecret := corev1.Secret{}
	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{
			Name:      i.GetKubeconfigRef(),
			Namespace: i.Namespace,
		},
		&kubeconfigSecret,
	); err != nil {
		return err
	}
	kubeconfig := api.Config{}
	if err := yaml.Unmarshal(kubeconfigSecret.Data["kubeconfig"], &kubeconfig); err != nil {
		return err
	}

	for _, clusterSetup := range clusterTemplate.Spec.ClusterSetup {
		setupAlreadyExists := false
		for _, app := range apps.Items {
			val := app.GetLabels()[CTISetupLabel]
			if val == clusterSetup.Name {
				setupAlreadyExists = true
			}
		}
		if !setupAlreadyExists {
			params, err := i.GetHelmParameters(clusterTemplate, clusterSetup.Name)

			if err != nil {
				return err
			}

			if len(params) > 0 {
				if clusterSetup.Spec.Source.Helm == nil {
					clusterSetup.Spec.Source.Helm = &argo.ApplicationSourceHelm{}
				}
				clusterSetup.Spec.Source.Helm.Parameters = params
			}

			if clusterSetup.Spec.Destination.Server == CTIClusterTargetVar {
				clusterSetup.Spec.Destination.Server = kubeconfig.Clusters[0].Cluster.Server
			}

			argoApp := argo.Application{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: i.Name + "-",
					Namespace:    ArgoNamespace,
					Labels: map[string]string{
						CTINameLabel:      i.Name,
						CTINamespaceLabel: i.Namespace,
						CTISetupLabel:     clusterSetup.Name,
					},
				},
				Spec: clusterSetup.Spec,
			}
			if err := k8sClient.Create(ctx, &argoApp); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *ClusterTemplateInstance) GetHelmParameters(
	ct ClusterTemplate,
	day2Name string,
) ([]argo.HelmParameter, error) {

	params := []argo.HelmParameter{}

	if day2Name == "" {
		if ct.Spec.ClusterDefinition.Source.Helm != nil {
			params = ct.Spec.ClusterDefinition.Source.Helm.Parameters
		}
	} else {
		for _, setup := range ct.Spec.ClusterSetup {
			if setup.Name == day2Name && setup.Spec.Source.Helm != nil {
				params = setup.Spec.Source.Helm.Parameters
			}
		}
	}

	for _, param := range i.Spec.Parameters {
		if param.ClusterSetup == day2Name {
			added := false
			for _, ctParam := range params {
				if ctParam.Name == param.Name {
					ctParam.Value = param.Value
					added = true
				}
			}
			if !added {
				params = append(params, argo.HelmParameter{
					Name:  param.Name,
					Value: param.Value,
				})
			}
		}
	}

	return params, nil
}

func (i *ClusterTemplateInstance) GetSubjectsWithClusterTemplateUserRole(
	ctx context.Context, k8sClient client.Client) ([]rbacv1.Subject, error) {
	allRoleBindingsInNamespace := &rbacv1.RoleBindingList{}

	if err := k8sClient.List(ctx, allRoleBindingsInNamespace, &client.ListOptions{
		Namespace: i.Namespace,
	}); err != nil {
		return nil, err
	}

	result := []rbacv1.Subject{}
	keys := make(map[string]bool)
	for _, rb := range allRoleBindingsInNamespace.Items {
		if rb.RoleRef.Kind == "ClusterRole" && rb.RoleRef.Name == "cluster-templates-user" {
			for _, subject := range rb.Subjects {
				key := subject.Kind + "*" + subject.Name
				if _, value := keys[key]; !value {
					keys[key] = true
					result = append(result, subject)
				}
			}
		}
	}

	return result, nil
}

func (i *ClusterTemplateInstance) CreateDynamicRole(
	ctx context.Context, k8sClient client.Client) (*rbacv1.Role, error) {
	roleName := i.Name + "-role-managed"
	roleNamespace := i.Namespace
	secretNames := []string{i.GetKubeadminPassRef(), i.GetKubeconfigRef()}

	existingRole := &rbacv1.Role{}
	err := k8sClient.Get(
		ctx,
		client.ObjectKey{
			Name:      roleName,
			Namespace: roleNamespace,
		},
		existingRole,
	)

	desiredRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            roleName,
			Namespace:       roleNamespace,
			OwnerReferences: []metav1.OwnerReference{i.GetOwnerReference()},
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups:     []string{""},
			Verbs:         []string{"get"},
			Resources:     []string{"secrets"},
			ResourceNames: secretNames,
		}},
	}

	if err == nil {
		// Results in no action if there is no difference in content
		return desiredRole, k8sClient.Update(ctx, desiredRole)
	} else if apierrors.IsNotFound(err) {
		return desiredRole, k8sClient.Create(ctx, desiredRole)
	} else {
		return nil, err
	}
}

func (i *ClusterTemplateInstance) CreateDynamicRoleBinding(
	ctx context.Context, k8sClient client.Client,
	role *rbacv1.Role, roleSubjects []rbacv1.Subject) (*rbacv1.RoleBinding, error) {
	roleBindingName := i.Name + "-rolebinding-managed"
	roleBindingNamespace := i.Namespace

	existingRoleBinding := &rbacv1.RoleBinding{}
	err := k8sClient.Get(
		ctx,
		client.ObjectKey{
			Name:      roleBindingName,
			Namespace: roleBindingNamespace,
		},
		existingRoleBinding,
	)

	desiredRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            roleBindingName,
			Namespace:       roleBindingNamespace,
			OwnerReferences: []metav1.OwnerReference{i.GetOwnerReference()},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     role.Name,
		},
		Subjects: roleSubjects,
	}

	if err == nil {
		return desiredRoleBinding, k8sClient.Update(ctx, desiredRoleBinding)
	} else if apierrors.IsNotFound(err) {
		return desiredRoleBinding, k8sClient.Create(ctx, desiredRoleBinding)
	} else {
		return nil, err
	}
}
