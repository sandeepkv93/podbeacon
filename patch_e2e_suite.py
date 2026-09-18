import sys

with open('test/e2e/e2e_suite_test.go', 'r') as f:
    content = f.read()

import re

new_preflight = """	By("verifying we are running against the expected kind cluster")
	contextCmd := exec.Command("kubectl", "config", "current-context")
	out, err := utils.Run(contextCmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to get current context")
	expectedContext := "kind-" + os.Getenv("KIND_CLUSTER")
	if expectedContext == "kind-" {
		expectedContext = "kind-podbeacon-test-e2e"
	}
	ExpectWithOffset(1, out).To(ContainSubstring(expectedContext), "Not running against expected kind cluster. Refusing to run tests against ambient context.")

	By("building the manager image")"""

content = content.replace('\tBy("building the manager image")', new_preflight)

with open('test/e2e/e2e_suite_test.go', 'w') as f:
    f.write(content)
