package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	wardv1 "github.com/AxellGS/WardK8s/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityPolicyReconciler manages SecurityPolicy lifecycle and status.
// Enforcement is handled by the webhook, not here.
type SecurityPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ward.io,resources=securitypolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ward.io,resources=securitypolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ward.io,resources=securitypolicies/finalizers,verbs=update

// Reconcile updates the status of SecurityPolicy resources.
func (r *SecurityPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var policy wardv1.SecurityPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling SecurityPolicy",
		"name", policy.Name,
		"mode", policy.Spec.Mode,
		"defaultAction", policy.Spec.DefaultAction,
		"ruleCount", len(policy.Spec.Rules),
	)

	now := metav1.NewTime(time.Now())
	policy.Status.LastEvaluated = &now
	readyCondition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             "PolicyActive",
		Message:            fmt.Sprintf("Policy is active in %s mode with %d rules", policy.Spec.Mode, len(policy.Spec.Rules)),
	}

	updated := false
	for i, c := range policy.Status.Conditions {
		if c.Type == "Ready" {
			policy.Status.Conditions[i] = readyCondition
			updated = true
			break
		}
	}
	if !updated {
		policy.Status.Conditions = append(policy.Status.Conditions, readyCondition)
	}

	if err := r.Status().Update(ctx, &policy); err != nil {
		logger.Error(err, "Failed to update SecurityPolicy status")
		return ctrl.Result{}, err
	}

	logger.Info("SecurityPolicy reconciled successfully",
		"name", policy.Name,
		"mode", policy.Spec.Mode,
	)

	return ctrl.Result{}, nil
}

// SetupWithManager registers the controller with the manager.
func (r *SecurityPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wardv1.SecurityPolicy{}).
		Complete(r)
}
