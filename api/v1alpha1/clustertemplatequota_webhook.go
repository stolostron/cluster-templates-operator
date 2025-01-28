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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var clustertemplatequotalog = logf.Log.WithName("clustertemplatequota-resource")
var quotaControllerClient client.Client

func (r *ClusterTemplateQuota) SetupWebhookWithManager(mgr ctrl.Manager) error {
	quotaControllerClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-clustertemplate-openshift-io-v1alpha1-clustertemplatequota,mutating=false,failurePolicy=fail,sideEffects=None,groups=clustertemplate.openshift.io,resources=clustertemplatequotas,verbs=create;update,versions=v1alpha1,name=vclustertemplatequota.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ClusterTemplateQuota{}

func (r *ClusterTemplateQuota) ValidateCreate() (admission.Warnings, error) {
	clustertemplatequotalog.Info("validate create", "name", r.Name)

	quotas := ClusterTemplateQuotaList{}

	opts := []client.ListOption{client.InNamespace(r.Namespace)}

	if err := quotaControllerClient.List(context.TODO(), &quotas, opts...); err != nil {
		return []string{}, fmt.Errorf("failed to list cluster quotas - %q", err)
	}

	if len(quotas.Items) > 0 {
		return []string{}, fmt.Errorf("cluster quota for this namespace already exists")
	}

	templates := ClusterTemplateList{}

	if err := quotaControllerClient.List(context.TODO(), &templates); err != nil {
		return []string{}, fmt.Errorf("failed to list cluster templates - %q", err)
	}

	for _, allowedTemplate := range r.Spec.AllowedTemplates {
		templateFound := false
		for _, template := range templates.Items {
			if template.Name == allowedTemplate.Name {
				templateFound = true
			}
		}
		if !templateFound {
			return []string{}, fmt.Errorf("template '%s' does not exist", allowedTemplate.Name)
		}
	}
	return []string{}, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateQuota) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	clustertemplatequotalog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return []string{}, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateQuota) ValidateDelete() (admission.Warnings, error) {
	clustertemplatequotalog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return []string{}, nil
}
