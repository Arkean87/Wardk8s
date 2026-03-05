package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	wardv1 "github.com/AxellGS/WardK8s/api/v1"
)

var (
	evaluationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "wardk8s",
			Name:      "policy_evaluations_total",
			Help:      "Total number of pod evaluations against security policies.",
		},
		[]string{"policy", "action"},
	)

	deniedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "wardk8s",
			Name:      "pods_denied_total",
			Help:      "Total number of pods denied by security policies.",
		},
	)

	allowedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "wardk8s",
			Name:      "pods_allowed_total",
			Help:      "Total number of pods allowed by security policies.",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(evaluationsTotal, deniedTotal, allowedTotal)
}

// RecordEvaluation increments the appropriate counters for a policy evaluation.
func RecordEvaluation(policyName string, action wardv1.PolicyAction) {
	evaluationsTotal.WithLabelValues(policyName, string(action)).Inc()

	switch action {
	case wardv1.ActionDeny:
		deniedTotal.Inc()
	case wardv1.ActionAllow:
		allowedTotal.Inc()
	}
}
