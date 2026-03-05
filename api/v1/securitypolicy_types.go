package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PolicyAction defines Allow or Deny.
type PolicyAction string

const (
	ActionAllow PolicyAction = "Allow"
	ActionDeny  PolicyAction = "Deny"
)

type PolicyMode string

const (
	ModeEnforce PolicyMode = "Enforce"
	ModeDryRun  PolicyMode = "DryRun"
)

// SecurityPolicySpec defines the desired policy configuration.
type SecurityPolicySpec struct {
	Mode PolicyMode `json:"mode,omitempty"`

	DefaultAction PolicyAction `json:"defaultAction"`

	Rules []PolicyRule `json:"rules,omitempty"`
}

// PolicyRule is a single rule within a SecurityPolicy.
type PolicyRule struct {
	Name   string       `json:"name"`
	Match  PodMatcher   `json:"match"`
	Action PolicyAction `json:"action"`
	Reason string       `json:"reason,omitempty"`
}

// PodMatcher defines criteria for matching pods.
type PodMatcher struct {
	Namespaces  []string          `json:"namespaces,omitempty"`
	PodLabels   map[string]string `json:"podLabels,omitempty"`
	Privileged  *bool             `json:"privileged,omitempty"`
	HostNetwork *bool             `json:"hostNetwork,omitempty"`
	RunAsRoot   *bool             `json:"runAsRoot,omitempty"`
}

// SecurityPolicyStatus tracks evaluation stats.
type SecurityPolicyStatus struct {
	EvaluatedTotal int64              `json:"evaluatedTotal,omitempty"`
	DeniedTotal    int64              `json:"deniedTotal,omitempty"`
	AllowedTotal   int64              `json:"allowedTotal,omitempty"`
	LastEvaluated  *metav1.Time       `json:"lastEvaluated,omitempty"`
	Conditions     []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
// +kubebuilder:printcolumn:name="Default",type=string,JSONPath=`.spec.defaultAction`
// +kubebuilder:printcolumn:name="Evaluated",type=integer,JSONPath=`.status.evaluatedTotal`
// +kubebuilder:printcolumn:name="Denied",type=integer,JSONPath=`.status.deniedTotal`

// SecurityPolicy enforces pod security via admission webhooks.
// Default Deny: pods are blocked unless explicitly allowed.
type SecurityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityPolicySpec   `json:"spec,omitempty"`
	Status SecurityPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityPolicyList contains a list of SecurityPolicy.
type SecurityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityPolicy `json:"items"`
}
