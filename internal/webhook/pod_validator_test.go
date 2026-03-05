package webhook

import (
	"testing"

	wardv1 "github.com/AxellGS/WardK8s/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// boolPtr is a helper to get a pointer to a bool value.
func boolPtr(b bool) *bool { return &b }

// int64Ptr is a helper to get a pointer to an int64 value.
func int64Ptr(i int64) *int64 { return &i }

// TestEvaluatePolicy_DefaultDeny verifies the core security model:
// if no rule matches, the default action (Deny) is applied.
func TestEvaluatePolicy_DefaultDeny(t *testing.T) {
	policy := &wardv1.SecurityPolicy{
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionDeny,
			Rules:         []wardv1.PolicyRule{},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "untrusted-pod",
			Namespace: "default",
		},
	}

	result := evaluatePolicy(policy, pod)

	if result.Action != wardv1.ActionDeny {
		t.Errorf("expected Deny, got %s", result.Action)
	}
	if result.MatchedRule != "<default>" {
		t.Errorf("expected <default> rule, got %s", result.MatchedRule)
	}
}

// TestEvaluatePolicy_ExplicitAllow verifies that an explicit Allow rule
// overrides the Default Deny when the pod matches.
func TestEvaluatePolicy_ExplicitAllow(t *testing.T) {
	policy := &wardv1.SecurityPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "production"},
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionDeny,
			Rules: []wardv1.PolicyRule{
				{
					Name: "allow-trusted",
					Match: wardv1.PodMatcher{
						PodLabels: map[string]string{"security-tier": "trusted"},
					},
					Action: wardv1.ActionAllow,
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "trusted-pod",
			Namespace: "production",
			Labels:    map[string]string{"security-tier": "trusted"},
		},
	}

	result := evaluatePolicy(policy, pod)

	if result.Action != wardv1.ActionAllow {
		t.Errorf("expected Allow, got %s", result.Action)
	}
	if result.MatchedRule != "allow-trusted" {
		t.Errorf("expected rule 'allow-trusted', got %s", result.MatchedRule)
	}
}

// TestEvaluatePolicy_DenyPrivileged verifies that privileged containers
// are correctly detected and denied.
func TestEvaluatePolicy_DenyPrivileged(t *testing.T) {
	policy := &wardv1.SecurityPolicy{
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionAllow,
			Rules: []wardv1.PolicyRule{
				{
					Name: "deny-privileged",
					Match: wardv1.PodMatcher{
						Privileged: boolPtr(true),
					},
					Action: wardv1.ActionDeny,
					Reason: "Privileged containers are not allowed",
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "evil-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "evil",
					Image: "nginx",
					SecurityContext: &corev1.SecurityContext{
						Privileged: boolPtr(true),
					},
				},
			},
		},
	}

	result := evaluatePolicy(policy, pod)

	if result.Action != wardv1.ActionDeny {
		t.Errorf("expected Deny for privileged pod, got %s", result.Action)
	}
}

// TestEvaluatePolicy_DenyRoot verifies that pods running as root (UID 0)
// are correctly detected and denied.
func TestEvaluatePolicy_DenyRoot(t *testing.T) {
	policy := &wardv1.SecurityPolicy{
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionAllow,
			Rules: []wardv1.PolicyRule{
				{
					Name:   "deny-root",
					Match:  wardv1.PodMatcher{RunAsRoot: boolPtr(true)},
					Action: wardv1.ActionDeny,
					Reason: "Running as root is prohibited",
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "root-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: int64Ptr(0),
			},
			Containers: []corev1.Container{{Name: "app", Image: "nginx"}},
		},
	}

	result := evaluatePolicy(policy, pod)

	if result.Action != wardv1.ActionDeny {
		t.Errorf("expected Deny for root pod, got %s", result.Action)
	}
}

