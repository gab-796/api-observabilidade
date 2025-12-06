# Setup do Vault com External Secrets Operator no Kubernetes

Este guia explica como configurar o HashiCorp Vault + External Secrets Operator para gerenciar secrets da aplicação.

⚠️ **IMPORTANTE**: Para simplificar o lab, Vault e API estão no **mesmo namespace** (`api-app-go`).  
Isso **NÃO é recomendado para produção** - Vault deve ter namespace separado por questões de segurança!

## Por que External Secrets ao invés de Vault Agent Injector?

✅ **External Secrets Operator** é usado porque:
- Compatível com imagens **scratch** (não precisa de shell)
- Injeta secrets como **env vars nativas do Kubernetes**
- Não precisa modificar código da aplicação
- Multi-provider (funciona com Vault, AWS, GCP, Azure, etc)

## Ordem de Deploy

A ordem é **crítica** porque há dependências entre os componentes:

```
1. External Secrets Operator (via Helm)
   ↓
2. Vault (infraestrutura de secrets)
   ↓
3. Configuração do Vault (criar secrets no Vault)
   ↓
4. External Secrets (SecretStore + ExternalSecret)
   ↓
5. MySQL (banco de dados)
   ↓
6. API (depende dos Kubernetes Secrets gerados pelo ESO)
```

**Por quê essa ordem?**
- ✅ ESO precisa estar instalado ANTES de criar SecretStore/ExternalSecret
- ✅ Vault precisa estar rodando ANTES de configurar secrets nele
- ✅ Secrets precisam existir no Vault ANTES do ExternalSecret sincronizar
- ✅ Kubernetes Secret precisa existir ANTES da API subir (referenciado no deployment)

## 0. Instalar External Secrets Operator (PRÉ-REQUISITO)

```bash
# Adicionar repositório Helm
helm repo add external-secrets https://charts.external-secrets.io
helm repo update

# Criar namespace da API primeiro
kubectl create namespace api-app-go

# Instalar External Secrets Operator no mesmo namespace da API (versão 0.20.4)
# ⚠️ NÃO RECOMENDADO PARA PRODUÇÃO - ESO deve ter namespace separado!

# OPÇÃO A: Se já tiver ESO instalado em outro namespace, desinstale primeiro
# helm uninstall external-secrets -n external-secrets
# kubectl delete namespace external-secrets

# OPÇÃO B: Ou apenas use o ESO que já existe em outro namespace
# (pule a instalação abaixo e vá direto para o Passo 1)

# Instalar ESO (apenas se não tiver nenhum instalado)
helm install external-secrets external-secrets/external-secrets \
  -n api-app-go \
  --version 0.20.4

# Aguardar estar pronto
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=external-secrets \
  -n api-app-go \
  --timeout=120s
```


## 1. Deploy do Vault

```bash
# Aplicar todos os recursos do Vault (agora SEM Agent Injector)
kubectl apply -f vault-all-in-one.yaml

# Aguardar o pod do Vault estar pronto
kubectl wait --for=condition=ready pod -l app=vault -n api-app-go --timeout=120s
```

## 2. Configurar o Vault e criar secrets

```bash
# Entrar no pod do Vault (já vem com CLI vault instalado)
kubectl exec -it -n api-app-go $(kubectl get pod -n api-app-go -l app=vault -o jsonpath='{.items[0].metadata.name}') -- sh

# Dentro do pod, exportar variáveis
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'

# Verificar status
vault status
```

## 3. Habilitar e criar secrets no Vault

```bash
# Ainda dentro do pod do Vault:

# Habilitar KV v2
vault secrets enable -path=secret kv-v2

# Criar secrets da aplicação
vault kv put secret/inventory-app/database \
  MYSQL_ROOT_PASSWORD=admin \
  DB_PASSWORD=admin

# Verificar
vault kv get secret/inventory-app/database

# Sair do pod
exit
```

## 4. Deploy do External Secrets (SecretStore + ExternalSecret)

```bash
# Aplicar configuração do External Secrets
kubectl apply -f external-secrets.yaml

# Verificar se o SecretStore está pronto
kubectl get secretstore -n api-app-go

# Verificar se o ExternalSecret está sincronizando
kubectl get externalsecret -n api-app-go

# Verificar se o Kubernetes Secret foi criado
kubectl get secret mysql-secrets -n api-app-go
kubectl describe secret mysql-secrets -n api-app-go
```

## 5. Deploy do MySQL e OTEL Collector

**Agora que o Vault está configurado**, podemos subir o MySQL e OTEL:

```bash
# Deploy do MySQL (pode subir em paralelo ao Vault, mas fazemos depois por segurança)
kubectl apply -f mysql-deployment.yaml

# Aguardar MySQL estar pronto
kubectl wait --for=condition=ready pod -l app=mysql -n api-app-go --timeout=120s

# Deploy do OTEL Collector (opcional)
kubectl apply -f otel-collector-deployment.yaml
kubectl apply -f otel-collector-service.yaml
```

## 6. Deploy da aplicação (ÚLTIMO PASSO)

**IMPORTANTE**: A API só deve subir DEPOIS do ExternalSecret criar o mysql-secrets + MySQL rodando!

```bash
# Aplicar manifesto da API
kubectl apply -f api-all-in-one.yaml

# Verificar pods (deve ter apenas 1 container, sem sidecars!)
kubectl get pods -n api-app-go

# Ver logs da aplicação
kubectl logs -n api-app-go -l app=inventory-app -f
```

## 7. Verificar secrets injetadas

