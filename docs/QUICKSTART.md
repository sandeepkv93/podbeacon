# PodBeacon Quickstart

PodBeacon automatically injects an OpenTelemetry Collector sidecar into your Kubernetes Pods based on the presence of a specific opt-in annotation.

## Prerequisites
- Kubernetes cluster (v1.31+)
- `cert-manager` installed and running

## Installation

1. Install `cert-manager` (if not already installed):
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.21.1/cert-manager.yaml
   ```

2. Deploy PodBeacon:
   ```bash
   kubectl apply -k github.com/sandeepkv93/podbeacon/config/default
   ```

## Usage

1. Create a `TelemetryProfile` to configure where the injected collector should send data:
   ```yaml
   apiVersion: telemetry.podbeacon.io/v1alpha1
   kind: TelemetryProfile
   metadata:
     name: default
     namespace: default
   spec:
     signals: ["traces", "metrics", "logs"]
     exporter:
       endpoint: "otel-collector.monitoring.svc.cluster.local:4317"
   ```
   Save this as `profile.yaml` and apply:
   ```bash
   kubectl apply -f profile.yaml
   ```

2. Opt-in a Pod (or Deployment) by adding the `telemetry: "enable"` annotation:
   ```yaml
   apiVersion: v1
   kind: Pod
   metadata:
     name: my-app
     annotations:
       telemetry: "enable"
   spec:
     containers:
     - name: app
       image: busybox:latest
       command: ["sleep", "3600"]
   ```

3. Verify the injection:
   ```bash
   kubectl get pod my-app -o jsonpath='{.spec.initContainers[*].name}'
   ```
   You should see `podbeacon-collector` listed, indicating the native sidecar was successfully injected.
