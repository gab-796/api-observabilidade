# K6 Load Testing - Inventory API

Este diretório contém testes de carga para a API de inventário usando K6.

## 📋 Pré-requisitos

### 1. Instalar K6

**MacOS:**
```bash
brew install k6
```

**Linux (Debian/Ubuntu):**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Docker:**
```bash
docker pull grafana/k6
```

### 2. Verificar instalação

```bash
k6 version
```

## 🚀 Formas de Executar

### 1. **Localmente (Recomendado para desenvolvimento)**

```bash
# Executar teste básico (usa configuração padrão do script)
k6 run inventory-load-test.js

# Executar com URL customizada
k6 run -e BASE_URL=http://localhost:10000 inventory-load-test.js

# Executar direto (ignora configuração do script - útil para debug)
k6 run --vus 1 --duration 30s inventory-load-test.js

# Ver resultados em tempo real com UI web
k6 run --out web-dashboard inventory-load-test.js
```

### 2. **Docker (Ambiente isolado)**

```bash
# Executar com Docker
docker run --rm -i \
  -e BASE_URL=http://inventory.local \
  --network host \
  grafana/k6 run - < inventory-load-test.js

# Com volume (para salvar resultados)
docker run --rm \
  -v $(pwd):/k6 \
  -e BASE_URL=http://inventory.local \
  --network host \
  grafana/k6 run /k6/inventory-load-test.js
```

### 3. **Kubernetes (Teste de carga real)**

#### 3.1. Criar ConfigMap com o script

```bash
kubectl create configmap k6-test-script \
  --from-file=inventory-load-test.js \
  -n api-app-go
```

#### 3.2. Criar Job K6

```bash
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: k6-load-test
  namespace: api-app-go
spec:
  backoffLimit: 0
  activeDeadlineSeconds: 300
  ttlSecondsAfterFinished: 120
  template:
    metadata:
      labels:
        app: k6-load-test
    spec:
      restartPolicy: Never
      containers:
      - name: k6
        image: grafana/k6:latest
        command:
        - k6
        - run
        - /scripts/inventory-load-test.js
        env:
        - name: BASE_URL
          value: "http://inventory-service:10000"
        - name: K6_OUT
          value: "json=/tmp/results.json"
        volumeMounts:
        - name: k6-script
          mountPath: /scripts
      volumes:
      - name: k6-script
        configMap:
          name: k6-test-script
EOF
```

#### 3.3. Monitorar execução

```bash
# Ver logs em tempo real
kubectl logs -f job/k6-load-test -n api-app-go

# Ver status
kubectl get job k6-load-test -n api-app-go

# Deletar após teste
kubectl delete job k6-load-test -n api-app-go
```

## 📊 Configurações de Carga

O script está configurado como **teste leve** por padrão. Você pode ajustar os cenários:

### Cenário Atual (Light Load)

```javascript
stages: [
  { duration: '30s', target: 5 },   // 5 req/s
  { duration: '60s', target: 10 },  // 10 req/s
  { duration: '60s', target: 10 },  // mantém 10 req/s
  { duration: '30s', target: 0 },   // ramp down
]
```

**Quando usar:**
- ✅ Desenvolvimento local
- ✅ Clusters pequenos (Kind, Minikube)
- ✅ Validação funcional
- ✅ Primeiros testes de observabilidade

### Cenário Moderado (50 req/s)

Edite o arquivo e ajuste:

```javascript
stages: [
  { duration: '1m', target: 20 },   // ramp up para 20 req/s
  { duration: '3m', target: 50 },   // ramp up para 50 req/s
  { duration: '5m', target: 50 },   // mantém 50 req/s
  { duration: '1m', target: 0 },    // ramp down
]
maxVUs: 30,
```

**Quando usar:**
- ✅ Testes de stress moderados
- ✅ Validação de HPA (Horizontal Pod Autoscaler)
- ✅ Clusters com recursos médios

### Cenário Heavy (200+ req/s)

```javascript
stages: [
  { duration: '2m', target: 50 },    // warm up
  { duration: '3m', target: 100 },   // ramp up
  { duration: '5m', target: 200 },   // carga alta
  { duration: '10m', target: 200 },  // sustentação
  { duration: '2m', target: 0 },     // ramp down
]
maxVUs: 100,
```

**Quando usar:**
- ✅ Clusters produção-like
- ✅ Testes de capacidade
- ✅ Identificação de breaking points
- ⚠️ Requer recursos significativos

## 🎯 Melhores Práticas

### 1. **Ordem de Execução Recomendada**

```bash
# Passo 1: Smoke test (validação rápida)
k6 run --vus 1 --duration 30s inventory-load-test.js

# Passo 2: Load test leve (padrão do script)
k6 run inventory-load-test.js

# Passo 3: Análise no Grafana
# Acesse http://grafana-web.local e veja métricas

# Passo 4: Ajuste recursos e repita
kubectl get hpa -n api-app-go -w  # Monitorar autoscaling
```

### 2. **Monitoramento Durante Testes**

**Terminal 1 - Executar K6:**
```bash
k6 run inventory-load-test.js
```

