NAMESPACE_FILE := namespace.yaml
KUBECTL := kubectl

.PHONY: all apply-namespace apply-resources destroy destroy-namespace

all: apply-namespace apply-resources

apply-namespace:
	@echo "Aplicando o namespace definido em $(NAMESPACE_FILE)"
	$(KUBECTL) apply -f $(NAMESPACE_FILE)

apply-resources:
	@echo "Aplicando os recursos definidos no diretório atual"
	$(KUBECTL) apply -f .

destroy-namespace:
	@echo "Excluindo o namespace definido em $(NAMESPACE_FILE)"
	$(KUBECTL) delete -f $(NAMESPACE_FILE) --ignore-not-found

destroy: destroy-namespace
	@echo "Ambiente destruído"