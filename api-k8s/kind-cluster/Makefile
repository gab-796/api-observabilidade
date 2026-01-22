# This Makefile only create the k8s kind cluster!
# 
# requirements:
# - docker
# - kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation
# super easy move Kind Installation mode:
# [ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.27.0/kind-linux-amd64
# chmod +x ./kind
# sudo mv ./kind /usr/local/bin/kind
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

install-dependencies: ## Instala todas as dependências de base do cluster
	@$(MAKE) install-cilium
	@$(MAKE) install-metrics-server
	@$(MAKE) install-observability-crds
	@$(MAKE) install-metallb
	@$(MAKE) install-kyverno
	@$(MAKE) install-ingress-nginx
	@$(MAKE) install-keda
	@$(MAKE) install-cert-manager
	@$(MAKE) install-external-secrets
	@$(MAKE) install-chaos-operator
	@$(MAKE) install-k6-operator
	@$(MAKE) install-argo-rollout

install-cert-manager:
	helm repo add jetstack https://charts.jetstack.io
	helm upgrade --install cert-manager jetstack/cert-manager \
	--namespace cert-manager \
	--create-namespace \
	--version v1.9.1 \
	--set installCRDs=true

uninstall-cert-manager:
	helm uninstall cert-manager --namespace=cert-manager

install-cilium:
	helm repo add cilium https://helm.cilium.io/ --force-update || true
	helm upgrade --install cilium cilium/cilium --version 1.18.1 --namespace kube-system \
		--set ipam.mode=kubernetes
	kubectl rollout -n kube-system status daemonset cilium

install-metrics-server:
	helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/ --force-update || true
	helm upgrade --install metrics-server metrics-server/metrics-server --namespace kube-system \
		--set args={--kubelet-insecure-tls}
	kubectl rollout -n kube-system status deployment metrics-server

install-metallb:
	kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml
	kubectl rollout status -n metallb-system deployment controller
	@ eval "$$METALLB_CONFIG" | kubectl apply -f -

install-ingress-nginx:
	helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx --force-update || true
	helm upgrade --install -n kube-system ingress-nginx ingress-nginx/ingress-nginx --version 4.10.1 \
        --set controller.metrics.enabled=true \
        --set controller.podAnnotations."ad\.datadoghq\.com/controller\.checks"='\{"nginx_ingress_controller":{"instances":[{"prometheus_url":"http://%%host%%:10254/metrics"}]}\}'
	kubectl rollout -n kube-system status deployment ingress-nginx-controller

install-keda:
	helm repo add kedacore https://kedacore.github.io/charts --force-update || true
	helm upgrade --install keda kedacore/keda --namespace keda --create-namespace --version 2.16.1

install-kyverno:
	helm repo add kyverno https://kyverno.github.io/kyverno/ --force-update || true
	# Get values.yaml used in Karavela clusters
    ifndef GITHUB_PERSONAL_ACCESS_TOKEN
		$(error GITHUB_PERSONAL_ACCESS_TOKEN is undefined)
    else
		curl -so /tmp/values.yaml -H "Authorization: token $$GITHUB_PERSONAL_ACCESS_TOKEN" https://raw.githubusercontent.com/stone-payments/platform-k8s-infrastructure/refs/heads/main/addons/kyverno/chart/values.yaml
		@echo "Detectando versão do yq: $(YQ_VERSION_STRING)"
		# Removing image references from values.yaml with yq (sudo apt install yq) and creating a new values-kyverno.yaml
		@yq $(YQ_OUTPUT_FLAG) '.kyverno | del(.[]?.image, .crds?.migration?.image, .cleanupJobs?[]?.image, .admissionController?.initContainer?.image, .admissionController?.container?.image, .reportsController?.resources)' /tmp/values.yaml > /tmp/values-kyverno.yaml
		# Install kyverno with the same settings used in Karavela
		helm upgrade --install kyverno kyverno/kyverno -n kyverno --create-namespace --version 3.5.1 -f /tmp/values-kyverno.yaml
    endif

uninstall-kyverno:
	helm uninstall kyverno --namespace kyverno

install-k6-operator: ## Instala o Grafana k6-operator
	@helm repo add grafana https://grafana.github.io/helm-charts --force-update || true
	@helm upgrade --install k6-operator grafana/k6-operator --namespace k6-operator --create-namespace --set namespace.create=false

uninstall-k6-operator:
	helm uninstall k6-operator --namespace k6-operator --ignore-not-found
	kubectl delete namespace k6-operator --ignore-not-found

install-external-secrets:
	helm repo add external-secrets https://charts.external-secrets.io --force-update || true
	helm upgrade --install external-secrets external-secrets/external-secrets --version 0.16.1 --namespace=external-secrets --create-namespace --set installCRDs=true

