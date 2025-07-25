# Integração Pyroscope + OpenTelemetry - Guia de Implementação

## 📋 Resumo da Implementação

A integração entre Pyroscope e OpenTelemetry foi implementada para correlacionar profiling de performance com traces distribuídos, permitindo análise profunda de gargalos de performance com contexto completo de traces.

## 🔧 Componentes Implementados

### 1. Arquivo de Configuração (.env.profiling)
```bash
# Controle granular do profiling
PYROSCOPE_UPLOAD_RATE=10
CPU_THRESHOLD_FOR_PROFILING=20
MEMORY_THRESHOLD_FOR_PROFILING=50
ENABLE_ENDPOINT_PROFILING=true
ENABLE_DATABASE_PROFILING=true
ENABLE_DETAILED_STACK_TRACES=true
INCLUDE_RUNTIME_INFO=true
ENABLE_GOROUTINE_PROFILING=true
ENABLE_DYNAMIC_TAGS=true
ENABLE_CONDITIONAL_PROFILING=true
ENABLE_USER_AGENT_PROFILING=true
```

### 2. Funções de Profiling Contextual (main.go)

#### ProfiledHTTPHandler
- Wraps HTTP handlers para correlacionar profiling com trace information
- Adiciona trace_id e span_id como tags do profiling
- Categoriza user-agent para análise granular
- Tags dinâmicas: handler, method, path, user_agent, trace_id, span_id

#### ProfiledDatabaseOperation
- Wraps operações de banco de dados com profiling contextual
- Correlaciona queries SQL com traces
- Tags específicas: db_operation, component, product_id, trace_id, span_id

#### Funções Auxiliares
- `categorizeUserAgent()`: Categoriza user agents (curl, postman, browser, etc.)
- `ProfileRuntime()`: Monitora métricas de runtime Go correlacionadas com traces

### 3. Atualizações nos Handlers HTTP (app.go)
```go
// Antes
app.Router.HandleFunc("/products", app.getProducts).Methods("GET")

// Depois - com profiling contextual
app.Router.HandleFunc("/products", ProfiledHTTPHandler("get_products", app.getProducts)).Methods("GET")
```

### 4. Atualizações nas Funções de Banco (module.go)
```go
// Antes
func (p *product) getProduct(ctx context.Context, db *sql.DB) error {
    // lógica direta
}

// Depois - com profiling contextual
func (p *product) getProduct(ctx context.Context, db *sql.DB) error {
    return ProfiledDatabaseOperation(ctx, "get_product", p.ID, func(profileCtx context.Context) error {
        // lógica original com profileCtx
    })
}
```

## 🎯 Benefícios da Integração

### 1. Correlação Completa
- **Trace ID + Span ID** em flamegraphs do Pyroscope
- **Contexto de requisição** em profiles de CPU/memória
- **Operações de banco** correlacionadas com traces

### 2. Análise Granular
- Profiling por endpoint específico
- Profiling por operação de banco (CRUD)
- Categorização por tipo de cliente (curl, browser, etc.)
- Tags dinâmicas baseadas em contexto

### 3. Debugging Avançado
- Identificar gargalos específicos por trace
- Analisar performance de queries SQL específicas
- Correlacionar high CPU usage com traces específicos
- Flamegraphs contextuais por operação

## 🔍 Como Validar a Integração

### 1. Verificar Logs de Inicialização
```bash
# Verificar se Pyroscope iniciou
docker logs inventory-app | grep "Pyroscope profiling iniciado"
```

### 2. Testar Endpoints com Profiling
```bash
# Fazer algumas requisições para gerar dados
curl -X GET http://localhost:8080/products
curl -X POST http://localhost:8080/product -d '{"name":"Test","price":10,"quantity":5}'
curl -X GET http://localhost:8080/product/1
```

### 3. Verificar no Pyroscope UI
1. Acesse: `http://localhost:4040`
2. Procure por profiles com tags:
   - `trace_id`: IDs de traces específicos
   - `span_id`: IDs de spans específicos
   - `handler`: nome do handler (get_products, create_product, etc.)
   - `db_operation`: operação de banco (get_product, create_product, etc.)
   - `user_agent`: tipo de cliente (curl, browser, etc.)

### 4. Verificar Correlação Traces + Profiles
1. **No Grafana Tempo**: Copie um trace_id
2. **No Pyroscope**: Filtre profiles por esse trace_id
3. **Resultado**: Flamegraph específico para aquele trace

### 5. Verificar Tags Dinâmicas
No Pyroscope UI, verifique se há tags disponíveis:
- `service=inventory-app`
- `handler=get_products|create_product|etc`
- `db_operation=get_product|create_product|etc`
- `trace_id=<hex_trace_id>`
- `span_id=<hex_span_id>`
- `user_agent=curl|browser|postman|etc`

## 📊 Casos de Uso Avançados

### 1. Debugging de Latência por Trace
```
1. Identifique trace lento no Tempo/Jaeger
2. Copie o trace_id
3. No Pyroscope, filtre: trace_id=<id>
4. Analise flamegraph específico desse trace
```

### 2. Análise de Performance por Endpoint
```
1. No Pyroscope, filtre: handler=get_products
2. Compare com handler=create_product
3. Identifique endpoints mais custosos
```

### 3. Profiling de Queries SQL Específicas
```
1. No Pyroscope, filtre: db_operation=get_product
2. Analise performance de queries específicas
3. Correlacione com traces de operações lentas
```

### 4. Análise por Tipo de Cliente
```
1. Filtre: user_agent=curl (testes automatizados)
2. Compare com user_agent=browser (usuários reais)
3. Identifique diferenças de performance
```

## 🚀 Melhorias Futuras

### 1. Profiling Condicional
- Ativar profiling apenas em high CPU/memory
- Thresholds configuráveis via environment variables

### 2. Alertas Automáticos
- Alertas quando flamegraphs excedem thresholds
- Correlação automática com traces problemáticos

### 3. Dashboards Customizados
- Dashboard Grafana combinando métricas + traces + profiles
- Visualização correlacionada de performance

### 4. Sampling Inteligente
- Profiling mais frequente para traces lentos
- Sampling adaptativo baseado em carga

## 📝 Arquivos Modificados

1. **`.env.profiling`**: Configuração avançada do profiling
2. **`main.go`**: Funções de profiling contextual
3. **`app.go`**: Handlers wrappados com profiling
4. **`module.go`**: Funções de banco com profiling contextual

## ✅ Status de Implementação

- [x] Configuração do Pyroscope com OpenTelemetry
- [x] Profiling contextual em HTTP handlers
- [x] Profiling contextual em operações de banco
- [x] Tags dinâmicas com trace_id/span_id
- [x] Categorização de user-agent
- [x] Funções auxiliares de profiling
- [x] Validação de integração

A implementação está completa e pronta para uso em produção com observabilidade avançada!
