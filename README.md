# PodBeacon

PodBeacon is a Kubernetes operator that automatically provisions and configures OpenTelemetry collector sidecars for pods based on explicit opt-in annotations. It leverages Kubernetes 1.28+ native sidecar containers (`initContainers` with `restartPolicy: Always`) to ensure telemetry pipelines are running before applications start and don't block Job completions.

## How it works

When a Pod is created with the annotation:
```yaml
metadata:
  annotations:
    telemetry: "enable"
```

The PodBeacon Mutating Webhook intercepts the request, reads a `TelemetryProfile` Custom Resource in the same namespace, and dynamically renders and mounts an OpenTelemetry Collector configuration.

## Documentation

- [Quickstart](docs/QUICKSTART.md) - Get up and running in minutes.
- [Architecture](docs/ARCHITECTURE.md) - Deep dive into controller and webhook design.
- [TelemetryProfile Reference](docs/PROFILE_REFERENCE.md) - CRD API reference.
- [Security](docs/SECURITY.md) - Security posture and RBAC details.
- [Runbooks](docs/RUNBOOKS.md) - Upgrades, uninstallation, and troubleshooting.
- [Benchmarks](docs/BENCHMARKS.md) - Webhook latency measurements.

## Development

PodBeacon is built using `controller-runtime` and Kubebuilder.

**Prerequisites:**
- Go 1.22+
- `make`
- Docker
- `kind` (for E2E tests)

```sh
# Run unit and integration tests (uses envtest)
make test

# Run end-to-end tests (spins up a kind cluster)
make test-e2e

# Build and push the controller image
make docker-build docker-push IMG=<your-registry>/podbeacon:latest

# Deploy to cluster
make deploy IMG=<your-registry>/podbeacon:latest
```
