<div align="center">
  <h1>🔭 PodBeacon</h1>
  <p><b>Lightweight, platform-controlled OpenTelemetry sidecar injection for Kubernetes.</b></p>
  <p>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
    <a href="https://kubernetes.io/"><img src="https://img.shields.io/badge/Kubernetes-1.28+-326ce5?style=flat-square&logo=kubernetes" alt="Kubernetes Version"></a>
    <a href="https://github.com/sandeepkv93/podbeacon/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square" alt="License"></a>
  </p>
</div>

---

**PodBeacon** is a specialized Kubernetes operator designed to provision and configure OpenTelemetry Collector sidecars automatically. By shifting the complexity of telemetry pipelines to the cluster platform team, application developers simply add a single annotation to their Pods and instantly receive standard, secure observability.

Unlike the upstream OpenTelemetry Operator which exposes massive configuration complexity, PodBeacon intentionally constrains its API to ensure safety, idempotency, and strict isolation.

## ✨ Key Features

- **Kubernetes Native Sidecars:** Leverages Kubernetes 1.28+ native sidecar containers (`initContainers` with `restartPolicy: Always`). Telemetry is guaranteed to start before the app, gracefully handles container crashes, and *never blocks Job completions*.
- **Developer Simplicity:** Zero configuration required for developers. Add `telemetry: "enable"` to a Pod, and PodBeacon handles the rest.
- **Strict Blast-Radius Limits:** The Mutating Webhook only intercepts explicitly annotated workloads. An unavailable webhook will never block the scheduling of unrelated system or application Pods.
- **Restricted Pod Security:** The injected sidecars comply with the strict Kubernetes `Restricted` Pod Security Standard (non-root, dropped capabilities, `RuntimeDefault` seccomp).
- **Immutable Provenance:** Collector configurations are generated dynamically from CRDs and bound immutably via ConfigMaps, ensuring safe rolling updates.

## 🚀 Quick Overview

When you apply a simple `TelemetryProfile` to your namespace and annotate a Pod:

```yaml
apiVersion: telemetry.podbeacon.io/v1alpha1
kind: TelemetryProfile
metadata:
  name: default
spec:
  exporter:
    endpoint: "otel-collector.monitoring.svc.cluster.local:4317"
```

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    telemetry: "enable"
spec:
  containers:
  - name: my-app
    image: my-app:latest
```

**PodBeacon automatically:**
1. Renders a structurally valid OpenTelemetry configuration.
2. Generates an immutable ConfigMap containing the pipeline.
3. Injects the `podbeacon-collector` native sidecar into the Pod.
4. Plumbs the Downward API, liveness probes, and resource bounds.

## 📚 Documentation

Dive deeper into PodBeacon's design and operation:

- 🏎️ **[Quickstart Guide](docs/QUICKSTART.md):** Get up and running in your cluster in under 5 minutes.
- 📐 **[Architecture](docs/ARCHITECTURE.md):** A detailed breakdown of the Controller and Webhook designs.
- 📖 **[Profile Reference](docs/PROFILE_REFERENCE.md):** Complete API documentation for the `TelemetryProfile` CRD.
- 🔒 **[Security Posture](docs/SECURITY.md):** Details on RBAC limits, PSS compliance, and injection isolation.
- 📊 **[Benchmarks](docs/BENCHMARKS.md):** Latency metrics and P99 guarantees for high-throughput scheduling.
- 🛠️ **[Runbooks](docs/RUNBOOKS.md):** Operator guides for upgrades, uninstallation, and troubleshooting.

## 🛠️ Development & Contributing

PodBeacon is built using [Kubebuilder](https://book.kubebuilder.io/) and the `controller-runtime` framework.

### Prerequisites
- Go 1.22+
- `make`
- Docker
- [kind](https://kind.sigs.k8s.io/) (for End-to-End tests)

### Local Development

The Makefile provides standard tooling to run, test, and deploy the operator locally.

```sh
# Run unit and integration tests (uses envtest)
make test

# Run isolated End-to-End tests (automatically spins up a dedicated Kind cluster)
make test-e2e

# Build the controller manager container image
make docker-build IMG=ghcr.io/your-org/podbeacon:latest

# Deploy the operator to your current active Kubernetes cluster
make deploy IMG=ghcr.io/your-org/podbeacon:latest
```

## 📄 License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
