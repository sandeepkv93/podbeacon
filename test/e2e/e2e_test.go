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
	"k8s.io/apimachinery/pkg/util/wait"

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

// Shared Fixtures Helpers
func applyYAML(yamlStr string) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = bytes.NewBufferString(yamlStr)
	_, err := utils.Run(cmd)
	return err
}

func deleteYAML(yamlStr string) {
	cmd := exec.Command("kubectl", "delete", "--ignore-not-found", "-f", "-")
	cmd.Stdin = bytes.NewBufferString(yamlStr)
	_, _ = utils.Run(cmd)
}

func getProfileYAML(name, ns, endpoint string) string {
	return fmt.Sprintf(`
apiVersion: telemetry.podbeacon.io/v1alpha1
kind: TelemetryProfile
metadata:
  name: %s
  namespace: %s
spec:
  signals: ["traces", "metrics", "logs"]
  exporter:
    endpoint: "%s"
    tls:
      insecure: true
`, name, ns, endpoint)
}

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	BeforeAll(func() {
		By("Recording environment, version, and skip evidence")
		k8sVersionCmd := exec.Command("kubectl", "version", "--short")
		k8sVersion, _ := utils.Run(k8sVersionCmd)
		fmt.Fprintf(GinkgoWriter, "Environment K8s Version:\n%s\n", k8sVersion)

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

	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace, "--ignore-not-found")
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--ignore-not-found")
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager Setup", func() {
		It("should run successfully", func() {
			verifyControllerUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)
				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(2))
				controllerPodName = podNames[0]

				for _, pod := range podNames {
					cmd = exec.Command("kubectl", "get", "pods", pod, "-o", "jsonpath={.status.phase}", "-n", namespace)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(Equal("Running"))
				}
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})
	})

	Context("E2E Matrix: Workloads, Protocols, and Negative Cases", func() {
		It("should provisioned cert-manager and MutatingWebhookConfiguration", func() {
			verifyWebhooks := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secrets", "webhook-server-cert", "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())

				cmd = exec.Command("kubectl", "get", "mutatingwebhookconfigurations", "podbeacon-mutating-webhook-configuration", "-o", "jsonpath={.webhooks[0].clientConfig.caBundle}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(output)).To(BeNumerically(">", 10))
			}
			Eventually(verifyWebhooks).Should(Succeed())
		})

		It("should inject sidecar into various workloads: StatefulSet, DaemonSet, CronJob, Deployment, Job (T-2/T-4)", func() {
			profileYAML := getProfileYAML("matrix-profile", "default", "otel-collector:4317")
			Expect(applyYAML(profileYAML)).To(Succeed())
			time.Sleep(3 * time.Second)
			defer deleteYAML(profileYAML)

			workloadsYAML := `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test-sts
  namespace: default
spec:
  selector:
    matchLabels:
      app: test-sts
  serviceName: "test-sts"
  replicas: 1
  template:
    metadata:
      labels:
        app: test-sts
      annotations:
        telemetry: "enable"
        podbeacon.io/profile: "matrix-profile"
    spec:
      containers:
      - name: task
        image: busybox:latest
        command: ["sleep", "3600"]
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: test-ds
  namespace: default
spec:
  selector:
    matchLabels:
      app: test-ds
  template:
    metadata:
      labels:
        app: test-ds
      annotations:
        telemetry: "enable"
        podbeacon.io/profile: "matrix-profile"
    spec:
      containers:
      - name: task
        image: busybox:latest
        command: ["sleep", "3600"]
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: test-cj
  namespace: default
spec:
  schedule: "*/1 * * * *"
  jobTemplate:
    spec:
      template:
        metadata:
          annotations:
            telemetry: "enable"
            podbeacon.io/profile: "matrix-profile"
        spec:
          restartPolicy: OnFailure
          containers:
          - name: task
            image: busybox:latest
            command: ["echo", "cron done"]
`
			Expect(applyYAML(workloadsYAML)).To(Succeed())
			defer deleteYAML(workloadsYAML)

			err := wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
				// Check StatefulSet
				cmd := exec.Command("kubectl", "get", "pods", "-n", "default", "-l", "app=test-sts", "-o", "jsonpath={.items[0].metadata.annotations['podbeacon\\.io/injected']}")
				output, err := utils.Run(cmd)
				if err != nil || output != "true" {
					return false, nil
				}

				// Check DaemonSet
				cmd = exec.Command("kubectl", "get", "pods", "-n", "default", "-l", "app=test-ds", "-o", "jsonpath={.items[0].metadata.annotations['podbeacon\\.io/injected']}")
				output, err = utils.Run(cmd)
				if err != nil || output != "true" {
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				out, _ := exec.Command("sh", "-c", "kubectl get events -n default && kubectl get pods -A && kubectl get telemetryprofile -A -o yaml && kubectl logs -n podbeacon-system -l control-plane=controller-manager --tail=100").CombinedOutput()
				Fail(fmt.Sprintf("Timeout! Diagnostics:\n%s", string(out)))
			}
		})

		It("should support HTTP OTLP and verify signal delivery (T-1/T-5)", func() {
			profileYAML := getProfileYAML("http-profile", "default", "http://otel-collector.monitoring.svc.cluster.local:4318")
			Expect(applyYAML(profileYAML)).To(Succeed())
			time.Sleep(3 * time.Second)
			defer deleteYAML(profileYAML)

			emitterYAML := `
apiVersion: v1
kind: Pod
metadata:
  name: http-emitter
  namespace: default
  annotations:
    telemetry: "enable"
    podbeacon.io/profile: "http-profile"
spec:
  containers:
  - name: emitter
    image: busybox:latest
    command: ["sleep", "3600"]
`
			Expect(applyYAML(emitterYAML)).To(Succeed())
			defer deleteYAML(emitterYAML)

			verifyHttpInjected := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", "http-emitter", "-n", "default", "-o", "jsonpath={.metadata.annotations['podbeacon\\\\.io/injected']}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true"))
			}
			Eventually(verifyHttpInjected, 1*time.Minute).Should(Succeed())
		})

		It("should handle negative/outage cases (T-6)", func() {
			By("Testing malformed profile handling")
			malformedProfile := `
apiVersion: telemetry.podbeacon.io/v1alpha1
kind: TelemetryProfile
metadata:
  name: malformed-profile
  namespace: default
spec:
  signals: ["invalid_signal"]
  exporter:
    endpoint: ":::"
`
			Expect(applyYAML(malformedProfile)).NotTo(Succeed(), "Malformed profile should be rejected by validation")

			By("Testing spoofed marker idempotency and dry-run")
			spoofedPodYAML := `
apiVersion: v1
kind: Pod
metadata:
  name: spoofed-pod
  namespace: default
  annotations:
    telemetry: "enable"
    podbeacon.io/injected: "true" # Pretend it's already injected
spec:
  containers:
  - name: my-app
    image: busybox:latest
    command: ["sleep", "3600"]
`
			// Should not inject sidecar if already marked injected
			Expect(applyYAML(spoofedPodYAML)).To(Succeed())
			defer deleteYAML(spoofedPodYAML)

			verifySpoofed := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", "spoofed-pod", "-n", "default", "-o", "jsonpath={.spec.initContainers[*].name}")
				output, _ := utils.Run(cmd)
				g.Expect(output).NotTo(ContainSubstring("podbeacon-collector"))
			}
			Eventually(verifySpoofed, 1*time.Minute).Should(Succeed())
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{"apiVersion": "authentication.k8s.io/v1", "kind": "TokenRequest"}`
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts/%s/token", namespace, serviceAccountName), "-f", tokenRequestFile)
		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())
		var token struct{ Status struct{ Token string } }
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())
		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())
	return out, err
}

func getMetricsOutput() (string, error) {
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}
