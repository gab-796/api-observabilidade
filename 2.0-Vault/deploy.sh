#!/bin/bash

set -e

echo "🚀 Deploy da Stack de Observabilidade + API"
echo ""

# Cores para output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Função para log com timestamp
log() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')]${NC} ⚠️  $1"
}

error() {
    echo -e "${RED}[$(date +'%H:%M:%S')]${NC} ❌ $1"
}

skip() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} ⏭️  $1"
}

# Função para verificar se um pod está rodando
check_pod_running() {
    local label=$1
    local namespace=${2:-api-app-go}
    kubectl get pod -l "$label" -n "$namespace" --field-selector=status.phase=Running --no-headers 2>/dev/null | grep -q .
}

# 1. Criar namespace
log "📦 Criando namespace api-app-go..."
kubectl create namespace api-app-go --dry-run=client -o yaml | kubectl apply -f -

# 2. Deploy Stack de Observabilidade (ANTES da aplicação)
log "📊 Deployando Stack de Observabilidade..."

# Mimir
if check_pod_running "app=mimir"; then
    skip "Mimir já está rodando"
else
    log "  └─ Deployando Mimir (métricas)..."
    kubectl apply -f observabilidade/mimir.yaml
    sleep 5
fi

# Loki
if check_pod_running "app=loki"; then
    skip "Loki já está rodando"
else
    log "  └─ Deployando Loki (logs)..."
    kubectl apply -f observabilidade/loki.yaml
    sleep 5
fi

# Tempo
if check_pod_running "app=tempo"; then
    skip "Tempo já está rodando"
else
    log "  └─ Deployando Tempo (traces)..."
    kubectl apply -f observabilidade/tempo.yaml
    sleep 5
fi

# Pyroscope
if check_pod_running "app=pyroscope"; then
    skip "Pyroscope já está rodando"
else
    log "  └─ Deployando Pyroscope (profiling)..."
    kubectl apply -f observabilidade/pyroscope.yaml
    sleep 5
fi

# OTEL Collector
if check_pod_running "app=otel-collector"; then
    skip "OTEL Collector já está rodando"
else
    log "  └─ Deployando OTEL Collector (coleta traces/métricas)..."
    kubectl apply -f observabilidade/otel-collector.yaml
    sleep 5
fi

# Alloy
if check_pod_running "app=alloy"; then
    skip "Alloy já está rodando"
else
    log "  └─ Deployando Alloy (coleta de logs)..."
    kubectl apply -f observabilidade/alloy.yaml
    sleep 5
fi

# Grafana DataSources (sempre aplicar ConfigMap)
log "  └─ Deployando Grafana DataSources..."
kubectl apply -f observabilidade/grafana-datasources-configmap.yaml
sleep 3

# Grafana
if check_pod_running "app=grafana"; then
    skip "Grafana já está rodando"
else
    log "  └─ Deployando Grafana (visualização)..."
    kubectl apply -f observabilidade/grafana.yaml
fi

# Ingress (sempre aplicar)
log "  └─ Deployando Ingress..."
kubectl apply -f observabilidade/ingress-grafana-stack.yaml

# Aguardar pods de observabilidade estarem prontos
log "⏳ Aguardando stack de observabilidade ficar pronta..."
kubectl wait --for=condition=ready pod -l app=mimir -n api-app-go --timeout=120s || warn "Mimir demorou para ficar pronto"
kubectl wait --for=condition=ready pod -l app=loki -n api-app-go --timeout=120s || warn "Loki demorou para ficar pronto"
kubectl wait --for=condition=ready pod -l app=tempo -n api-app-go --timeout=120s || warn "Tempo demorou para ficar pronto"
kubectl wait --for=condition=ready pod -l app=grafana -n api-app-go --timeout=120s || warn "Grafana demorou para ficar pronto"

log "✅ Stack de observabilidade deployada!"
echo ""

# 3. Deploy Vault
if check_pod_running "app=vault"; then
    skip "Vault já está rodando"
else
    log "🔐 Deployando Vault..."
    kubectl apply -f vault.yaml
    kubectl wait --for=condition=ready pod -l app=vault -n api-app-go --timeout=120s
fi

# 3.5. Configurar Vault (criar secrets) - sempre executar para garantir que secrets existem
# ⚠️ ATENÇÃO: Em produção, você criaria secrets no Vault MANUALMENTE antes do deploy
# Este passo é automatizado apenas para facilitar o lab/desenvolvimento
log "🔧 Configurando Vault (criar secrets)..."
VAULT_POD=$(kubectl get pod -n api-app-go -l app=vault -o jsonpath='{.items[0].metadata.name}')

# Usar variáveis de ambiente ou valores padrão (apenas para lab)
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-admin}"
DB_PASSWORD="${DB_PASSWORD:-admin}"

warn "🔒 Criando secrets no Vault (apenas para lab - em produção, faça isso manualmente!)"

kubectl exec -n api-app-go $VAULT_POD -- sh -c "
  export VAULT_ADDR='http://127.0.0.1:8200'
  export VAULT_TOKEN='root'
  vault kv put secret/inventory-app/database MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD} DB_PASSWORD=${DB_PASSWORD} 2>/dev/null || true
" && log "✅ Secrets criados/verificados no Vault!" || warn "Falha ao criar secrets no Vault"

# 4. External Secrets (sempre aplicar)
log "🔑 Deployando External Secrets..."
kubectl apply -f external-secrets.yaml
sleep 5

# Verificar se ExternalSecret sincronizou
log "⏳ Aguardando ExternalSecret sincronizar..."
for i in {1..30}; do
    if kubectl get secret mysql-secrets -n api-app-go &>/dev/null; then
        log "✅ Secret mysql-secrets criado com sucesso!"
        break
    fi
    if [ $i -eq 30 ]; then
        warn "Secret mysql-secrets não foi criado. Verifique o ExternalSecret!"
    fi
    sleep 2
done

# 5. MySQL
if check_pod_running "app=mysql"; then
    skip "MySQL já está rodando"
else
    log "🗄️  Deployando MySQL..."
    kubectl apply -f mysql.yaml
    kubectl wait --for=condition=ready pod -l app=mysql -n api-app-go --timeout=120s
fi

# 6. API (por último, depende de tudo)
if check_pod_running "app=inventory-app"; then
    skip "API já está rodando"
else
    log "🌐 Deployando API..."
    kubectl apply -f api.yaml
fi

# 7. Ingress da API (sempre aplicar)
log "🔌 Deployando Ingress da API..."
kubectl apply -f ingress.yaml

echo ""
log "✅ Deploy completo!"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Informações úteis:"
echo ""
echo "🌐 Acessar Grafana:"
echo "   http://grafana-web.local"
echo "   User: admin | Password: admin"
echo ""
echo "⚠️  Adicione ao /etc/hosts:"
echo "   echo \"127.0.0.1 grafana-web.local\" | sudo tee -a /etc/hosts"
echo ""
echo "📊 Verificar pods:"
echo "   kubectl get pods -n api-app-go"
echo ""
echo "📝 Ver logs da API:"
echo "   kubectl logs -n api-app-go -l app=inventory-app -f"
echo ""
echo "🔍 Ver logs do Vault:"
echo "   kubectl logs -n api-app-go -l app=vault -f"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"