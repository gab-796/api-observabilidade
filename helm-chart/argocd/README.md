# Deploying with ArgoCD

Este documento explica como usar este Helm chart com ArgoCD para deployments automatizados e GitOps.

## Estrutura para ArgoCD

```
helm-chart/
├── argocd/
│   ├── application.yaml      # ArgoCD Application CRD
│   └── values-argocd.yaml   # Values específicos para ArgoCD
├── pdi-gabriel/             # Helm chart original
└── Makefile                 # Comandos para ArgoCD
```

## Pré-requisitos

1. **ArgoCD instalado no cluster:**
   ```bash
   kubectl create namespace argocd
   kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
   ```

2. **ArgoCD CLI (opcional):**
   ```bash
   curl -sSL -o argocd-linux-amd64 https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
   sudo install -m 555 argocd-linux-amd64 /usr/local/bin/argocd
   ```

3. **Repositório Git acessível pelo ArgoCD**

## Deploy usando ArgoCD

### 1. Aplicar a Application

```bash
# Deploy da Application do ArgoCD
make argocd-apply

# Ou manualmente:
kubectl apply -f ./argocd/application.yaml
```

### 2. Monitorar o deployment

```bash
# Status via CLI
make argocd-status

# Sync manual se necessário
make argocd-sync

# Via UI (port-forward se necessário)
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Acesse: https://localhost:8080
```

### 3. Configurar credenciais iniciais

```bash
# Obter senha inicial do admin
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

# Login
argocd login localhost:8080
```

## Configurações importantes

### Valores específicos para ArgoCD

O arquivo `argocd/values-argocd.yaml` contém configurações otimizadas para produção:

- **Persistência habilitada** (grafana, loki, mimir, etc.)
- **Recursos aumentados** para componentes
- **Health checks mais tolerantes**
- **Ingress com TLS**
- **Autenticação não-anônima no Grafana**

### Sync Policies

A Application está configurada com:

- **Automated sync:** `prune: true`, `selfHeal: true`
- **Retry logic:** 5 tentativas com backoff exponencial
- **CreateNamespace:** Cria namespace automaticamente

### Ignorar diferenças

Configurado para ignorar:
- Mudanças de réplicas (gerenciadas pelo HPA)
- Anotações dinâmicas do Grafana

## Ambientes diferentes

### Development
```bash
# Usar values padrão (ephemeral, recursos menores)
helm upgrade --install api-observabilidade ./pdi-gabriel -n api-app-go
```

### Production via ArgoCD
```bash
# Aplicar Application que usa values-argocd.yaml
make argocd-apply
```

### Staging (exemplo)
```bash
# Criar values-staging.yaml e ajustar application.yaml
helm upgrade --install api-observabilidade ./pdi-gabriel \
  -n api-app-staging \
  --values ./pdi-gabriel/values.yaml \
  --values ./argocd/values-staging.yaml
```

## Troubleshooting

### Application não synca

```bash
# Check events
kubectl describe application api-observabilidade -n argocd

# Check logs do ArgoCD
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller
```

### Recursos ficam OutOfSync

```bash
# Sync manual
argocd app sync api-observabilidade

# Hard refresh
argocd app sync api-observabilidade --force
```

### Health checks falham

```bash
# Verificar pods no namespace target
kubectl get pods -n api-app-go

# Logs específicos
kubectl logs -n api-app-go -l app=inventory-app
kubectl logs -n api-app-go -l app=grafana
```

## Customizações

### Adicionar environments

1. Criar `values-{env}.yaml` em `argocd/`
2. Duplicar `application.yaml` com nome diferente
3. Ajustar `spec.source.helm.valueFiles`
4. Aplicar: `kubectl apply -f argocd/application-{env}.yaml`

### Usar Projects customizados

1. Editar `argocd/application.yaml`
2. Mudar `spec.project` de `default` para seu project
3. Aplicar o Project antes da Application

### Integrar com External Secrets

1. Instalar External Secrets Operator
2. Criar SecretStore (Vault, AWS, etc.)
3. Habilitar `externalSecrets.enabled: true` em values-argocd.yaml
4. Configurar secrets externos para senhas sensíveis

## Comandos úteis

```bash
# Status completo
make argocd-status

# Sync forçado
make argocd-sync

# Deletar application (mantém resources)
make argocd-delete

# Deletar tudo (application + resources)
argocd app delete api-observabilidade --cascade

# Refresh sem sync
argocd app get api-observabilidade --refresh

# Diff entre Git e cluster
argocd app diff api-observabilidade
```

## Considerações de segurança

Para produção, considere:

1. **Usar RBAC granular** (Projects customizados)
2. **External Secrets** para senhas
3. **Network Policies** se suportado
4. **Pod Security Standards**
5. **TLS/mTLS** entre componentes
6. **Backup** de PVCs importantes
7. **Monitoring** do próprio ArgoCD