```bash
# Exec no container da aplicação
kubectl exec -it -n api-app-go -l app=inventory-app -c inventory-app -- sh

## 6. Deploy da aplicação (ÚLTIMO PASSO)

**IMPORTANTE**: A API só deve subir DEPOIS do ExternalSecret criar o mysql-secrets + MySQL rodando!

```bash
# Aplicar manifesto da API
kubectl apply -f api-all-in-one.yaml

# Verificar pods (deve ter apenas 1 container, sem sidecars!)
kubectl get pods -n api-app-go

# Ver logs da aplicação
kubectl logs -n api-app-go -l app=inventory-app -f
```

## 7. Verificar secrets injetadas

```bash
# Ver o Kubernetes Secret criado pelo External Secrets
kubectl get secret mysql-secrets -n api-app-go -o yaml

# Decodificar os valores (base64)
kubectl get secret mysql-secrets -n api-app-go -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
echo

# Verificar que a aplicação recebeu as env vars
kubectl exec -n api-app-go -l app=inventory-app -- env | grep -E '(DB_PASSWORD|MYSQL_ROOT_PASSWORD)'
```

## Resumo do Fluxo Completo

```bash
# 0. Instalar External Secrets Operator (uma vez só)
# Se já tiver em outro namespace: helm uninstall external-secrets -n external-secrets
kubectl create namespace api-app-go
helm install external-secrets external-secrets/external-secrets \
  -n api-app-go --version 0.20.4

# 1. Deploy do Vault
kubectl apply -f vault-all-in-one.yaml

# 2. Configurar Vault (dentro do pod)
kubectl exec -it -n api-app-go $(kubectl get pod -n api-app-go -l app=vault -o jsonpath='{.items[0].metadata.name}') -- sh
# Dentro do pod:
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'
vault secrets enable -path=secret kv-v2
vault kv put secret/inventory-app/database MYSQL_ROOT_PASSWORD=admin DB_PASSWORD=admin
exit

# 3. Deploy External Secrets
kubectl apply -f external-secrets.yaml

# 4. Deploy MySQL
kubectl apply -f mysql-deployment.yaml

# 5. Deploy OTEL (opcional)
kubectl apply -f otel-collector-deployment.yaml
kubectl apply -f otel-collector-service.yaml

# 6. Deploy da API (por último!)
kubectl apply -f api-all-in-one.yaml
```

## Troubleshooting

### Ver logs do Vault
```bash
kubectl logs -n api-app-go -l app=vault
```
### Ver status do External Secrets Operator
```bash
kubectl logs -n api-app-go -l app.kubernetes.io/name=external-secrets
```ectl logs -n external-secrets-system -l app.kubernetes.io/name=external-secrets
```

### ExternalSecret não está sincronizando?
```bash
# Ver detalhes do ExternalSecret
kubectl describe externalsecret mysql-credentials -n api-app-go

# Ver eventos
kubectl get events -n api-app-go --sort-by='.lastTimestamp'
```

### Secret não foi criado?
```bash
# Verificar se o SecretStore está válido
kubectl get secretstore vault-backend -n api-app-go -o yaml

# Testar conexão com o Vault manualmente
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -H "X-Vault-Token: root" http://vault.api-app-go:8200/v1/secret/data/inventory-app/database
```ploy do Vault e namespace
kubectl apply -f vault-all-in-one.yaml

# 2. Configurar Vault (passos 2-5 do guia)
# ... (port-forward, criar secrets, policies, roles)

# 3. Deploy MySQL
kubectl apply -f mysql-deployment.yaml

# 4. Deploy OTEL (opcional)
kubectl apply -f otel-collector-deployment.yaml
kubectl apply -f otel-collector-service.yaml

# 5. Deploy da API (por último!)
kubectl apply -f api-all-in-one.yaml
```

## Troubleshooting

### Ver logs do Vault
```bash
kubectl logs -n api-app-go -l app=vault
```

### Ver logs do Agent Injector
```bash
kubectl logs -n api-app-go -l app=vault-agent-injector
```

# 4. Deploy OTEL (opcional)
kubectl apply -f otel-collector-deployment.yaml
kubectl apply -f otel-collector-service.yaml

# 5. Deploy da API (por último!)
kubectl apply -f api-all-in-one.yaml
```ectl logs -n api-app-go -l app=inventory-app -c vault-agent-init
kubectl logs -n api-app-go -l app=inventory-app -c vault-agent
```

### Pod não está criando com 2 containers?
```bash
# Verificar se o MutatingWebhook do Vault está configurado
kubectl get mutatingwebhookconfigurations

# Verificar annotations no pod
kubectl describe pod -n api-app-go -l app=inventory-app | grep -A 5 Annotations
```

### Erro de autenticação no Vault?
```bash
# Verificar se o role foi criado corretamente
vault read auth/kubernetes/role/inventory-app

# Verificar se o ServiceAccount existe
kubectl get sa inventory-app -n api-app-go
```

### Ver logs do Agent Injector
```bash
kubectl logs -n vault -l app=vault-agent-injector
```

## Notas Importantes

- **DEV MODE**: Esta configuração usa Vault em modo desenvolvimento (dados não persistem em restart)
- **External Secrets**: Secrets são sincronizadas do Vault para Kubernetes Secrets nativos
- **Compatibilidade**: Funciona com imagens scratch (não precisa de shell)
- **PRODUÇÃO**: Para produção:
  - Use Vault com backend persistente (Consul, etcd, etc)
  - Configure TLS para todas as comunicações
  - Use autenticação Kubernetes ao invés de token root
  - Configure múltiplas réplicas do Vault para HA
  - Configure auto-unseal com Cloud KMS
  - Separe Vault em namespace dedicado
