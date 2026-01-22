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

⚠️ **Por que essa etapa é necessária?**

O Vault é deployado em **modo development** para facilitar o lab. Neste modo:
- ✅ **Inicialização automática** (não precisa de `vault operator init`)
- ✅ **Auto-unsealed** (não precisa fazer unseal manualmente)
- ✅ **Token root fixo** (`root`) para facilitar acesso
- ❌ **Dados NÃO são persistentes** (se o pod reiniciar, perde tudo)
- ❌ **NÃO recomendado para produção**

**Fluxo de sincronização de secrets:**
```
1. Vault (infraestrutura) → deployado pelo ArgoCD
2. [VOCÊ] cria secrets manualmente → dentro do Vault
3. External Secrets Operator → sincroniza Vault → Kubernetes Secret
4. MySQL/API → leem do Kubernetes Secret nativo
```

#### 5.1. Criar secrets no Vault

Esta é uma etapa **CRÍTICA** e **OBRIGATÓRIA**. O Vault foi deployado em modo dev, mas está **vazio**. Você precisa configurar as senhas manualmente.

##### Como o fluxo funciona:

```
Vault (empty) → [VOCÊ cria secrets] → External Secrets Operator sincroniza → Kubernetes Secret → MySQL/API usa
```

##### Passo a passo detalhado:

```bash
# 1. Esperar Vault ficar pronto
kubectl wait --for=condition=ready pod -l app=vault -n api-app-go --timeout=120s

# 2. Obter nome do pod do Vault
VAULT_POD=$(kubectl get pod -n api-app-go -l app=vault -o jsonpath='{.items[0].metadata.name}')

# 3. Criar secrets no Vault com suas senhas
# ⚠️ ATENÇÃO: Altere 'admin' para suas senhas reais!
kubectl exec -n api-app-go $VAULT_POD -- sh -c "
  export VAULT_ADDR='http://127.0.0.1:8200'
  export VAULT_TOKEN='root'
  vault kv put secret/inventory-app/database \
    MYSQL_ROOT_PASSWORD=admin \
    DB_PASSWORD=admin
"

# 4. Verificar se foi criado corretamente
kubectl exec -n api-app-go $VAULT_POD -- sh -c "
  export VAULT_ADDR='http://127.0.0.1:8200'
  export VAULT_TOKEN='root'
  vault kv get secret/inventory-app/database
"
```

##### Customizar senhas:

Para usar senhas diferentes, modifique o comando no passo 3:

```bash
# Exemplo com senhas customizadas:
kubectl exec -n api-app-go $VAULT_POD -- sh -c "
  export VAULT_ADDR='http://127.0.0.1:8200'
  export VAULT_TOKEN='root'
  vault kv put secret/inventory-app/database \
    MYSQL_ROOT_PASSWORD='SuaSenhaSegura123!' \
    DB_PASSWORD='OutraSenhaSegura456!'
"
```

##### Estrutura dos secrets no Vault:

- **Path**: `secret/inventory-app/database`
- **Keys disponíveis**:
  - `MYSQL_ROOT_PASSWORD`: Senha do root do MySQL
  - `DB_PASSWORD`: Senha do usuário da aplicação

Esses valores serão sincronizados automaticamente pelo External Secrets Operator para **dois** Kubernetes Secrets:
- `mysql-secrets` → usado pelo MySQL
- `inventory-app-secrets` → usado pela API

**Por que dois secrets?**
- Separação de responsabilidades (MySQL vs API)
- Ambos leem do mesmo path no Vault (`secret/inventory-app/database`)
- Facilita controle de acesso granular no futuro

##### Em ambiente de produção:

Para produção, você **NÃO deve** usar Vault em dev mode. As diferenças principais:

| Dev Mode (LAB) | Produção |
|----------------|----------|
| Token fixo `root` | Autenticação Kubernetes/OIDC |
| Dados em memória | Backend persistente (Consul, Raft, etcd) |
| Single instance | Alta disponibilidade (3+ réplicas) |
| HTTP | TLS obrigatório |
| Auto-unsealed | KMS auto-unseal ou manual |
| Secrets criados manualmente | Integração com CI/CD |

Ver documentação completa em: [api-k8s/2.0-Vault/SETUP.md](../../api-k8s/2.0-Vault/SETUP.md)

#### 5.2. Verificar sincronização dos External Secrets

**Importante:** O ArgoCD cria **2 ExternalSecrets** automaticamente:
1. `mysql-credentials` → cria secret `mysql-secrets` (para o MySQL)
2. `inventory-app-credentials` → cria secret `inventory-app-secrets` (para a API)

