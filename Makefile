# Project config
APP_NAME := wardk8s
IMG ?= $(APP_NAME):latest
GOFLAGS ?= -v

# Build
.PHONY: build
build: ## Build the controller binary
	go build $(GOFLAGS) -o bin/$(APP_NAME) ./cmd/

# Test
.PHONY: test
test: ## Run all unit tests
	go test ./... -v -count=1

.PHONY: bench
bench: ## Run benchmarks (proves webhook latency is in microseconds)
	go test ./internal/webhook/ -bench=. -benchmem -run=^$$ -count=3

.PHONY: lint
lint: ## Run go vet and static analysis
	go vet ./...

.PHONY: coverage
coverage: ## Run tests with coverage report
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@echo "---"
	@echo "HTML report: go tool cover -html=coverage.out"

# Docker
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	docker push $(IMG)

# Deploy
.PHONY: deploy
deploy: ## Deploy to current Kubernetes cluster
	kubectl apply -f config/crd/
	kubectl apply -f config/rbac/
	kubectl apply -f config/webhook/
	go run hack/certs.go --patch-only
	kubectl apply -f config/deploy/

.PHONY: undeploy
undeploy: ## Remove all resources from cluster
	kubectl delete -f config/deploy/ --ignore-not-found
	kubectl delete -f config/webhook/ --ignore-not-found
	kubectl delete -f config/rbac/ --ignore-not-found
	kubectl delete -f config/crd/ --ignore-not-found

# Certs (for webhook TLS)
.PHONY: certs
certs: ## Generate self-signed TLS certificates for webhook
	go run hack/certs.go

# Clean
.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin/ coverage.out

# Help
.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
