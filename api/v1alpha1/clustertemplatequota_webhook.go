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
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var clustertemplatequotalog = logf.Log.WithName("clustertemplatequota-resource")

func (r *ClusterTemplateQuota) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-clustertemplate-rawagner-com-v1alpha1-clustertemplatequota,mutating=true,failurePolicy=fail,sideEffects=None,groups=clustertemplate.rawagner.com,resources=clustertemplatequotas,verbs=create;update,versions=v1alpha1,name=mclustertemplatequota.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ClusterTemplateQuota{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ClusterTemplateQuota) Default() {
	clustertemplatequotalog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-clustertemplate-rawagner-com-v1alpha1-clustertemplatequota,mutating=false,failurePolicy=fail,sideEffects=None,groups=clustertemplate.rawagner.com,resources=clustertemplatequotas,verbs=create;update,versions=v1alpha1,name=vclustertemplatequota.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ClusterTemplateQuota{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateQuota) ValidateCreate() error {
	clustertemplatequotalog.Info("validate create", "name", r.Name)

	ownerFound := false
	for _, owner := range r.OwnerReferences {
		if owner.APIVersion == APIVersion && owner.Kind == "ClusterTemplate" {
			ownerFound = true
			break
		}
	}

	if !ownerFound {
		return errors.New("missing owner reference for ClusterTemplate")
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