// TestEvaluatePolicy_DenyHostNetwork verifies that pods requesting
// host network access are correctly denied.
func TestEvaluatePolicy_DenyHostNetwork(t *testing.T) {
	policy := &wardv1.SecurityPolicy{
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionAllow,
			Rules: []wardv1.PolicyRule{
				{
					Name:   "deny-hostnet",
					Match:  wardv1.PodMatcher{HostNetwork: boolPtr(true)},
					Action: wardv1.ActionDeny,
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "hostnet-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			HostNetwork: true,
			Containers:  []corev1.Container{{Name: "app", Image: "nginx"}},
		},
	}

	result := evaluatePolicy(policy, pod)

	if result.Action != wardv1.ActionDeny {
		t.Errorf("expected Deny for host network pod, got %s", result.Action)
	}
}

// TestEvaluatePolicy_FirstMatchWins verifies that rule evaluation stops
// at the first match (iptables-like behavior).
func TestEvaluatePolicy_FirstMatchWins(t *testing.T) {
	policy := &wardv1.SecurityPolicy{
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionDeny,
			Rules: []wardv1.PolicyRule{
				{
					Name:   "allow-nginx",
					Match:  wardv1.PodMatcher{PodLabels: map[string]string{"app": "nginx"}},
					Action: wardv1.ActionAllow,
				},
				{
					Name:   "deny-all",
					Match:  wardv1.PodMatcher{PodLabels: map[string]string{"app": "nginx"}},
					Action: wardv1.ActionDeny,
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "nginx-pod",
			Labels: map[string]string{"app": "nginx"},
		},
	}

	result := evaluatePolicy(policy, pod)

	if result.Action != wardv1.ActionAllow {
		t.Errorf("expected first rule (Allow) to win, got %s", result.Action)
	}
	if result.MatchedRule != "allow-nginx" {
		t.Errorf("expected 'allow-nginx', got %s", result.MatchedRule)
	}
}

// TestEvaluatePolicy_NamespaceFilter verifies that namespace-scoped rules
// only match pods in the specified namespaces.
func TestEvaluatePolicy_NamespaceFilter(t *testing.T) {
	policy := &wardv1.SecurityPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "production"},
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionDeny,
			Rules: []wardv1.PolicyRule{
				{
					Name: "allow-prod-only",
					Match: wardv1.PodMatcher{
						Namespaces: []string{"production"},
					},
					Action: wardv1.ActionAllow,
				},
			},
		},
	}

	// Pod in production should match
	prodPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-pod", Namespace: "production"},
	}
	result := evaluatePolicy(policy, prodPod)
	if result.Action != wardv1.ActionAllow {
		t.Errorf("expected Allow for production pod, got %s", result.Action)
	}

	// Pod in staging should NOT match, fall to default Deny
	stagingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "staging-pod", Namespace: "staging"},
	}
	result = evaluatePolicy(policy, stagingPod)
	if result.Action != wardv1.ActionDeny {
		t.Errorf("expected Deny for staging pod, got %s", result.Action)
	}
}

// BenchmarkEvaluatePolicy_SimplePolicy benchmarks a single-rule policy.
func BenchmarkEvaluatePolicy_SimplePolicy(b *testing.B) {
	policy := &wardv1.SecurityPolicy{
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionDeny,
			Rules: []wardv1.PolicyRule{
				{
					Name:   "allow-trusted",
					Match:  wardv1.PodMatcher{PodLabels: map[string]string{"tier": "trusted"}},
					Action: wardv1.ActionAllow,
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"tier": "trusted"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		evaluatePolicy(policy, pod)
	}
}

// BenchmarkEvaluatePolicy_10Rules benchmarks with 10 rules (realistic policy).
func BenchmarkEvaluatePolicy_10Rules(b *testing.B) {
	rules := make([]wardv1.PolicyRule, 10)
	for i := 0; i < 9; i++ {
		rules[i] = wardv1.PolicyRule{
			Name:   "no-match",
			Match:  wardv1.PodMatcher{PodLabels: map[string]string{"nope": "nope"}},
			Action: wardv1.ActionDeny,
		}
	}
	rules[9] = wardv1.PolicyRule{
		Name:   "last-match",
		Match:  wardv1.PodMatcher{PodLabels: map[string]string{"app": "target"}},
		Action: wardv1.ActionAllow,
	}

	policy := &wardv1.SecurityPolicy{
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionDeny,
			Rules:         rules,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"app": "target"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		evaluatePolicy(policy, pod)
	}
}

// BenchmarkEvaluatePolicy_50Rules benchmarks worst-case with 50 rules.
func BenchmarkEvaluatePolicy_50Rules(b *testing.B) {
	rules := make([]wardv1.PolicyRule, 50)
	for i := range rules {
		rules[i] = wardv1.PolicyRule{
			Name:   "no-match",
			Match:  wardv1.PodMatcher{PodLabels: map[string]string{"miss": "miss"}},
			Action: wardv1.ActionDeny,
		}
	}

	policy := &wardv1.SecurityPolicy{
		Spec: wardv1.SecurityPolicySpec{
			DefaultAction: wardv1.ActionDeny,
			Rules:         rules,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"app": "something-else"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		evaluatePolicy(policy, pod)
	}
}
