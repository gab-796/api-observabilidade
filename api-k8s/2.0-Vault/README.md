# Vault Integration - Gerenciamento de Secrets

Esta pasta contém os manifestos Kubernetes para integrar HashiCorp Vault ao gerenciamento de secrets da aplicação.

## Arquivos a Copiar da Pasta api-k8s

Copie os seguintes arquivos **SEM MODIFICAÇÃO** da pasta `api-k8s`:

```bash
# Services
cp api-k8s/api-service.yaml 2.0-Vault/
cp api-k8s/mysql-service.yaml 2.0-Vault/ # se existir

# MySQL (Database)
cp api-k8s/mysql-deployment.yaml 2.0-Vault/
cp api-k8s/mysql-pvc.yaml 2.0-Vault/ # se existir
cp api-k8s/mysql-pv.yaml 2.0-Vault/ # se existir

# Namespace da aplicação
cp api-k8s/api-namespace.yaml 2.0-Vault/ # se existir

# Ingress/Network
cp api-k8s/api-ingress.yaml 2.0-Vault/ # se existir
cp api-k8s/network-policy.yaml 2.0-Vault/ # se existir

# OTEL Collector
cp api-k8s/otel-collector-*.yaml 2.0-Vault/ # se existir

# HPA/Resource configs
cp api-k8s/api-hpa.yaml 2.0-Vault/ # se existir
```

## Arquivos Específicos do Vault (já criados)

Estes arquivos são **NOVOS** e específicos para a integração com Vault:

- ✅ `vault-namespace.yaml` - Namespace dedicado para o Vault
- ✅ `vault-configmap.yaml` - Configuração do servidor Vault
- ✅ `vault-serviceaccount.yaml` - ServiceAccount e permissões do Vault
- ✅ `vault-service.yaml` - Service para expor o Vault
- ✅ `vault-statefulset.yaml` - Deploy do Vault como StatefulSet
- ✅ `vault-agent-injector.yaml` - Sidecar injector para injetar secrets nos pods
- ✅ `app-serviceaccount.yaml` - ServiceAccount para a aplicação
- ✅ `SETUP.md` - Guia completo de configuração

## Arquivos Modificados (não copiar, usar versão nova)

Estes arquivos foram **MODIFICADOS** para integração com Vault:

- ⚠️ `api-configmap.yaml` - Removidas secrets (MYSQL_ROOT_PASSWORD)
- ⚠️ `api-deployment.yaml` - Adicionadas annotations do Vault + serviceAccountName

## Arquivos que NÃO devem ser copiados

- ❌ `api-secret.yaml` - Substituído pelo Vault
- ❌ Qualquer arquivo com secrets/passwords em texto plano

## Estrutura de Arquivos

- `vault-namespace.yaml` - Namespace dedicado para o Vault
- `vault-configmap.yaml` - Configuração do servidor Vault
- `vault-serviceaccount.yaml` - ServiceAccount e permissões do Vault
- `vault-service.yaml` - Service para expor o Vault
- `vault-statefulset.yaml` - Deploy do Vault como StatefulSet
- `vault-agent-injector.yaml` - Sidecar injector para injetar secrets nos pods
- `app-serviceaccount.yaml` - ServiceAccount para a aplicação
- `SETUP.md` - Guia completo de configuração

## Quick Start

```bash
# 1. Deploy Vault
kubectl apply -f vault-namespace.yaml
kubectl apply -f vault-serviceaccount.yaml
kubectl apply -f vault-configmap.yaml
kubectl apply -f vault-service.yaml
kubectl apply -f vault-statefulset.yaml
kubectl apply -f vault-agent-injector.yaml

# 2. Configurar Vault (seguir SETUP.md)

# 3. Deploy aplicação
kubectl apply -f app-serviceaccount.yaml
kubectl apply -f api-configmap.yaml
kubectl apply -f api-deployment.yaml
```

## Benefícios

- ✅ Secrets centralizados e auditados
- ✅ Rotação automática de credenciais
- ✅ Controle de acesso granular
- ✅ Secrets nunca armazenadas em Git
- ✅ Injeção automática via sidecar

## Referências

- [Vault Kubernetes Documentation](https://developer.hashicorp.com/vault/docs/platform/k8s)
- [Vault Agent Injector](https://developer.hashicorp.com/vault/docs/platform/k8s/injector)
