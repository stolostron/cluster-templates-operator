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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var clustertemplateinstancelog = logf.Log.WithName("clustertemplateinstance-resource")
var instanceControllerClient client.Client

func (r *ClusterTemplateInstance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	instanceControllerClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-clustertemplate-openshift-io-v1alpha1-clustertemplateinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=clustertemplate.openshift.io,resources=clustertemplateinstances,verbs=create;update,versions=v1alpha1,name=vclustertemplateinstance.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ClusterTemplateInstance{}

func (r *ClusterTemplateInstance) ValidateCreate() error {
	clustertemplateinstancelog.Info("validate create", "name", r.Name)

	if err := r.checkQuota(); err != nil {
		return err
	}
	if err := r.checkProps(); err != nil {
		return err
	}

	return nil
}

func (r *ClusterTemplateInstance) checkProps() error {
	template := ClusterTemplate{}
	if err := instanceControllerClient.Get(
		context.TODO(),
		client.ObjectKey{Name: r.Spec.ClusterTemplateRef},
		&template,
	); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("cluster template '%v' not found", r.Spec.ClusterTemplateRef)
		}
		return fmt.Errorf("failed to get cluster template - %q", err)
	}

	// TODO check values
	return nil

}

func (r *ClusterTemplateInstance) checkQuota() error {
	quotas := ClusterTemplateQuotaList{}
	opts := []client.ListOption{
		client.InNamespace(r.Namespace),
	}
	err := instanceControllerClient.List(context.TODO(), &quotas, opts...)
	if err != nil || len(quotas.Items) == 0 {
		return fmt.Errorf("could not find quota for namespace")
	}

	templates := ClusterTemplateList{}
	err = instanceControllerClient.List(context.TODO(), &templates)
	if err != nil {
		return fmt.Errorf("could not list cluster templates - %q", err)
	}

	templateIdx := -1
	for index := range templates.Items {
		if templates.Items[index].Name == r.Spec.ClusterTemplateRef {
			templateIdx = index
			break
		}
	}

	if templateIdx == -1 {
		return fmt.Errorf("could not find cluster template")
	}

	templateAllowed := false
	for _, quota := range quotas.Items {
		if quota.Spec.Budget > 0 &&
			quota.Spec.Budget < quota.Status.BudgetSpent+templates.Items[templateIdx].Spec.Cost {
			return fmt.Errorf("cost is too much")
		}

		maxAllowed := 0
		for _, tempInstance := range quota.Spec.AllowedTemplates {
			if tempInstance.Name == r.Spec.ClusterTemplateRef {
				templateAllowed = true
				maxAllowed = tempInstance.Count
			}
		}

		if maxAllowed > 0 {
			for _, tempInstance := range quota.Status.TemplateInstances {
				if tempInstance.Name == r.Spec.ClusterTemplateRef {
					if tempInstance.Count >= maxAllowed {
						return fmt.Errorf("not enough quota")
					}
				}
			}
		}
	}

	if !templateAllowed {
		return fmt.Errorf("template not allowed")
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
