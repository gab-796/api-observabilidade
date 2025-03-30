# This Makefile only create the k8s kind cluster!
# 
# requirements:
# - docker
# - kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation
# 
# observations:
# ifneq is a make command. Therefore, we should not use TAB characters as the first characters in the line,
# as it would be passed to the shell, and the shell doesn't know anything about ifneq.
# the same is valid to comments and any make commands.

CLUSTER_NAME = $(shell echo karavela-$(shell hostname) | tr '[:upper:]' '[:lower:]')
CLUSTER_EXISTS = $(shell kind get clusters -q | grep $(CLUSTER_NAME))

# DOCKER_SUBNET should be a valid ipv4 address.
IPV4_INDEX = $(shell docker network inspect -f '{{.EnableIPv6}}' kind)
ifeq ($(IPV4_INDEX), "true")
	IPV4_INDEX = 1
else
	IPV4_INDEX = 0
endif
DOCKER_SUBNET = $(shell docker network inspect -f '{{(index .IPAM.Config $(IPV4_INDEX)).Subnet}}' kind)

define METALLB_CONFIG
cat <<EOF
{
	"apiVersion": "v1",
	"kind": "List",
	"items": [
		{
			"apiVersion": "metallb.io/v1beta1",
			"kind": "IPAddressPool",
			"metadata": {
				"name": "metallb-pool",
				"namespace": "metallb-system"
			},
			"spec": {
				"addresses": [
					"$(DOCKER_SUBNET)"
				]
			}
		},
		{
			"apiVersion": "metallb.io/v1beta1",
			"kind": "L2Advertisement",
			"metadata": {
				"name": "empty",
				"namespace": "metallb-system"
			}
		}
	]
}
EOF
endef

export METALLB_CONFIG

install-cilium:
	helm repo add cilium https://helm.cilium.io/
	helm upgrade --install cilium cilium/cilium --version 1.15.0 --namespace kube-system --set ipam.mode=kubernetes
	kubectl rollout -n kube-system status daemonset cilium

install-metrics-server:
	helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
	helm upgrade --install metrics-server metrics-server/metrics-server --namespace kube-system --set args={--kubelet-insecure-tls}
	kubectl rollout -n kube-system status deployment metrics-server

install-metallb:
	kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml
	kubectl rollout status -n metallb-system deployment controller
	@ eval "$$METALLB_CONFIG" | kubectl apply -f -

install-ingress-nginx:
	helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
	helm upgrade --install -n kube-system ingress-nginx ingress-nginx/ingress-nginx --version 4.10.1
	kubectl rollout -n kube-system status deployment ingress-nginx-controller

install-opentelemetry-demo:
	helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
	helm upgrade --install opentelemetry-demo observability-helm-charts/observability-generator-otlp-demo --namespace observability-generator-otlp --create-namespace -f charts/observability-generator-otlp-demo/values-local.yaml

install-dependencies: install-cilium install-metrics-server install-metallb install-ingress-nginx install-opentelemetry-demo install-kube-state-metrics

.create-cluster:
    # Check if the cluster exists
    ifneq ($(CLUSTER_EXISTS), $(CLUSTER_NAME))
        # Ensure the creation of ipv4 network
		docker network inspect kind >/dev/null 2>&1 || docker network create kind
		kind create cluster --name $(CLUSTER_NAME) --config=cluster/config.yaml --wait 10s
    else
		kubectl cluster-info --context kind-$(CLUSTER_NAME)
    endif
	sudo sysctl fs.inotify.max_user_watches=524288
	sudo sysctl fs.inotify.max_user_instances=512

create-cluster: .create-cluster install-dependencies

delete-cluster:
	kind delete clusters $(CLUSTER_NAME)
