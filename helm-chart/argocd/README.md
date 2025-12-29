# Deploying with ArgoCD
Este documento explica como usar este Helm chart com ArgoCD para deployments automatizados e GitOps.

## Estrutura para ArgoCD

```
helm-chart/
├── argocd/
│   ├── application.yaml      # ArgoCD Application CRD (API + Vault)
│   └── values-argocd.yaml   # Values específicos para ArgoCD
├── pdi-gabriel/             # Helm chart original
└── Makefile                 # Comandos para ArgoCD

k8s/
└── vault/
    └── vault.yaml           # Manifestos do Vault
```

## Pré-requisitos

1. **ArgoCD instalado no cluster:**
   ```bash
   kubectl create namespace argocd
   kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
   
   # Aguardar pods prontos
   kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=300s
   ```

2. **Ingress Controller (NGINX):**
   ```bash
   # Para Kind cluster
   kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.1/deploy/static/provider/kind/deploy.yaml
   ```

3. **External Secrets Operator (para Vault):**
   ```bash
   helm repo add external-secrets https://charts.external-secrets.io
   helm repo update
   helm install external-secrets external-secrets/external-secrets -n external-secrets --create-namespace --version 0.20.4
   ```

4. **Repositório Git acessível pelo ArgoCD**

5. **Configurar /etc/hosts:**
   ```bash
   echo "127.0.0.1 argocd.local grafana-web.local" | sudo tee -a /etc/hosts
   ```

## Deploy usando ArgoCD

### 1. Obter senha inicial do ArgoCD

```bash
# Pegar senha do admin
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo

# Port-forward (se não tiver Ingress)
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

### 2. Login no ArgoCD

```bash
# Via CLI
argocd login argocd.local --username admin --password <senha-obtida-acima> --insecure

# Ou via UI
# https://argocd.local (ou http://localhost:8080)
```

### 3. Aplicar as Applications

```bash
# Deploy das Applications (API Stack + Vault)
kubectl apply -f ./argocd/application.yaml

# Ou usando ArgoCD CLI
argocd app create -f ./argocd/application.yaml
```

### 4. Monitorar o deployment

```bash
# Via CLI
argocd app list
argocd app get api-observabilidade
argocd app get vault

# Via UI
# Acesse https://argocd.local
```

### 5. Aguardar Vault e configurar secrets (IMPORTANTE!)

```bash
# Esperar Vault ficar pronto
kubectl wait --for=condition=ready pod -l app=vault -n api-app-go --timeout=120s

# Criar secrets no Vault
VAULT_POD=$(kubectl get pod -n api-app-go -l app=vault -o jsonpath='{.items[0].metadata.name}')

kubectl exec -it -n api-app-go $VAULT_POD -- sh -c "
  export VAULT_ADDR='http://127.0.0.1:8200'
  export VAULT_TOKEN='root'
  vault kv put secret/inventory-app/database MYSQL_ROOT_PASSWORD=admin DB_PASSWORD=admin
"

# Verificar se sincronizou
kubectl get externalsecret -n api-app-go
kubectl get secret mysql-secrets -n api-app-go
```

### 6. Verificar se tudo subiu

```bash
kubectl get pods -n api-app-go
kubectl get svc -n api-app-go
kubectl get ingress -n api-app-go
```

**Acessar Grafana:**
```bash
http://grafana-web.local
User: admin | Password: admin
```

## Ordem de Deploy (Automático pelo ArgoCD)

O ArgoCD vai deployar nesta ordem (sync waves):

1. **Vault** → Gerenciamento de secrets
2. **External Secrets** → Sincronização Vault → K8s
3. **Stack de Observabilidade** → Mimir, Loki, Tempo, Pyroscope
4. **Coletores** → OTEL Collector, Alloy
5. **Grafana** → Visualização
6. **MySQL** → Banco de dados
7. **API** → Aplicação

## Configurações importantes

### Sync Policies

As Applications estão configuradas com:

- **Automated sync:** `prune: true`, `selfHeal: true`
- **Retry logic:** 5 tentativas com backoff exponencial
- **CreateNamespace:** Cria namespace automaticamente

### Ignorar diferenças

Configurado para ignorar:
- Mudanças de réplicas (gerenciadas pelo HPA)
- Anotações dinâmicas do Grafana

## Acessar as aplicações

```bash
# Grafana
http://grafana-web.local
User: admin | Password: admin

