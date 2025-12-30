# usado para instalação via Helm, sem ArgoCD. Mantido apenas para referência.

# Variáveis
NAMESPACE ?= api-app-go
RELEASE_NAME ?= api-observabilidade
CHART_PATH ?= ./pdi-gabriel

# Comandos principais
install:
	helm upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
	    --namespace $(NAMESPACE) \
	    --create-namespace

uninstall:
	helm uninstall $(RELEASE_NAME) --namespace $(NAMESPACE)

# Utilitários
lint:
	helm lint $(CHART_PATH)

template:
	helm template $(RELEASE_NAME) $(CHART_PATH) \
	    --namespace $(NAMESPACE) \
	    --debug

dry-run:
	helm upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
	    --namespace $(NAMESPACE) \
	    --create-namespace \
	    --dry-run

# Status e debug
status:
	helm status $(RELEASE_NAME) --namespace $(NAMESPACE)

logs:
	kubectl logs -n $(NAMESPACE) -l app=$(RELEASE_NAME) --tail=100 -f

# Limpeza
clean:
	kubectl delete namespace $(NAMESPACE) --ignore-not-found=true

.PHONY: install uninstall install-dev install-prod lint template dry-run status logs clean