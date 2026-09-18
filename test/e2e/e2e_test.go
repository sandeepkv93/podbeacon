//go:build e2e
// +build e2e

/*
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
*/

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sandeepkv93/podbeacon/test/utils"
)

// namespace where the project is deployed in
const namespace = "podbeacon-system"

// serviceAccountName created for the project
const serviceAccountName = "podbeacon-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "podbeacon-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "podbeacon-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", managerImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				By("getting the name of the controller-manager pod")
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				By("validating the pod's status")
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=podbeacon-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			utils.Run(cmd) // ignore error as it may already exist

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("ensuring the controller pod is ready")
			verifyControllerPodReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
					"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"), "Controller pod not ready")
			}
			Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Serving metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

			By("waiting for the webhook service endpoints to be ready")
			verifyWebhookEndpointsReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpointslices.discovery.k8s.io", "-n", namespace,
					"-l", "kubernetes.io/service-name=podbeacon-webhook-service",
					"-o", "jsonpath={range .items[*]}{range .endpoints[*]}{.addresses[*]}{end}{end}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Webhook endpoints should exist")
				g.Expect(output).ShouldNot(BeEmpty(), "Webhook endpoints not yet ready")
			}
			Eventually(verifyWebhookEndpointsReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying the mutating webhook server is ready")
			verifyMutatingWebhookReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "mutatingwebhookconfigurations.admissionregistration.k8s.io",
					"podbeacon-mutating-webhook-configuration",
					"-o", "jsonpath={.webhooks[0].clientConfig.caBundle}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "MutatingWebhookConfiguration should exist")
				g.Expect(output).ShouldNot(BeEmpty(), "Mutating webhook CA bundle not yet injected")
			}
			Eventually(verifyMutatingWebhookReady, 3*time.Minute, time.Second).Should(Succeed())

			By("waiting additional time for webhook server to stabilize")
			time.Sleep(5 * time.Second)

			// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": [
								"for i in $(seq 1 30); do curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics && exit 0 || sleep 2; done; exit 1"
							],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		It("should provisioned cert-manager", func() {
			By("validating that cert-manager has the certificate Secret")
			verifyCertManager := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secrets", "webhook-server-cert", "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(verifyCertManager).Should(Succeed())
		})

		It("should have CA injection for mutating webhooks", func() {
			By("checking CA injection for mutating webhooks")
			verifyCAInjection := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"mutatingwebhookconfigurations.admissionregistration.k8s.io",
					"podbeacon-mutating-webhook-configuration",
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				mwhOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))
			}
			Eventually(verifyCAInjection).Should(Succeed())
		})

		It("should inject the collector sidecar into an opted-in pod", func() {
			By("Creating a TelemetryProfile")
			profileYAML := `
apiVersion: telemetry.podbeacon.io/v1alpha1
kind: TelemetryProfile
metadata:
  name: default
  namespace: default
spec:
  signals: ["traces", "metrics"]
  exporter:
    endpoint: "otel-collector.monitoring.svc.cluster.local:4317"
    tls:
      insecure: true
`
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(profileYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create TelemetryProfile")

			// Wait a few seconds for the controller to process it and generate the configmap
			time.Sleep(3 * time.Second)

			By("Creating a test pod with the telemetry: enable annotation")
			podYAML := `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
  annotations:
    telemetry: "enable"
spec:
  containers:
  - name: my-app
    image: busybox:latest
    command: ["sleep", "3600"]
`
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(podYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create test pod")

			By("Verifying the pod received the podbeacon-collector sidecar")
			verifySidecarInjected := func(g Gomega) {
				cmd = exec.Command("kubectl", "get", "pod", "test-pod", "-n", "default", "-o", "jsonpath={.spec.initContainers[*].name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("podbeacon-collector"), "podbeacon-collector sidecar not found in test-pod initContainers")
				
				cmd = exec.Command("kubectl", "get", "pod", "test-pod", "-n", "default", "-o", "jsonpath={.metadata.annotations['podbeacon\\.io/injected']}")
				output, err = utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"), "podbeacon.io/injected annotation should be true")
			}
			Eventually(verifySidecarInjected, 1*time.Minute).Should(Succeed())

			By("Cleaning up test resources")
			cmd = exec.Command("kubectl", "delete", "pod", "test-pod", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "telemetryprofile", "default", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
		})
		It("should verify end-to-end signal delivery to OTLP receiver (T-1)", func() {
			By("Deploying a mock OTLP receiver")
			receiverYAML := `
apiVersion: v1
kind: Pod
metadata:
  name: otel-receiver
  namespace: default
  labels:
    app: otel-receiver
spec:
  containers:
  - name: otel-collector
    image: otel/opentelemetry-collector:latest
    command: ["/otelcol", "--config=/etc/otelcol/config.yaml"]
    ports:
    - containerPort: 4317
    volumeMounts:
    - name: config-volume
      mountPath: /etc/otelcol
  volumes:
  - name: config-volume
    configMap:
      name: otel-receiver-config
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-receiver-config
  namespace: default
data:
  config.yaml: |
    receivers:
      otlp:
        protocols:
          grpc:
              endpoint: 0.0.0.0:4317
    exporters:
      debug:
        verbosity: detailed
    service:
      pipelines:
        traces:
          receivers: [otlp]
          exporters: [debug]
        metrics:
          receivers: [otlp]
          exporters: [debug]
        logs:
          receivers: [otlp]
          exporters: [debug]
---
apiVersion: v1
kind: Service
metadata:
  name: otel-receiver
  namespace: default
spec:
  selector:
    app: otel-receiver
  ports:
  - port: 4317
    targetPort: 4317
`
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(receiverYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to deploy mock OTLP receiver")

			By("Waiting for receiver to be ready")
			verifyReceiverReady := func(g Gomega) {
				cmd = exec.Command("kubectl", "get", "pod", "otel-receiver", "-n", "default", "-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"))
			}
			Eventually(verifyReceiverReady, 2*time.Minute).Should(Succeed())

			By("Creating a TelemetryProfile for the mock receiver")
			profileYAML := `
apiVersion: telemetry.podbeacon.io/v1alpha1
kind: TelemetryProfile
metadata:
  name: signal-test
  namespace: default
spec:
  signals: ["traces", "metrics", "logs"]
  exporter:
    endpoint: "otel-receiver.default.svc.cluster.local:4317"
    tls:
      insecure: true
`
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(profileYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create TelemetryProfile")

			// Wait for profile processing
			time.Sleep(3 * time.Second)

			By("Deploying a mock telemetry emitter pod")
			emitterYAML := `
apiVersion: v1
kind: Pod
metadata:
  name: mock-emitter
  namespace: default
  annotations:
    telemetry: "enable"
    podbeacon.io/profile: "signal-test"
spec:
  containers:
  - name: emitter
    image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:latest
    args:
    - traces
    - --otlp-endpoint=localhost:4317
    - --otlp-insecure
    - --rate=10
    - --duration=10s
`
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(emitterYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create mock emitter pod")

			By("Verifying signal reception in the receiver logs")
			verifySignalsReceived := func(g Gomega) {
				cmd = exec.Command("kubectl", "logs", "otel-receiver", "-n", "default")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("ResourceSpans"))
			}
			Eventually(verifySignalsReceived, 2*time.Minute).Should(Succeed())

			By("Cleaning up T-1 resources")
			cmd = exec.Command("kubectl", "delete", "pod", "mock-emitter", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "telemetryprofile", "signal-test", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "pod", "otel-receiver", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "service", "otel-receiver", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "configmap", "otel-receiver-config", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
		})

		It("should inject sidecar into various workload types and Job should complete (T-2)", func() {
			By("Creating a TelemetryProfile")
			profileYAML := `
apiVersion: telemetry.podbeacon.io/v1alpha1
kind: TelemetryProfile
metadata:
  name: workload-test
  namespace: default
spec:
  signals: ["traces"]
  exporter:
    endpoint: "otel-collector.monitoring.svc.cluster.local:4317"
    tls:
      insecure: true
`
			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(profileYAML)
			utils.Run(cmd)
			time.Sleep(3 * time.Second)

			By("Deploying a Job with telemetry enabled")
			jobYAML := `
apiVersion: batch/v1
kind: Job
metadata:
  name: test-job
  namespace: default
spec:
  template:
    metadata:
      annotations:
        telemetry: "enable"
        podbeacon.io/profile: "workload-test"
    spec:
      containers:
      - name: task
        image: busybox:latest
        command: ["echo", "job done"]
      restartPolicy: Never
`
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(jobYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create Job")

			By("Verifying Job pod is injected and Job completes successfully")
			verifyJobComplete := func(g Gomega) {
				// Job should eventually complete
				cmd = exec.Command("kubectl", "get", "job", "test-job", "-n", "default", "-o", "jsonpath={.status.conditions[?(@.type=='Complete')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"))
				
				// Verify the pod was injected
				cmd = exec.Command("kubectl", "get", "pods", "-n", "default", "-l", "job-name=test-job", "-o", "jsonpath={.items[0].metadata.annotations['podbeacon\\.io/injected']}")
				output, err = utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyJobComplete, 3*time.Minute).Should(Succeed())

			By("Deploying a Deployment with telemetry enabled")
			deployYAML := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-deploy
  template:
    metadata:
      labels:
        app: test-deploy
      annotations:
        telemetry: "enable"
        podbeacon.io/profile: "workload-test"
    spec:
      containers:
      - name: task
        image: busybox:latest
        command: ["sleep", "3600"]
`
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(deployYAML)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create Deployment")

			By("Verifying Deployment pod is injected")
			verifyDeployInjected := func(g Gomega) {
				cmd = exec.Command("kubectl", "get", "pods", "-n", "default", "-l", "app=test-deploy", "-o", "jsonpath={.items[0].metadata.annotations['podbeacon\\.io/injected']}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyDeployInjected, 2*time.Minute).Should(Succeed())

			By("Cleaning up T-2 resources")
			cmd = exec.Command("kubectl", "delete", "job", "test-job", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "deployment", "test-deploy", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "telemetryprofile", "workload-test", "-n", "default", "--ignore-not-found")
			utils.Run(cmd)
		})

		It("should inject correctly under restricted PSS and handle cert rotation (T-3)", func() {
			By("Creating a restricted namespace")
			cmd := exec.Command("kubectl", "create", "ns", "restricted-ns")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "label", "ns", "restricted-ns", "pod-security.kubernetes.io/enforce=restricted")
			utils.Run(cmd)

			By("Creating a TelemetryProfile in restricted-ns")
			profileYAML := `
apiVersion: telemetry.podbeacon.io/v1alpha1
kind: TelemetryProfile
metadata:
  name: restricted-test
  namespace: restricted-ns
spec:
  signals: ["traces"]
  exporter:
    endpoint: "otel-collector.monitoring.svc.cluster.local:4317"
    tls:
      insecure: true
`
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(profileYAML)
			utils.Run(cmd)
			time.Sleep(3 * time.Second)

			By("Rotating the webhook certificate")
			cmd = exec.Command("kubectl", "delete", "secret", "webhook-server-cert", "-n", namespace)
			utils.Run(cmd)
			// Wait for cert-manager to recreate it
			verifyCertRecreated := func(g Gomega) {
				cmd = exec.Command("kubectl", "get", "secret", "webhook-server-cert", "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(verifyCertRecreated, 1*time.Minute).Should(Succeed())
			time.Sleep(10 * time.Second) // wait for webhook server to pick up new cert and for admission registration to update

			By("Deploying a pod in restricted namespace")
			// A restricted pod needs a securityContext that drops all capabilities, runs as non-root, etc.
			podYAML := `
apiVersion: v1
kind: Pod
metadata:
  name: restricted-pod
  namespace: restricted-ns
  annotations:
    telemetry: "enable"
    podbeacon.io/profile: "restricted-test"
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: my-app
    image: busybox:latest
    command: ["sleep", "3600"]
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
`
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = bytes.NewBufferString(podYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create restricted pod")

			By("Verifying restricted pod is injected and starts successfully")
			verifyRestrictedInjected := func(g Gomega) {
				cmd = exec.Command("kubectl", "get", "pod", "restricted-pod", "-n", "restricted-ns", "-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"))

				cmd = exec.Command("kubectl", "get", "pod", "restricted-pod", "-n", "restricted-ns", "-o", "jsonpath={.metadata.annotations['podbeacon\\.io/injected']}")
				output, err = utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyRestrictedInjected, 2*time.Minute).Should(Succeed())

			By("Cleaning up T-3 resources")
			cmd = exec.Command("kubectl", "delete", "ns", "restricted-ns")
			utils.Run(cmd)
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	By("creating temporary file to store the token request")
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		By("executing kubectl command to create the token")
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		By("parsing the JSON output to extract the token")
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
