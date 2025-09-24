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

2. **Repositório Git acessível pelo ArgoCD**

## Deploy usando ArgoCD

### 1. Aplicar a Application

```bash
# Deploy da Application do ArgoCD
make argocd-apply

# Ou manualmente:
kubectl apply -f ./argocd/application.yaml
```

### 2. Monitorar o deployment

Entre na UI do ARgoCD via o ingress `argocd.local` com as credenciais do ArgoCD obtidas abaixo.

### 3. Configurar credenciais iniciais

```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

## Configurações importantes

### Sync Policies

A Application está configurada com:

- **Automated sync:** `prune: true`, `selfHeal: true`
- **Retry logic:** 5 tentativas com backoff exponencial
- **CreateNamespace:** Cria namespace automaticamente

### Ignorar diferenças

Configurado para ignorar:
- Mudanças de réplicas (gerenciadas pelo HPA)
- Anotações dinâmicas do Grafana


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