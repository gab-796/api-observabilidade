#!/bin/bash

# Script para deploy em produção
# Busca senhas do Vault de produção antes do deploy

set -e

echo "🔐 Buscando credenciais do Vault de produção..."

# 1. Autenticar no Vault de produção
export VAULT_ADDR="https://vault.empresa.com"
vault login -method=ldap username=seu-usuario

# 2. Buscar senhas reais
export MYSQL_ROOT_PASSWORD=$(vault kv get -field=MYSQL_ROOT_PASSWORD secret/prod/inventory-app/database)
export DB_PASSWORD=$(vault kv get -field=DB_PASSWORD secret/prod/inventory-app/database)

echo "✅ Credenciais obtidas!"

# 3. Deploy (usando as variáveis de ambiente)
./deploy.sh
