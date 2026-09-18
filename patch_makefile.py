import sys

with open('Makefile', 'r') as f:
    content = f.read()

import re

# We need to replace setup-test-e2e properly.
old_case = """	@case "$$($(KIND) get clusters)" in \\
		*"$(KIND_CLUSTER)"*) \\
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \\
		*) \\
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \\
			$(KIND) create cluster --name $(KIND_CLUSTER) --image kindest/node:v1.31.0 ;; \\
	esac"""

new_case = """	@mkdir -p $(shell pwd)/bin
	@case "$$($(KIND) get clusters)" in \\
		*"$(KIND_CLUSTER)"*) \\
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \\
		*) \\
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \\
			$(KIND) create cluster --name $(KIND_CLUSTER) --image kindest/node:v1.33.0 --kubeconfig $(KUBECONFIG) && touch $(shell pwd)/bin/.kind_cluster_created ;; \\
	esac
	@$(KIND) export kubeconfig --name $(KIND_CLUSTER) --kubeconfig $(KUBECONFIG)"""

content = content.replace(old_case, new_case)

with open('Makefile', 'w') as f:
    f.write(content)
