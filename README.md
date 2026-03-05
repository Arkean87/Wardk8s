# WardK8s

A Kubernetes Security Policy Controller that enforces pod security through Validating Admission Webhooks, implementing a **Default Deny** model.

## What It Does

WardK8s intercepts pod creation requests via a Validating Admission Webhook and evaluates them against `SecurityPolicy` custom resources. If no rule explicitly allows a pod, it is **denied by default** — the same model used by firewalls and Linux Security Modules.

```
Pod Create Request → API Server → WardK8s Webhook → Allow/Deny
                                        ↓
                                  SecurityPolicy CRD
                                  (Default Deny + Rules)
```

## Key Features

- **Default Deny Architecture** — Pods are blocked unless explicitly allowed
- **Dry-Run Mode** — Log violations without enforcing, for safe rollout
- **Rule Evaluation** — iptables-style top-to-bottom evaluation, first match wins
- **Pod Matchers** — Namespace, labels, privileged, hostNetwork, runAsRoot
- **Prometheus Metrics** — `pods_denied_total`, `pods_allowed_total`, `policy_evaluations_total`
- **Sub-microsecond Latency** — 297ns per evaluation (see benchmarks below)

## Prerequisites

To build, test, and deploy this controller locally, ensure the following are installed:
- **Go 1.22+** (For building and testing)
- **Make** (For executing `Makefile` targets)
- **Docker** & **Kind** (or Minikube) (For the local cluster)
- **kubectl** (For interacting with the cluster)

## Quick Start

Execute the following commands from your terminal (cross-platform compatible):

```bash
# Build the controller binary
go build -v -o bin/wardk8s ./cmd/

# Run unit tests
go test ./... -v -count=1

# Run benchmarks (proves nanosecond latency)
go test ./internal/webhook/ -bench=. -benchmem -run=^$ -count=3

# Generate TLS certs locally
go run hack/certs.go
```

## SecurityPolicy Example

```yaml
apiVersion: ward.io/v1
kind: SecurityPolicy
metadata:
  name: production-policy
spec:
  mode: Enforce       # or DryRun
  defaultAction: Deny
  rules:
    - name: allow-trusted
      match:
        namespaces: ["production"]
        podLabels:
          security-tier: "trusted"
      action: Allow
    - name: deny-privileged
      match:
        privileged: true
      action: Deny
      reason: "Privileged containers are not allowed"
```

## Benchmarks

Policy evaluation algorithm is optimized for high-throughput, resulting in minimal latency overhead to the API server:

```
BenchmarkEvaluatePolicy_SimplePolicy-12    4102887    297.4 ns/op    48 B/op    2 allocs/op
BenchmarkEvaluatePolicy_10Rules-12         1654720    741.4 ns/op    48 B/op    2 allocs/op
BenchmarkEvaluatePolicy_50Rules-12          431432   2704   ns/op    64 B/op    2 allocs/op
```

Even with 50 rules, evaluation takes < 3µs with only 2 allocations. 

> 💡 **Environment Note:** Benchmarks executed on local developer hardware (12 logical cores). Exact nanosecond metrics will naturally vary depending on the host architecture and deployment environment, but the algorithmic complexity ensures O(1) allocation overhead.

## Production Deployment

Tested on a real Kubernetes cluster (Kind v0.27.0, K8s v1.32.2):

```bash
# Create cluster
kind create cluster --name wardk8s

# Build and load image
docker build -t wardk8s:latest .
kind load docker-image wardk8s:latest --name wardk8s

# Generate TLS certs locally
go run hack/certs.go

# Create namespace so the secret can be injected
kubectl create namespace wardk8s-system

# Create secret for TLS
kubectl create secret tls wardk8s-webhook-certs \
  --cert=config/webhook/certs/tls.crt \
  --key=config/webhook/certs/tls.key \
  -n wardk8s-system

# Deploy CRDs, RBAC, Webhook config and the Controller
kubectl apply -f config/crd/
kubectl apply -f config/rbac/
kubectl apply -f config/webhook/
go run hack/certs.go --patch-only
kubectl apply -f config/deploy/
```

```
$ kubectl get pods -n wardk8s-system
NAME                                  READY   STATUS    RESTARTS   AGE
wardk8s-controller-7c97fdc478-5sd2n   1/1     Running   0          3m

$ kubectl get all -n wardk8s-system
pod/wardk8s-controller-7c97fdc478-5sd2n   1/1     Running
service/wardk8s-webhook                   ClusterIP   10.96.124.113   443/TCP
deployment.apps/wardk8s-controller        1/1     1            1
```

### Resource Profiling (Baseline)

Measured with `kubectl top pods` via metrics-server at idle (no incoming admission requests):

| Metric | Baseline (Idle) | Notes |
|---|---|---|
| **CPU** | <2m (millicores) | Event-driven — no polling loops |
| **Memory** | 8Mi | Multi-stage Alpine build, static Go binary |
| **Cold Start** | <100ms | Ready to serve before readiness probe fires |
| **Resource Requests** | 50m CPU / 64Mi RAM | Configured in deployment |
| **Resource Limits** | 200m CPU / 128Mi RAM | Hard ceiling prevents noisy-neighbor |

**Why this matters:** In large-scale clusters, control plane efficiency is crucial. A webhook that consumes 8Mi minimizes footprint overhead for infrastructure administrators. For architectural comparison:

| Stack | Typical Idle Memory | Profile |
|---|---|---|
| JVM-based webhook | ~300Mi | JIT compiled, garbage collected |
| Python/FastAPI webhook | ~60Mi | Interpreted |
| **WardK8s (Go)** | **8Mi** | **AOT compiled, statically linked** |

Combined with the sub-microsecond algorithmic evaluation, WardK8s is designed to respect the Kubernetes control plane's resource constraints.

## Architecture

Built with [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) (the same foundation as Kubebuilder), without scaffolding generators, for maximum control over the architecture.

```
cmd/main.go                              # Entry point, wires manager
api/v1/
  securitypolicy_types.go                # CRD type definitions
  groupversion_info.go                   # Scheme registration
  zz_generated.deepcopy.go              # runtime.Object implementation
internal/
  controller/
    securitypolicy_controller.go         # Reconciler (status management)
  webhook/
    pod_validator.go                     # Admission webhook (enforcement)
    pod_validator_test.go                # 7 unit tests + 3 benchmarks
  metrics/
    prometheus.go                        # Metrics exporter
```

**Design Decisions:**

| Decision | Rationale |
|---|---|
| Validating (not Mutating) webhook | Pods should be denied, not silently modified |
| Default Deny | Same model as iptables/LSM: explicit rules, then default policy |
| Controller handles status, webhook handles enforcement | Separation of concerns |
| DryRun mode | Production-safe rollout — test before enforce |
| No Kubebuilder scaffolding | Clean codebase, demonstrates understanding of internals |

## Testing

```bash
# Unit tests (7 tests covering all security matchers)
go test ./... -v -count=1

# Benchmarks (latency proof)
go test ./internal/webhook/ -bench=. -benchmem -run=^$ -count=3

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Tech Stack

- **Go 1.26** with controller-runtime v0.23.1
- **Kubernetes** API machinery v0.35.2
- **Prometheus** client_golang for metrics
- **Zap** structured JSON logging

## License

Apache 2.0