```bash
# Verificar se SecretStore foi criado (deve mostrar Ready: True)
kubectl get secretstore -n api-app-go

# Verificar AMBOS os ExternalSecrets (deve mostrar status: SecretSynced e Ready:True)
kubectl get externalsecret -n api-app-go

# Detalhes de cada um
kubectl describe externalsecret mysql-credentials -n api-app-go
kubectl describe externalsecret inventory-app-credentials -n api-app-go

# Verificar se os dois secrets do Kubernetes foram criados
kubectl get secret mysql-secrets -n api-app-go
kubectl get secret inventory-app-secrets -n api-app-go

# Ver conteúdo dos secrets (deve mostrar 'admin' ou suas senhas customizadas)
kubectl get secret mysql-secrets -n api-app-go -o jsonpath='{.data.MYSQL_ROOT_PASSWORD}' | base64 -d && echo
kubectl get secret mysql-secrets -n api-app-go -o jsonpath='{.data.DB_PASSWORD}' | base64 -d && echo
kubectl get secret inventory-app-secrets -n api-app-go -o jsonpath='{.data.DB_PASSWORD}' | base64 -d && echo
```

#### 5.3. Troubleshooting se não sincronizar

⚠️ **Warnings de "Secret does not exist" ou "UpdateFailed" são NORMAIS** durante a inicialização!  
Eles ocorrem porque:
1. External Secrets Operator tenta sincronizar antes do Vault estar pronto
2. SecretStore ainda não existe
3. Secret ainda não foi criado no Vault (passo 5.1)

**Após criar os secrets no Vault (passo 5.1), os ExternalSecrets sincronizarão automaticamente em ~15 segundos.**

```bash
# Ver logs do External Secrets Operator
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets --tail=50

# Ver eventos dos ExternalSecrets
kubectl describe externalsecret mysql-credentials -n api-app-go
kubectl describe externalsecret inventory-app-credentials -n api-app-go
kubectl describe secretstore vault-backend -n api-app-go

# Testar conectividade com Vault
kubectl run test --rm -it --image=curlimages/curl -- curl -v http://vault.api-app-go:8200/v1/sys/health

# Se os secrets não foram criados, verificar se o path no Vault está correto
VAULT_POD=$(kubectl get pod -n api-app-go -l app=vault -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n api-app-go $VAULT_POD -- sh -c "
  export VAULT_ADDR='http://127.0.0.1:8200'
  export VAULT_TOKEN='root'
  vault kv get secret/inventory-app/database
"
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

### API não consegue conectar no MySQL (Access Denied)

**Erro típico:** `Error 1045 (28000): Access denied for user 'root'@'<IP>' (using password: YES)`

**Causa:** MySQL e API estão usando senhas diferentes.

**Diagnóstico:**

```bash
# 1. Verificar se os secrets existem e têm valores
kubectl get secret mysql-secrets -n api-app-go -o jsonpath='{.data.MYSQL_ROOT_PASSWORD}' | base64 -d && echo
kubectl get secret inventory-app-secrets -n api-app-go -o jsonpath='{.data.DB_PASSWORD}' | base64 -d && echo

# 2. Verificar variáveis de ambiente no pod do MySQL
kubectl exec -n api-app-go -l app=mysql -- printenv | grep MYSQL

# 3. Verificar variáveis de ambiente no pod da API
kubectl exec -n api-app-go -l app=inventory-app -- printenv | grep DB_

# 4. Se as senhas forem diferentes, corrigir no Vault e aguardar sincronização
VAULT_POD=$(kubectl get pod -n api-app-go -l app=vault -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n api-app-go $VAULT_POD -- sh -c "
  export VAULT_ADDR='http://127.0.0.1:8200'
  export VAULT_TOKEN='root'
  vault kv put secret/inventory-app/database \
    MYSQL_ROOT_PASSWORD=admin \
    DB_PASSWORD=admin
"

# 5. Aguardar ExternalSecrets sincronizarem (~15s)
kubectl get externalsecret -n api-app-go -w

# 6. Reiniciar MySQL para pegar nova senha
kubectl rollout restart deployment/mysql -n api-app-go

# 7. Reiniciar API após MySQL estar pronto
kubectl wait --for=condition=ready pod -l app=mysql -n api-app-go --timeout=60s
kubectl rollout restart deployment/inventory-app -n api-app-go
```

**Importante:** 
- MySQL usa `MYSQL_ROOT_PASSWORD` do secret `mysql-secrets`
- API usa `DB_PASSWORD` do secret `inventory-app-secrets`
- **Ambos devem ter o mesmo valor** (vêm do mesmo path no Vault)

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