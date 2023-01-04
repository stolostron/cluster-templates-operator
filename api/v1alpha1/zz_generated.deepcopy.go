//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AllowedTemplate) DeepCopyInto(out *AllowedTemplate) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AllowedTemplate.
func (in *AllowedTemplate) DeepCopy() *AllowedTemplate {
	if in == nil {
		return nil
	}
	out := new(AllowedTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterDefinitionSchema) DeepCopyInto(out *ClusterDefinitionSchema) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterDefinitionSchema.
func (in *ClusterDefinitionSchema) DeepCopy() *ClusterDefinitionSchema {
	if in == nil {
		return nil
	}
	out := new(ClusterDefinitionSchema)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterSetup) DeepCopyInto(out *ClusterSetup) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterSetup.
func (in *ClusterSetup) DeepCopy() *ClusterSetup {
	if in == nil {
		return nil
	}
	out := new(ClusterSetup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterSetupSchema) DeepCopyInto(out *ClusterSetupSchema) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterSetupSchema.
func (in *ClusterSetupSchema) DeepCopy() *ClusterSetupSchema {
	if in == nil {
		return nil
	}
	out := new(ClusterSetupSchema)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterSetupStatus) DeepCopyInto(out *ClusterSetupStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterSetupStatus.
func (in *ClusterSetupStatus) DeepCopy() *ClusterSetupStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterSetupStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplate) DeepCopyInto(out *ClusterTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplate.
func (in *ClusterTemplate) DeepCopy() *ClusterTemplate {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateInstance) DeepCopyInto(out *ClusterTemplateInstance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateInstance.
func (in *ClusterTemplateInstance) DeepCopy() *ClusterTemplateInstance {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateInstance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTemplateInstance) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateInstanceList) DeepCopyInto(out *ClusterTemplateInstanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterTemplateInstance, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateInstanceList.
func (in *ClusterTemplateInstanceList) DeepCopy() *ClusterTemplateInstanceList {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateInstanceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTemplateInstanceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateInstanceSpec) DeepCopyInto(out *ClusterTemplateInstanceSpec) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make([]Parameter, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateInstanceSpec.
func (in *ClusterTemplateInstanceSpec) DeepCopy() *ClusterTemplateInstanceSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateInstanceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateInstanceStatus) DeepCopyInto(out *ClusterTemplateInstanceStatus) {
	*out = *in
	if in.ClusterTemplateSpec != nil {
		in, out := &in.ClusterTemplateSpec, &out.ClusterTemplateSpec
		*out = new(ClusterTemplateSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.AdminPassword != nil {
		in, out := &in.AdminPassword, &out.AdminPassword
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.Kubeconfig != nil {
		in, out := &in.Kubeconfig, &out.Kubeconfig
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ClusterSetup != nil {
		in, out := &in.ClusterSetup, &out.ClusterSetup
		*out = new([]ClusterSetupStatus)
		if **in != nil {
			in, out := *in, *out
			*out = make([]ClusterSetupStatus, len(*in))
			copy(*out, *in)
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateInstanceStatus.
func (in *ClusterTemplateInstanceStatus) DeepCopy() *ClusterTemplateInstanceStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateInstanceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateList) DeepCopyInto(out *ClusterTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateList.
func (in *ClusterTemplateList) DeepCopy() *ClusterTemplateList {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateQuota) DeepCopyInto(out *ClusterTemplateQuota) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateQuota.
func (in *ClusterTemplateQuota) DeepCopy() *ClusterTemplateQuota {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateQuota)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTemplateQuota) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateQuotaList) DeepCopyInto(out *ClusterTemplateQuotaList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterTemplateQuota, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateQuotaList.
func (in *ClusterTemplateQuotaList) DeepCopy() *ClusterTemplateQuotaList {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateQuotaList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTemplateQuotaList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateQuotaSpec) DeepCopyInto(out *ClusterTemplateQuotaSpec) {
	*out = *in
	if in.AllowedTemplates != nil {
		in, out := &in.AllowedTemplates, &out.AllowedTemplates
		*out = make([]AllowedTemplate, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateQuotaSpec.
func (in *ClusterTemplateQuotaSpec) DeepCopy() *ClusterTemplateQuotaSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateQuotaSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateQuotaStatus) DeepCopyInto(out *ClusterTemplateQuotaStatus) {
	*out = *in
	if in.TemplateInstances != nil {
		in, out := &in.TemplateInstances, &out.TemplateInstances
		*out = make([]AllowedTemplate, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateQuotaStatus.
func (in *ClusterTemplateQuotaStatus) DeepCopy() *ClusterTemplateQuotaStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateQuotaStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateSpec) DeepCopyInto(out *ClusterTemplateSpec) {
	*out = *in
	in.ClusterDefinition.DeepCopyInto(&out.ClusterDefinition)
	if in.ClusterSetup != nil {
		in, out := &in.ClusterSetup, &out.ClusterSetup
		*out = make([]ClusterSetup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateSpec.
func (in *ClusterTemplateSpec) DeepCopy() *ClusterTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTemplateStatus) DeepCopyInto(out *ClusterTemplateStatus) {
	*out = *in
	out.ClusterDefinition = in.ClusterDefinition
	if in.ClusterSetup != nil {
		in, out := &in.ClusterSetup, &out.ClusterSetup
		*out = make([]ClusterSetupSchema, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTemplateStatus.
func (in *ClusterTemplateStatus) DeepCopy() *ClusterTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Parameter) DeepCopyInto(out *Parameter) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Parameter.
func (in *Parameter) DeepCopy() *Parameter {
	if in == nil {
		return nil
	}
	out := new(Parameter)
	in.DeepCopyInto(out)
	return out
}