**Terminal 2 - Monitorar pods:**
```bash
watch -n 2 'kubectl get pods -n api-app-go -o wide'
```

**Terminal 3 - Monitorar HPA:**
```bash
kubectl get hpa -n api-app-go -w
```

**Browser - Grafana:**
```bash
# Acesse: http://grafana-web.local
# Dashboard: API Inventory - Observability Dashboard
```

### 3. **Exportar Resultados**

```bash
# JSON (análise posterior)
k6 run --out json=results.json inventory-load-test.js

# CSV (planilhas)
k6 run --out csv=results.csv inventory-load-test.js

# InfluxDB (séries temporais)
k6 run --out influxdb=http://influxdb:8086/k6 inventory-load-test.js

# Grafana Cloud (recomendado)
k6 run --out cloud inventory-load-test.js
```

### 4. **Integração CI/CD**

```yaml
# .github/workflows/load-test.yml
name: Load Test

on:
  schedule:
    - cron: '0 2 * * *'  # Diário às 2am
  workflow_dispatch:

jobs:
  k6-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Run K6 test
        uses: grafana/k6-action@v0.3.1
        with:
          filename: helm-chart/k6/inventory-load-test.js
        env:
          BASE_URL: https://staging.example.com
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: k6-results
          path: summary.json
```

## 📈 Interpretando Resultados

### Métricas Principais

```
✓ http_req_duration.............: avg=125ms  p(95)=345ms  p(99)=567ms
✓ http_req_failed...............: 0.12%      (24 failed / 20000 total)
✓ http_reqs.....................: 20000      111.11/s
✓ iteration_duration............: avg=1.5s   med=1.4s     max=3.2s
✓ vus...........................: 10         min=2        max=10
```

**O que observar:**

| Métrica | Bom | Atenção | Crítico |
|---------|-----|---------|---------|
| `p(95) http_req_duration` | <500ms | 500-1000ms | >1000ms |
| `http_req_failed` | <1% | 1-5% | >5% |
| `iteration_duration` | <2s | 2-5s | >5s |
| Thresholds passed | 100% | 90-99% | <90% |

### Thresholds Configurados

```javascript
thresholds: {
  http_req_duration: ['p(95)<800', 'p(99)<1500'],  // Latência
  http_req_failed: ['rate<0.05'],                  // Taxa de erro <5%
}
```

Se algum threshold falhar, K6 retorna **exit code 1** (útil para CI/CD).

## 🔍 Troubleshooting

### Erro: Connection refused

```bash
# Problema: API não acessível
# Solução 1: Port-forward
kubectl port-forward svc/inventory-service -n api-app-go 10000:10000

# Solução 2: Usar Ingress
# Certifique-se que /etc/hosts tem: 127.0.0.1 inventory.local
```

### Erro: Too many open files

```bash
# Aumentar limites (Linux/Mac)
ulimit -n 10000

# Ou reduzir VUs no teste
# maxVUs: 10 → maxVUs: 5
```

### Teste muito lento

```bash
# Verificar se API está saudável
kubectl get pods -n api-app-go
kubectl logs -n api-app-go -l app=inventory-app --tail=50

# Verificar recursos
kubectl top pods -n api-app-go
```

### Métricas não aparecem no Grafana

```bash
# Verificar se Alloy/OTEL está coletando
kubectl logs -n api-app-go -l app.kubernetes.io/name=alloy

# Verificar se Mimir está recebendo dados
kubectl logs -n api-app-go -l app=mimir
```

## 🎓 Casos de Uso

### 1. Validar Observabilidade

```bash
# Objetivo: Verificar se traces/logs/métricas aparecem no Grafana
k6 run --vus 2 --duration 1m inventory-load-test.js

# Verifique no Grafana:
# - Tempo real de requests
# - Traces no Tempo
# - Logs no Loki
# - Métricas no Mimir
```

### 2. Teste de Autoscaling (HPA)

```bash
# Edite o HPA para trigger mais baixo (teste)
kubectl edit hpa inventory-app-hpa -n api-app-go
# Mude: targetCPUUtilizationPercentage: 50 → 30

# Execute teste moderado
k6 run inventory-load-test.js

# Monitore escalonamento
kubectl get hpa -n api-app-go -w
```

### 3. Identificar Breaking Point

```bash
# Aumente progressivamente a carga
# Edite stages: target: 10 → 50 → 100 → 200
# Continue até:
# - Taxa de erro > 5%
# - p(95) > 2s
# - Pods crashando
```

## 📚 Recursos Adicionais

- [K6 Documentation](https://k6.io/docs/)
- [K6 Examples](https://github.com/grafana/k6-learn)
- [K6 Extensions](https://k6.io/docs/extensions/)
- [Grafana Cloud K6](https://grafana.com/products/cloud/k6/)

## 🔗 Arquivos Relacionados

- [application.yaml](../argocd/application.yaml) - ArgoCD config
- [values.yaml](../pdi-gabriel/values.yaml) - HPA settings
- [README.md](../argocd/README.md) - Deployment guide
