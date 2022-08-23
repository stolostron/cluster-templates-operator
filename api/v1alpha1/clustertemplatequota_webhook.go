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
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var clustertemplatequotalog = logf.Log.WithName("clustertemplatequota-resource")
var quotaControllerClient client.Client

func (r *ClusterTemplateQuota) SetupWebhookWithManager(mgr ctrl.Manager) error {
	quotaControllerClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-clustertemplate-rawagner-com-v1alpha1-clustertemplatequota,mutating=false,failurePolicy=fail,sideEffects=None,groups=clustertemplate.rawagner.com,resources=clustertemplatequotas,verbs=create;update,versions=v1alpha1,name=vclustertemplatequota.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ClusterTemplateQuota{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateQuota) ValidateCreate() error {
	clustertemplatequotalog.Info("validate create", "name", r.Name)

	templates := ClusterTemplateList{}

	opts := []client.ListOption{}

	err := quotaControllerClient.List(context.TODO(), &templates, opts...)
	if err != nil {
		return errors.New("could not find template")
	}

	for _, allowedTemplate := range r.Spec.AllowedTemplates {
		templateFound := false
		for _, template := range templates.Items {
			if template.Name == allowedTemplate.Name {
				templateFound = true
			}
		}
		if !templateFound {
			return errors.New("template not found")
		}
	}

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateQuota) ValidateUpdate(old runtime.Object) error {
	clustertemplatequotalog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateQuota) ValidateDelete() error {
	clustertemplatequotalog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
