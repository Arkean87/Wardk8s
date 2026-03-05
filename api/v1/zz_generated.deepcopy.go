package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies all properties into another SecurityPolicy instance.
func (in *SecurityPolicy) DeepCopyInto(out *SecurityPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy creates a deep copy of SecurityPolicy.
func (in *SecurityPolicy) DeepCopy() *SecurityPolicy {
	if in == nil {
		return nil
	}
	out := new(SecurityPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *SecurityPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies all properties into another SecurityPolicyList instance.
func (in *SecurityPolicyList) DeepCopyInto(out *SecurityPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SecurityPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a deep copy of SecurityPolicyList.
func (in *SecurityPolicyList) DeepCopy() *SecurityPolicyList {
	if in == nil {
		return nil
	}
	out := new(SecurityPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *SecurityPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies SecurityPolicySpec.
func (in *SecurityPolicySpec) DeepCopyInto(out *SecurityPolicySpec) {
	*out = *in
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]PolicyRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies PolicyRule.
func (in *PolicyRule) DeepCopyInto(out *PolicyRule) {
	*out = *in
	in.Match.DeepCopyInto(&out.Match)
}

// DeepCopyInto copies PodMatcher.
func (in *PodMatcher) DeepCopyInto(out *PodMatcher) {
	*out = *in
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.PodLabels != nil {
		in, out := &in.PodLabels, &out.PodLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Privileged != nil {
		in, out := &in.Privileged, &out.Privileged
		*out = new(bool)
		**out = **in
	}
	if in.HostNetwork != nil {
		in, out := &in.HostNetwork, &out.HostNetwork
		*out = new(bool)
		**out = **in
	}
	if in.RunAsRoot != nil {
		in, out := &in.RunAsRoot, &out.RunAsRoot
		*out = new(bool)
		**out = **in
	}
}

// DeepCopyInto copies SecurityPolicyStatus.
func (in *SecurityPolicyStatus) DeepCopyInto(out *SecurityPolicyStatus) {
	*out = *in
	if in.LastEvaluated != nil {
		in, out := &in.LastEvaluated, &out.LastEvaluated
		*out = (*in).DeepCopy()
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
