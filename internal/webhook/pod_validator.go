package webhook

import (
	"context"
	"fmt"
	"net/http"

	wardv1 "github.com/AxellGS/WardK8s/api/v1"
	"github.com/AxellGS/WardK8s/internal/metrics"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// PodValidator intercepts pod creation and evaluates SecurityPolicy rules.
// Validating (not Mutating)  Ewe deny or allow, never silently modify.
type PodValidator struct {
	Client  client.Client
	Decoder admission.Decoder
}

// Handle evaluates incoming pods against all SecurityPolicies in the namespace.
func (v *PodValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithValues(
		"pod", req.Name,
		"namespace", req.Namespace,
		"operation", req.Operation,
	)

	pod := &corev1.Pod{}
	if err := v.Decoder.Decode(req, pod); err != nil {
		logger.Error(err, "Failed to decode pod")
		return admission.Errored(http.StatusBadRequest, err)
	}

	var policies wardv1.SecurityPolicyList
	if err := v.Client.List(ctx, &policies, client.InNamespace(req.Namespace)); err != nil {
		logger.Error(err, "Failed to list SecurityPolicies")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if len(policies.Items) == 0 {
		logger.V(1).Info("No SecurityPolicies found, allowing pod")
		return admission.Allowed("no SecurityPolicies defined")
	}

	for _, policy := range policies.Items {
		result := evaluatePolicy(&policy, pod)
		metrics.RecordEvaluation(policy.Name, result.Action)

		if result.Action == wardv1.ActionDeny {
			if policy.Spec.Mode == wardv1.ModeDryRun {
				logger.Info("DryRun: pod would be denied",
					"policy", policy.Name,
					"rule", result.MatchedRule,
					"reason", result.Reason,
				)
				continue
			}

			logger.Info("Pod denied by SecurityPolicy",
				"policy", policy.Name,
				"rule", result.MatchedRule,
				"reason", result.Reason,
			)
			return admission.Denied(fmt.Sprintf(
				"denied by SecurityPolicy %q (rule: %s): %s",
				policy.Name, result.MatchedRule, result.Reason,
			))
		}
	}

	logger.V(1).Info("Pod allowed by all SecurityPolicies")
	return admission.Allowed("all policies passed")
}

type EvaluationResult struct {
	Action      wardv1.PolicyAction
	MatchedRule string
	Reason      string
}

// evaluatePolicy checks rules top-to-bottom (iptables-style). First match wins.
func evaluatePolicy(policy *wardv1.SecurityPolicy, pod *corev1.Pod) EvaluationResult {
	for _, rule := range policy.Spec.Rules {
		if matchesPod(&rule.Match, pod, policy.Namespace) {
			reason := rule.Reason
			if reason == "" {
				reason = fmt.Sprintf("matched rule %q", rule.Name)
			}
			return EvaluationResult{
				Action:      rule.Action,
				MatchedRule: rule.Name,
				Reason:      reason,
			}
		}
	}

	return EvaluationResult{
		Action:      policy.Spec.DefaultAction,
		MatchedRule: "<default>",
		Reason:      fmt.Sprintf("no rule matched, default action is %s", policy.Spec.DefaultAction),
	}
}

// matchesPod returns true if all specified matcher fields match (AND logic).
func matchesPod(matcher *wardv1.PodMatcher, pod *corev1.Pod, policyNamespace string) bool {
	if len(matcher.Namespaces) > 0 {
		nsMatch := false
		podNs := pod.Namespace
		if podNs == "" {
			podNs = policyNamespace // fallback for pods without explicit namespace
		}
		for _, ns := range matcher.Namespaces {
			if ns == podNs {
				nsMatch = true
				break
			}
		}
		if !nsMatch {
			return false
		}
	}

	if len(matcher.PodLabels) > 0 {
		for key, value := range matcher.PodLabels {
			if podValue, ok := pod.Labels[key]; !ok || podValue != value {
				return false
			}
		}
	}

	if matcher.Privileged != nil && *matcher.Privileged {
		if !hasPrivilegedContainer(pod) {
			return false
		}
	}

	if matcher.HostNetwork != nil && *matcher.HostNetwork {
		if !pod.Spec.HostNetwork {
			return false
		}
	}

	if matcher.RunAsRoot != nil && *matcher.RunAsRoot {
		if !isRunAsRoot(pod) {
			return false
		}
	}

	return true
}

func hasPrivilegedContainer(pod *corev1.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.SecurityContext != nil && c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
			return true
		}
	}
	for _, c := range pod.Spec.InitContainers {
		if c.SecurityContext != nil && c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
			return true
		}
	}
	return false
}

func isRunAsRoot(pod *corev1.Pod) bool {
	if pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.RunAsUser != nil {
		if *pod.Spec.SecurityContext.RunAsUser == 0 {
			return true
		}
	}
	for _, c := range pod.Spec.Containers {
		if c.SecurityContext != nil && c.SecurityContext.RunAsUser != nil {
			if *c.SecurityContext.RunAsUser == 0 {
				return true
			}
		}
	}
	return false
}
