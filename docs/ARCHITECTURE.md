# Architecture

## Overview

WardK8s is a Kubernetes Security Policy Controller that enforces pod security via Validating Admission Webhooks. It implements a Default Deny model: pods are blocked unless explicitly allowed by a SecurityPolicy rule.

## Data Flow

```
1. User runs: kubectl apply -f pod.yaml
2. API Server receives the request
3. API Server calls WardK8s webhook (HTTPS, port 9443)
4. Webhook fetches SecurityPolicy CRDs in the pod's namespace
5. Rules are evaluated top-to-bottom (first match wins)
6. If no rule matches → Default Action applies (Deny)
7. If Mode=DryRun → log violation, allow pod
8. If Mode=Enforce → reject pod with reason
9. Prometheus metrics are updated
```

## Security Model: Default Deny

The Default Deny model is borrowed from firewall design (iptables) and Linux Security Modules:

**Everything is denied unless explicitly allowed.**

This inverts the typical Kubernetes behavior where everything is allowed unless explicitly denied. The security advantage is that new, unknown pod configurations are automatically blocked until a security team reviews and creates a rule for them.

### Rule Evaluation (iptables-style)

Rules are evaluated in the order they appear in the SecurityPolicy spec. The first matching rule determines the outcome. If no rule matches, the `defaultAction` is applied.

This is identical to how iptables chains work:
1. Check rule 1 → no match → continue
2. Check rule 2 → no match → continue
3. Check rule N → match → apply action (Allow/Deny)
4. No rules matched → apply default policy

### Dry-Run Mode

DryRun mode allows security teams to deploy policies in "audit mode":
- Violations are logged with full context (policy name, rule name, pod details)
- Prometheus metrics are updated (so dashboards show what WOULD be blocked)
- But the pod is allowed to proceed

This is critical for production deployments where you want to validate policy impact before enforcement.

## Component Separation

### Controller (Reconciler)
- Watches SecurityPolicy CRD changes
- Updates status (conditions, timestamps)
- Does NOT enforce policies

### Webhook (PodValidator)
- Intercepts pod creation in real-time
- Evaluates policies against the pod
- Returns Allow or Deny to the API Server
- Records Prometheus metrics

This separation follows the standard Kubernetes operator pattern where controllers manage desired state and webhooks handle admission control.

## Why Not Kubebuilder?

WardK8s uses controller-runtime directly (the same library Kubebuilder generates code from) without the scaffolding tool. This is intentional:

1. **Clean codebase** — No auto-generated boilerplate files
2. **Full understanding** — Every line of code is intentional and documented
3. **Windows development** — Kubebuilder does not support Windows natively
4. **Demonstrates internals knowledge** — Shows understanding of how operators work, not just how to run a generator