uninstall-external-secrets:
	helm uninstall external-secrets --namespace=external-secrets

install-vault-dev:
	helm repo add hashicorp https://helm.releases.hashicorp.com --force-update || true
	kubectl create secret generic vault-token --from-literal=token=root -n default
	helm upgrade --install vault hashicorp/vault --namespace vault --create-namespace \
		--set server.ingress.enabled="true" \
		--set server.ingress.ingressClassName=nginx \
		--set server.ingress.hosts[0].host="vault.local"  \
		--set server.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"=\"false\" \
		--set server.dev.enabled="true" \
		--set ui.enabled="true"

uninstall-vault-dev:
	helm uninstall vault --namespace vault

install-chaos-operator:
	helm repo add litmuschaos https://litmuschaos.github.io/litmus-helm/
	helm upgrade --install kubernetes-chaos litmuschaos/kubernetes-chaos --namespace chaos-operator --create-namespace
	helm upgrade --install chaos-operator litmuschaos/litmus-core --namespace chaos-operator --create-namespace \
		--set operatorMode="admin"

uninstall-chaos-operator:
	helm uninstall chaos-operator --namespace chaos-operator --ignore-not-found
	kubectl delete namespace chaos-operator --ignore-not-found

install-observability-crds:
    ifndef GITHUB_PERSONAL_ACCESS_TOKEN
		$(error GITHUB_PERSONAL_ACCESS_TOKEN is undefined)
    else
		helm repo add observability-helm-charts --force-update --username $$GITHUB_PERSONAL_ACCESS_TOKEN --password $$GITHUB_PERSONAL_ACCESS_TOKEN https://raw.githubusercontent.com/stone-payments/observability-helm-charts/gh-pages || true
		helm upgrade --install observability-crds observability-helm-charts/observability-crds \
			--namespace observability-crds --create-namespace \
			--dependency-update \
			--set prometheus-operator-crds.crds.alertmanagers.enabled=true \
			--set prometheus-operator-crds.crds.prometheuses.enabled=true \
			--set prometheus-operator-crds.crds.prometheusrules.enabled=true \
			--set prometheus-operator-crds.crds.servicemonitors.enabled=true \
			--set prometheus-operator-crds.crds.podmonitors.enabled=true \
			--set prometheus-operator-crds.crds.thanosrulers.enabled=true
    endif

uninstall-observability-crds:
	helm uninstall observability-crds --namespace observability-crds --ignore-not-found

install-argo-rollout:
	helm repo add argo https://argoproj.github.io/argo-helm
	helm upgrade --install argo-rollouts argo/argo-rollouts \
		--namespace argo-rollouts --create-namespace \
		--set dashboard.enabled=true \
		--set dashboard.ingress.enabled=true \
		--set dashboard.ingress.ingressClassName=nginx \
		--set dashboard.ingress.hosts[0]=rollouts.local \
		--set dashboard.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="false"

uninstall-argo-rollout:
	helm uninstall argo-rollouts argo/argo-rollouts --namespace argo-rollouts

install-argocd:
	helm repo add argo https://argoproj.github.io/argo-helm --force-update
    ifndef GITHUB_PERSONAL_ACCESS_TOKEN
		$(error GITHUB_PERSONAL_ACCESS_TOKEN is undefined)
    else
		helm upgrade --install argo-cd argo/argo-cd --namespace argocd --create-namespace --version 7.8.2 \
			--set server.ingress.enabled=true \
			--set server.ingress.ingressClassName=nginx \
			--set server.ingress.hostname=argocd.local \
			--set server.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"=\"false\" \
			--set configs.params."server\.insecure"=true \
			--set configs.credentialTemplates."https-creds".url=https://github.com/stone-payments \
			--set configs.credentialTemplates."https-creds".password=$$GITHUB_PERSONAL_ACCESS_TOKEN
    endif

install_ksm: ## Instala kube-state-metrics com annotations para OTEL
	@echo "Instalando kube-state-metrics..."
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts || true
	helm repo update prometheus-community
	helm upgrade --install kube-state-metrics prometheus-community/kube-state-metrics \
		--create-namespace \
		--namespace ksm \
		--set-string "podAnnotations.telemetry\.karavela\.stone\.tech/metrics\.scrape=true" \
		--set-string "podAnnotations.prometheus\.io/port=8080" \
		--set-string "podAnnotations.prometheus\.io/path=/metrics" \
		--set-string "podAnnotations.telemetry\.karavela\.stone\.tech/metrics\.infra-cardinality=low"

uninstall_ksm: ## Desinstala kube-state-metrics
	helm uninstall kube-state-metrics --namespace kube-system

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