# ArgoCD
https://argocd.local
User: admin | Password: <obtida no passo 1>

# API
http://inventory.local (se configurado Ingress)
```

## Troubleshooting

### Application não synca

```bash
# Check events
kubectl describe application api-observabilidade -n argocd
kubectl describe application vault -n argocd

# Check logs do ArgoCD
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller --tail=50
```

### Vault não está pronto

```bash
# Ver logs
kubectl logs -n api-app-go -l app=vault

# Verificar health
kubectl exec -n api-app-go -l app=vault -- vault status
```

### ExternalSecret não sincroniza

```bash
# Verificar SecretStore
kubectl get secretstore -n api-app-go -o yaml

# Ver detalhes do ExternalSecret
kubectl describe externalsecret mysql-credentials -n api-app-go

# Logs do External Secrets Operator
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets
```

### Recursos ficam OutOfSync

```bash
# Sync manual
argocd app sync api-observabilidade
argocd app sync vault

# Hard refresh
argocd app sync api-observabilidade --force
argocd app sync vault --force
```

### Health checks falham

```bash
# Verificar pods no namespace target
kubectl get pods -n api-app-go

# Logs específicos
kubectl logs -n api-app-go -l app=inventory-app
kubectl logs -n api-app-go -l app=grafana
kubectl logs -n api-app-go -l app=vault
```

## Customizações

### Adicionar environments

1. Criar `values-{env}.yaml` em `argocd/`
2. Duplicar `application.yaml` com nome diferente
3. Ajustar `spec.source.helm.valueFiles`
4. Aplicar: `kubectl apply -f argocd/application-{env}.yaml`

### Usar Projects customizados

1. Editar `argocd/application.yaml`
2. Mudar `spec.project` de `default` para `observability` (já configurado)
3. O AppProject já está definido no `application.yaml`

### Integrar com Vault de Produção

1. Desabilitar Vault dev mode no `values-argocd.yaml`
2. Configurar SecretStore para apontar para Vault externo
3. Ajustar External Secrets para usar o SecretStore correto

## Comandos úteis

```bash
# Status completo
argocd app list
argocd app get api-observabilidade
argocd app get vault

# Sync forçado
argocd app sync api-observabilidade
argocd app sync vault

# Deletar application (mantém resources)
argocd app delete api-observabilidade --cascade=false
argocd app delete vault --cascade=false

# Deletar tudo (application + resources)
argocd app delete api-observabilidade --cascade
argocd app delete vault --cascade

# Refresh sem sync
argocd app get api-observabilidade --refresh
argocd app get vault --refresh

# Diff entre Git e cluster
argocd app diff api-observabilidade
argocd app diff vault

# Ver todos os recursos gerenciados
argocd app resources api-observabilidade
argocd app resources vault
```

## Limpeza completa

```bash
# Deletar Applications
kubectl delete -f ./argocd/application.yaml

# Deletar namespace
kubectl delete namespace api-app-go

# Deletar ArgoCD (opcional)
kubectl delete namespace argocd
```

## Notas de Produção

⚠️ **Este setup é para LAB/DEV**. Para produção:

1. **Vault:** Use Vault em modo HA com backend persistente (não dev mode)
2. **Secrets:** Use External Secrets com Vault/AWS/GCP real
3. **Persistência:** Habilite PVCs para Grafana, Loki, Mimir, Tempo
4. **TLS:** Configure TLS para todos os endpoints
5. **RBAC:** Configure RBAC apropriado no ArgoCD
6. **Backup:** Configure backup dos dados persistentes