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
var clustertemplateinstancelog = logf.Log.WithName("clustertemplateinstance-resource")

var instanceControllerClient client.Client

func (r *ClusterTemplateInstance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	instanceControllerClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-clustertemplate-rawagner-com-v1alpha1-clustertemplateinstance,mutating=true,failurePolicy=fail,sideEffects=None,groups=clustertemplate.rawagner.com,resources=clustertemplateinstances,verbs=create;update,versions=v1alpha1,name=mclustertemplateinstance.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ClusterTemplateInstance{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ClusterTemplateInstance) Default() {
	clustertemplateinstancelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-clustertemplate-rawagner-com-v1alpha1-clustertemplateinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=clustertemplate.rawagner.com,resources=clustertemplateinstances,verbs=create;update,versions=v1alpha1,name=vclustertemplateinstance.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ClusterTemplateInstance{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateInstance) ValidateCreate() error {
	clustertemplateinstancelog.Info("validate create", "name", r.Name)

	quotas := ClusterTemplateQuotaList{}

	opts := []client.ListOption{
		client.InNamespace(r.Namespace),
	}

	err := instanceControllerClient.List(context.TODO(), &quotas, opts...)
	if err != nil {
		return errors.New("could not find quota for namespace")
	}

	quota := quotas.Items[0]

	if quota.Status.InstancesCount >= quota.Spec.Quota {
		return errors.New("not enough quota")
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateInstance) ValidateUpdate(old runtime.Object) error {
	clustertemplateinstancelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterTemplateInstance) ValidateDelete() error {
	clustertemplateinstancelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
