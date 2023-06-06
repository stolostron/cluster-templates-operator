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
	"encoding/json"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/kubernetes-client/go-base/config/api"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	CTIlog = logf.Log.WithName("cti-utils")
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
	argoCDNamespace string,
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
		Namespace:     argoCDNamespace,
	}); err != nil {
		return nil, err
	}

	if len(apps.Items) == 0 {
		err := apierrors.NewNotFound(schema.GroupResource{
			Group:    argo.ApplicationSchemaGroupVersionKind.Group,
			Resource: argo.ApplicationSchemaGroupVersionKind.Kind,
		}, argoCDNamespace+"/"+i.Name)
		return nil, err
	}
	return &apps.Items[0], nil
}

func (i *ClusterTemplateInstance) DeleteDay1Application(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
	clusterDefinition string,
) error {
	appSet := &argo.ApplicationSet{}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{Name: clusterDefinition, Namespace: argoCDNamespace},
		appSet,
	); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	var generators []argo.ApplicationSetGenerator
	for _, g := range appSet.Spec.Generators {
		if g.List != nil && g.List.Template.Labels[CTINameLabel] == i.Name && g.List.Template.Labels[CTINamespaceLabel] == i.Namespace {
			continue
		}
		generators = append(generators, g)
	}

	appSet.Spec.Generators = generators

	return k8sClient.Update(ctx, appSet)
}

func (i *ClusterTemplateInstance) DeleteDay2Application(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
	clusterSetup []string,
) error {
	appsets, err := getDay2Appsets(ctx, k8sClient, argoCDNamespace, clusterSetup, false)
	if err != nil {
		return err
	}

	for _, appSet := range appsets {
		var generators []argo.ApplicationSetGenerator
		for _, g := range appSet.Spec.Generators {
			if g.List != nil && g.List.Template.Labels[CTINameLabel] == i.Name && g.List.Template.Labels[CTINamespaceLabel] == i.Namespace {
				continue
			}
			generators = append(generators, g)
		}

		appSet.Spec.Generators = generators

		if err := k8sClient.Update(ctx, appSet); err != nil {
			return err
		}
	}

	return nil
}

func (i *ClusterTemplateInstance) UpdateApplicationSet(
	ctx context.Context,
	k8sClient client.Client,
	appSet *argo.ApplicationSet,
	server string,
	isDay2 bool,
) error {
	for _, g := range appSet.Spec.Generators {
		if g.List != nil && g.List.Template.Labels[CTINameLabel] == i.Name && g.List.Template.Labels[CTINamespaceLabel] == i.Namespace {
			return nil
		}
	}

	name := string(i.UID)
	if isDay2 {
		name = name + "-" + appSet.Name
	}

	elements, _ := json.Marshal(map[string]string{"url": server, "instance_ns": i.Namespace})
	gen := argo.ApplicationSetGenerator{List: &argo.ListGenerator{
		Elements: []apiextensionsv1.JSON{{Raw: elements}},
		Template: argo.ApplicationSetTemplate{
			ApplicationSetTemplateMeta: argo.ApplicationSetTemplateMeta{
				Name: name,
				Finalizers: []string{
					argo.ResourcesFinalizerName,
				},
				Labels: map[string]string{
					CTINameLabel:      i.Name,
					CTINamespaceLabel: i.Namespace,
				},
			},
		},
	},
	}

	if appSet.Spec.Template.Spec.Source.Chart != "" {
		params, err := i.GetHelmParameters(appSet)
		if err != nil {
			return err
		}

		gen.List.Template.Spec = argo.ApplicationSpec{
			Source: argo.ApplicationSource{
				Helm: &argo.ApplicationSourceHelm{
					Parameters: params,
				},
			},
		}
	}

	if isDay2 {
		gen.List.Template.ApplicationSetTemplateMeta.Labels[CTISetupLabel] = ""
	}
	appSet.Spec.Generators = append(appSet.Spec.Generators, gen)

	return k8sClient.Update(ctx, appSet)
}

func labelDestionationNamespace(ctx context.Context, appSet *argo.ApplicationSet, k8sClient client.Client, argoCDNamespace string) error {
	if appSet.Spec.Template.Spec.Destination.Namespace == "" {
		return nil
	}
	// Set the Argo label for the destination namespace:
	ns := &corev1.Namespace{}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{Name: appSet.Spec.Template.Spec.Destination.Namespace},
		ns,
	); err != nil {
		return err
	}

	if l, lOk := ns.Labels["argocd.argoproj.io/managed-by"]; !lOk || l != argoCDNamespace {
		ns.Labels["argocd.argoproj.io/managed-by"] = argoCDNamespace
		if err := k8sClient.Update(ctx, ns); err != nil {
			return err
		}
	}
	return nil
}

func (i *ClusterTemplateInstance) CreateDay1Application(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
	labelNamespace bool,
	clusterDefinition string,
) error {
	appSet := &argo.ApplicationSet{}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{Name: clusterDefinition, Namespace: argoCDNamespace},
		appSet,
	); err != nil {
		return err
	}

	if labelNamespace {
		if err := labelDestionationNamespace(ctx, appSet, k8sClient, argoCDNamespace); err != nil {
			return err
		}
	}

	return i.UpdateApplicationSet(ctx, k8sClient, appSet, "https://kubernetes.default.svc", false)
}

func (i *ClusterTemplateInstance) GetDay2Applications(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
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
			Namespace:     argoCDNamespace,
		},
	)
	return applications, err
}

func (i *ClusterTemplateInstance) CreateDay2Applications(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
	labelNamespace bool,
	clusterSetup []string,
) error {
	appsets, err := getDay2Appsets(ctx, k8sClient, argoCDNamespace, clusterSetup, true)
	if err != nil {
		return err
	}

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

	for _, appset := range appsets {
		if labelNamespace {
			if err := labelDestionationNamespace(ctx, appset, k8sClient, argoCDNamespace); err != nil {
				return err
			}
		}

		if err := i.UpdateApplicationSet(ctx, k8sClient, appset, kubeconfig.Clusters[0].Cluster.Server, true); err != nil {
			return err
		}
	}

	return nil
}

func getDay2Appsets(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
	clusterSetup []string,
	failOnMissing bool,
) ([]*argo.ApplicationSet, error) {
	appSets := []*argo.ApplicationSet{}
	appSet := &argo.ApplicationSet{}
	for _, cs := range clusterSetup {
		if err := k8sClient.Get(
			ctx,
			types.NamespacedName{Name: cs, Namespace: argoCDNamespace},
			appSet,
		); err != nil {
			if !failOnMissing && apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		appSets = append(appSets, appSet)
	}

	return appSets, nil
}

func (i *ClusterTemplateInstance) GetHelmParameters(
	appset *argo.ApplicationSet,
) ([]argo.HelmParameter, error) {
	params := []argo.HelmParameter{}
	if appset != nil && appset.Spec.Template.Spec.Source.Helm != nil {
		params = appset.Spec.Template.Spec.Source.Helm.Parameters
	}
	for _, param := range i.Spec.Parameters {
		if param.ApplicationSet == appset.Name {
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
	for _, clusterSetupSecrets := range i.Status.ClusterSetupSecrets {
		secretNames = append(secretNames, clusterSetupSecrets.Name)
	}

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

func (i *ClusterTemplateInstance) ContainsSetupSecret(secretName string) bool {
	for _, secret := range i.Status.ClusterSetupSecrets {
		if secret.Name == secretName {
			return true
		}
	}

	return false
}
