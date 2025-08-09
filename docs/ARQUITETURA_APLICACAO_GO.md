# Arquitetura da Aplicação Go - My Inventory API

## Visão Geral

Esta aplicação é uma **API REST para gerenciamento de inventário de produtos** construída em Go, com foco em **observabilidade completa**. A aplicação demonstra como implementar uma stack completa de observabilidade incluindo métricas, logs, traces e profiling contínuo.

## Estrutura da Aplicação

A aplicação está organizada em 3 arquivos principais:

### 📁 Estrutura de Arquivos

```
my-inventory-docker-traces-usando-bibliotecas-v2-pyroscope-mimir/
├── main.go      # Ponto de entrada, configuração de observabilidade e servidores
├── app.go       # Lógica da aplicação HTTP, handlers e middlewares
├── module.go    # Camada de dados e operações do banco de dados
├── go.mod       # Dependências do projeto
└── docker-compose.yml # Orquestração dos serviços
```

---

## 📋 Análise Detalhada dos Arquivos

### 1. `main.go` - Orquestração e Observabilidade

**Responsabilidades:**
- **Configuração da stack de observabilidade completa**
- **Inicialização de múltiplos TracerProviders OpenTelemetry**
- **Configuração de métricas Prometheus personalizadas**
- **Integração com Pyroscope para profiling contínuo**
- **Gerenciamento do ciclo de vida de 3 serviços concorrentes**

#### Componentes Implementados:

##### 🔍 **OpenTelemetry (Tracing) - Arquitetura Multi-Service**
```go
// TRÊS TracerProviders distintos para separação granular
mainTp, err := newTracerProvider(endpoint, "my-inventory-app")     // HTTP traces
metricsTP, err := newTracerProvider(endpoint, "my-inventory-mysql") // DB traces  
sqlTP, err := newTracerProvider(endpoint, "my-inventory-sql")      // SQL traces

// Configuração global vs específica
otel.SetTracerProvider(mainTp)           // Provider global para HTTP
```

**Justificativa:** 
- **3 serviços separados** no Tempo permitem análise granular por camada
- **Provider global** usado pelo `otelhttp.NewHandler` para instrumentação automática
- **Providers específicos** passados para `otelsql` e componentes customizados

##### 📊 **Métricas Prometheus Customizadas**
```go
var (
    // Métricas HTTP (RED)
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total de requisições HTTP",
        },
        []string{"method", "endpoint", "status_code"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "Duração das requisições HTTP",
            Buckets: prometheus.DefBuckets, // [.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10]
        },
        []string{"method", "endpoint"},
    )
    
    // Métricas de Negócio
    productsInDB = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "products_in_database_total",
        Help: "Número atual de produtos no banco de dados",
    })
    
    // Métricas de Infraestrutura
    sqlErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "sql_errors_total", 
        Help: "Total de erros SQL",
    })
)
```

**Justificativa:** 
- **RED metrics** para monitoramento de SLA (Rate, Errors, Duration)
- **Métricas de negócio** para alertas funcionais
- **Labels dinâmicos** para agregação flexível no Grafana

##### 🔥 **Pyroscope (Profiling Contínuo)**
```go
profiler, err := pyroscope.Start(pyroscope.Config{
    ApplicationName: "inventory-app",
    ServerAddress:   "http://pyroscope:4040", // Conecta ao Pyroscope server
    Logger:          pyroscope.StandardLogger,
    ProfileTypes: []pyroscope.ProfileType{
        pyroscope.ProfileCPU,           // CPU profiling
        pyroscope.ProfileAllocObjects,  // Memory allocations
        pyroscope.ProfileAllocSpace,    // Memory space
        pyroscope.ProfileInuseObjects,  // Objects in use
        pyroscope.ProfileInuseSpace,    // Space in use
    },
})
defer profiler.Stop()
```

**Justificativa:** 
- **Profiling contínuo** permite identificar hotspots e vazamentos de memória
- **Overhead mínimo** (< 2% CPU) adequado para produção
- **Múltiplos tipos** de profile para análise completa de performance

##### 📝 **Logging Estruturado com OpenTelemetry**
```go
// Hook global para correlação automática de traces
logrus.AddHook(otellogrus.NewHook(
    otellogrus.WithLevels(logrus.AllLevels...),
))

// Configuração JSON para parsing estruturado
logrus.SetFormatter(&logrus.JSONFormatter{
    TimestampFormat: time.RFC3339,
    FieldMap: logrus.FieldMap{
        logrus.FieldKeyTime:  "timestamp",
        logrus.FieldKeyLevel: "level",
        logrus.FieldKeyMsg:   "message",
    },
})
```

**Justificativa:** 
- **Correlação automática** de trace_id e span_id nos logs
- **Formato JSON** facilita parsing pelo Loki/Elasticsearch
- **Structured logging** permite queries e agregações complexas

#### Arquitetura de Serviços Multi-Port:

```go
var wg sync.WaitGroup

// 1. Servidor de Métricas Prometheus (porta 2113)
wg.Add(1)
go func() {
    defer wg.Done()
    muxMetrics := http.NewServeMux()
    muxMetrics.Handle("/metrics", promhttp.Handler())
    
    logrus.Info("Servidor de métricas iniciando na porta 2113")
    if err := http.ListenAndServe(":2113", muxMetrics); err != nil {
        logrus.WithError(err).Error("Servidor de métricas falhou")
    }
}()

// 2. Aplicação Principal com instrumentação completa (porta 10000)
wg.Add(1)
go func() {
    defer wg.Done()
    logrus.Info("Aplicação principal iniciando na porta 10000")
    
    // otelhttp.NewHandler usa o TracerProvider GLOBAL (mainTp)
    handler := otelhttp.NewHandler(app.Router, "my-inventory-app")
    if err := http.ListenAndServe(":10000", handler); err != nil {
        logrus.WithError(err).Fatal("Servidor da aplicação falhou")
    }
}()

wg.Wait() // Aguarda todos os serviços
```

**Justificativa:**
- **Isolamento de responsabilidades**: Métricas independentes da aplicação
- **Disponibilidade**: Falhas na app não afetam coleta de métricas
- **Segurança**: Endpoint de métricas pode ter ACL diferentes
- **Performance**: Evita contenção de recursos entre serviços

#### Função newTracerProvider - Configuração Detalhada:

```go
func newTracerProvider(endpoint string, serviceName string) (*sdktrace.TracerProvider, error) {
    ctx := context.Background()

    // gRPC connection com configuração otimizada
    conn, err := grpc.NewClient(endpoint,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, fmt.Errorf("falha ao configurar gRPC client para %s: %w", serviceName, err)
    }

    // OTLP exporter com conexão reutilizada
    traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("falha ao criar OTLP exporter para %s: %w", serviceName, err)
    }

    // Resource com metadados do serviço
    res, err := resource.New(ctx,
        resource.WithSchemaURL(semconv.SchemaURL),
        resource.WithAttributes(
            semconv.ServiceNameKey.String(serviceName),
            attribute.String("environment", "local"),
            attribute.String("version", "1.0.0"),
        ),
    )
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("falha ao criar recurso para %s: %w", serviceName, err)
    }

    // TracerProvider com BatchSpanProcessor para performance
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(traceExporter,
            sdktrace.WithMaxExportBatchSize(512),
            sdktrace.WithBatchTimeout(5*time.Second),
        ),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()), // 100% sampling em dev
    )

    return tp, nil
}
```

**Justificativa:**
- **Batch processing** reduz overhead de rede
- **Resource attributes** permitem filtering no Tempo/Jaeger
- **Connection reuse** otimiza performance do gRPC
- **Configuração por serviço** permite tuning específico

---

### 2. `app.go` - Camada de Aplicação HTTP

**Responsabilidades:**
- **Handlers HTTP para operações CRUD completas**
- **Middleware customizado de métricas Prometheus**
- **Inicialização da aplicação com instrumentação SQL**
- **Gestão do ciclo de vida da aplicação e graceful shutdown**
- **Sistema de atualização periódica de métricas de negócio**

#### Estrutura Principal:

```go
type App struct {
    Router *mux.Router  // Gorilla Mux com middleware stack
    DB     *sql.DB      // Conexão MySQL instrumentada com OTEL
}
```

#### Endpoints Implementados (API RESTful):

| Endpoint | Método | Funcionalidade | Validações | Context-Aware |
|----------|--------|---------------|------------|---------------|
| `/products` | GET | Lista todos os produtos | - | ✅ |
| `/product/{id:[0-9]+}` | GET | Busca produto por ID | ID numérico | ✅ |
| `/product` | POST | Cria novo produto | Nome obrigatório, preço/qty ≥ 0 | ✅ |
| `/product/{id:[0-9]+}` | PUT | Atualiza produto existente | ID + validações de POST | ✅ |
| `/product/{id:[0-9]+}` | DELETE | Remove produto | ID numérico | ✅ |
| `/health` | GET | Health check da aplicação | - | ✅ |

#### Middleware Stack (Ordem Crítica):

```go
func (app *App) Initialise(sqlTracerProvider trace.TracerProvider) error {
    app.Router = mux.NewRouter().StrictSlash(true)
    
    // ORDEM FUNDAMENTAL: Tracing ANTES de métricas
    app.Router.Use(otelmux.Middleware("inventory-app")) // 1º - Instrumentação OTEL
    app.Router.Use(prometheusMiddleware)                // 2º - Coleta de métricas
    
    app.HandleRequests()
    go app.startBackgroundProductCountUpdate() // Goroutine para métricas
    
    return nil
}
```

**Justificativa da Ordem:**
1. **`otelmux.Middleware`** deve ser primeiro para capturar TODOS os requests
2. **`prometheusMiddleware`** vem depois para ter contexto de trace disponível
3. **Handlers específicos** executam com contexto completo estabelecido

#### Middleware de Métricas Customizado:

```go
func prometheusMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // ResponseWriter wrapper para capturar status code
        ww := &responseWriterWrapper{ResponseWriter: w, statusCode: 200}
        
        // Executa o handler real
        next.ServeHTTP(ww, r)
        
        // Coleta métricas RED
        method := r.Method
        endpoint := r.URL.Path
        status := strconv.Itoa(ww.statusCode)
        
        // Incrementa counter de requests
        httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
        
        // Registra duração
        duration := time.Since(start).Seconds()
        httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
        
        // Logging contextual com trace correlation
        span := trace.SpanFromContext(r.Context())
        logEntry := logrus.WithContext(r.Context()).WithFields(logrus.Fields{
            "component":    "http_middleware",
            "method":       method,
            "endpoint":     endpoint,
            "status_code":  status,
            "duration_ms":  duration * 1000,
        })
        
        if span.SpanContext().IsValid() {
            logEntry = logEntry.WithFields(logrus.Fields{
                "trace_id": span.SpanContext().TraceID().String(),
                "span_id":  span.SpanContext().SpanID().String(),
            })
        }
        
        if ww.statusCode >= 400 {
            logEntry.Warn("HTTP request completed with error")
        } else {
            logEntry.Debug("HTTP request completed successfully")
        }
    })
}
```

**Justificativa:**
- **Métricas RED completas** para SLA monitoring
- **Trace correlation** nos logs para debugging eficiente
- **Performance mínima** com overhead < 1ms por request

#### Inicialização de Banco com Instrumentação Completa:

```go
func (app *App) Initialise(sqlTracerProvider trace.TracerProvider) error {
    // Configuração de variáveis de ambiente
    dbUser := os.Getenv("DB_USER")
    dbPassword := os.Getenv("DB_PASSWORD") 
    dbName := os.Getenv("DB_NAME")
    dbHost := os.Getenv("DB_HOST")
    
    if dbUser == "" || dbPassword == "" || dbName == "" || dbHost == "" {
        return errors.New("variáveis de ambiente do banco não configuradas")
    }
    
    connectionString := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", 
        dbUser, dbPassword, dbHost, dbName)
    
    // Instrumentação SQL com OTELSQL
    app.DB, err = otelsql.Open("mysql", connectionString,
        otelsql.WithTracerProvider(sqlTracerProvider),    // Provider específico para SQL
        otelsql.WithAttributes(
            semconv.DBSystemMySQL,                        // Sistema de banco
            semconv.DBNameKey.String(dbName),             // Nome do banco
            semconv.NetPeerNameKey.String(dbHost),        // Host do banco
            semconv.NetPeerPortKey.Int(3306),             // Porta do banco
        ),
        otelsql.WithSQLCommenter(true),                   // Adiciona trace context no SQL
    )
    if err != nil {
        return fmt.Errorf("falha ao abrir conexão instrumentada: %w", err)
    }
    
    // Health check inicial com timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err = app.DB.PingContext(ctx)
    if err != nil {
        app.DB.Close()
        return fmt.Errorf("falha no ping inicial do banco: %w", err)
    }
    
    logrus.Infof("Conexão MySQL instrumentada estabelecida: %s@%s", dbName, dbHost)
    return nil
}
```

**Justificativa:**
- **otelsql** instrumenta automaticamente TODAS as queries SQL
- **SQLCommenter** adiciona trace context como comentários nas queries
- **Attributes semconv** padronizam metadados para ferramentas OTEL
- **Provider específico** permite separação de traces SQL no Tempo

#### Handlers com Trace Correlation Completa:

```go
func (app *App) getProducts(w http.ResponseWriter, r *http.Request) {
    // Extrai span atual para correlação nos logs
    span := trace.SpanFromContext(r.Context())
    
    // Log estruturado com trace correlation
    entry := logrus.WithContext(r.Context()).WithFields(logrus.Fields{
        "component": "http_handler",
        "operation": "get_products",
    })
    
    if span.SpanContext().IsValid() {
        entry = entry.WithFields(logrus.Fields{
            "trace_id": span.SpanContext().TraceID().String(),
            "span_id":  span.SpanContext().SpanID().String(),
        })
    }
    entry.Info("Iniciando busca de produtos")
    
    // Chama função de dados passando contexto completo
    products, err := getProductsFromDB(r.Context(), app.DB)
    if err != nil {
        // Log de erro com trace correlation
        logEntry := logrus.WithContext(r.Context()).WithError(err).WithFields(logrus.Fields{
            "component": "http_handler",
            "operation": "get_products",
        })
        if span.SpanContext().IsValid() {
            logEntry = logEntry.WithFields(logrus.Fields{
                "trace_id": span.SpanContext().TraceID().String(),
                "span_id":  span.SpanContext().SpanID().String(),
            })
        }
        logEntry.Error("Erro ao obter produtos do banco")
        
        // Incrementa métrica de erro
        sqlErrorsTotal.Inc()
        sendError(w, r, http.StatusInternalServerError, errors.New("failed to retrieve products"))
        return
    }
    
    // Log de sucesso com informações úteis
    successEntry := logrus.WithContext(r.Context()).WithField("num_products", len(products))
    if span.SpanContext().IsValid() {
        successEntry = successEntry.WithFields(logrus.Fields{
            "trace_id": span.SpanContext().TraceID().String(),
            "span_id":  span.SpanContext().SpanID().String(),
        })
    }
    successEntry.Info("Produtos listados com sucesso")
    
    sendResponse(r.Context(), w, http.StatusOK, products)
}
```

**Justificativa:**
- **Context propagation** permite rastreamento end-to-end
- **Trace correlation** nos logs facilita debugging distribuído
- **Structured logging** permite queries e alertas precisos
- **Métricas de erro** alimentam dashboards de SLA

#### Sistema de Atualização de Métricas de Negócio:

```go
func (app *App) startBackgroundProductCountUpdate() {
    // Métrica inicial
    count, err := app.getCurrentProductCount()
    if err == nil {
        productsInDB.Set(float64(count))
        logrus.Infof("Métrica inicial 'products_in_db' definida: %d", count)
    } else {
        logrus.Warn("Não foi possível definir métrica inicial 'products_in_db'")
    }
    
    // Timer para atualizações periódicas
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    logrus.Info("Iniciando atualização periódica da métrica 'products_in_db' (5min)")
    
    for range ticker.C {
        count, err := app.getCurrentProductCount()
        if err == nil {
            productsInDB.Set(float64(count))
            logrus.Debugf("Métrica 'products_in_db' atualizada: %d", count)
        } else {
            logrus.Warn("Falha ao atualizar métrica 'products_in_db'")
        }
    }
}

func (app *App) getCurrentProductCount() (int, error) {
    // Context com timeout para operação interna
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    
    // Chama função instrumentada de contagem
    count, err := countProducts(ctx, app.DB)
    if err != nil {
        logrus.WithError(err).Error("Erro ao contar produtos para métrica")
        sqlErrorsTotal.Inc() // Incrementa métrica de erro
        return 0, err
    }
    return count, nil
}
```

**Justificativa:**
- **Goroutine dedicada** não bloqueia requests HTTP
- **Métrica de negócio** permite alertas funcionais
- **Context com timeout** evita operações infinitas
- **Error handling** mantém a aplicação resiliente

---

### 3. `module.go` - Camada de Dados com Instrumentação Completa

**Responsabilidades:**
- **Definição da estrutura de dados do domínio (`product`)**
- **Operações CRUD completas com context propagation**
- **Logging estruturado de todas as operações de banco**
- **Tratamento robusto de erros com error wrapping**
- **Funções auxiliares para métricas de negócio**

#### Estrutura de Dados:

```go
type product struct {
    ID       int     `json:"id"`
    Name     string  `json:"name"`
    Quantity int     `json:"quantity"`
    Price    float64 `json:"price"`
}
```

#### Operações Implementadas (100% Context-Aware):

| Função | Responsabilidade | Contexto | Error Handling | Logging |
|--------|------------------|----------|----------------|---------|
| `getProductsFromDB(ctx, db)` | Lista todos os produtos | ✅ | Error wrapping | Debug estruturado |
| `(p *product) getProduct(ctx, db)` | Busca produto por ID | ✅ | sql.ErrNoRows detection | Warn para NotFound |
| `(p *product) createProduct(ctx, db)` | Cria novo produto | ✅ | LastInsertId handling | Debug com product_id |
| `(p *product) updateProduct(ctx, db)` | Atualiza produto | ✅ | RowsAffected validation | Debug + Warn |
| `(p *product) deleteProduct(ctx, db)` | Remove produto | ✅ | RowsAffected validation | Debug + Warn |
| `countProducts(ctx, db)` | Conta produtos (métrica) | ✅ | Query error handling | Error on failure |

#### Implementação Detalhada com Context Propagation:

##### getProductsFromDB - Listagem com Instrumentação:
```go
func getProductsFromDB(ctx context.Context, db *sql.DB) ([]product, error) {
    logrus.WithContext(ctx).WithFields(logrus.Fields{
        "component": "database",
        "operation": "get_products",
    }).Debug("Iniciando getProductsFromDB")

    query := "SELECT id, name, quantity, price FROM products"
    
    // QueryContext propaga contexto e permite cancelamento
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component": "database",
            "operation": "get_products",
            "error":     err.Error(),
        }).Error("Erro ao executar QueryContext")
        
        // Error wrapping preserva stack trace
        return nil, fmt.Errorf("erro ao buscar produtos: %w", err)
    }
    defer rows.Close()

    products := []product{}
    for rows.Next() {
        var p product
        err := rows.Scan(&p.ID, &p.Name, &p.Quantity, &p.Price)
        if err != nil {
            logrus.WithContext(ctx).WithFields(logrus.Fields{
                "component": "database",
                "operation": "get_products", 
                "error":     err.Error(),
            }).Error("Erro ao ler dados da linha")
            return nil, fmt.Errorf("erro ao ler produto: %w", err)
        }
        products = append(products, p)
    }

    // Verifica erros de iteração
    if err = rows.Err(); err != nil {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component": "database",
            "operation": "get_products",
            "error":     err.Error(),
        }).Error("Erro durante iteração das linhas")
        return nil, fmt.Errorf("erro ao iterar produtos: %w", err)
    }

    logrus.WithContext(ctx).WithFields(logrus.Fields{
        "component":    "database",
        "operation":    "get_products",
        "num_products": len(products),
    }).Debug("Produtos encontrados com sucesso")
    
    return products, nil
}
```

**Justificativa:**
- **QueryContext** permite cancelamento via context
- **Error wrapping** preserva stack trace completo
- **Structured logging** facilita debugging e monitoring
- **Validation completa** de todas as etapas da query

##### createProduct - Criação com LastInsertId:
```go
func (p *product) createProduct(ctx context.Context, db *sql.DB) error {
    logrus.WithContext(ctx).WithFields(logrus.Fields{
        "component":    "database",
        "operation":    "create_product",
        "product_name": p.Name,
    }).Debug("Iniciando createProduct")
    
    query := "INSERT INTO products(name, quantity, price) VALUES(?,?,?)"
    
    // ExecContext para operações de modificação
    result, err := db.ExecContext(ctx, query, p.Name, p.Quantity, p.Price)
    if err != nil {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component": "database",
            "operation": "create_product",
            "error":     err.Error(),
        }).Error("Erro ao executar ExecContext")
        return fmt.Errorf("erro ao criar produto: %w", err)
    }

    // Obtém ID gerado pelo auto_increment
    id, err := result.LastInsertId()
    if err != nil {
        logrus.WithContext(ctx).WithError(err).Error("Erro ao obter LastInsertId")
        return fmt.Errorf("erro ao obter ID do produto: %w", err)
    }
    
    p.ID = int(id) // Atualiza struct com ID gerado

    logrus.WithContext(ctx).WithFields(logrus.Fields{
        "component":  "database",
        "operation":  "create_product",
        "product_id": p.ID,
    }).Debug("Produto criado com sucesso")
    
    return nil
}
```

##### updateProduct - Atualização com RowsAffected:
```go
func (p *product) updateProduct(ctx context.Context, db *sql.DB) error {
    logrus.WithContext(ctx).WithFields(logrus.Fields{
        "component":  "database",
        "operation":  "update_product",
        "product_id": p.ID,
    }).Debug("Iniciando updateProduct")
    
    query := "UPDATE products SET name =?, quantity =?, price =? WHERE id =?"
    
    result, err := db.ExecContext(ctx, query, p.Name, p.Quantity, p.Price, p.ID)
    if err != nil {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component":  "database",
            "operation":  "update_product",
            "product_id": p.ID,
            "error":      err.Error(),
        }).Error("Erro ao executar ExecContext")
        return fmt.Errorf("erro ao atualizar produto %d: %w", p.ID, err)
    }

    // Verifica se o registro foi realmente atualizado
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component":  "database",
            "operation":  "update_product",
            "product_id": p.ID,
            "error":      err.Error(),
        }).Error("Erro ao obter RowsAffected")
        return fmt.Errorf("erro ao verificar linhas afetadas para produto %d: %w", p.ID, err)
    }
    
    // Se nenhuma linha foi afetada, o produto não existe
    if rowsAffected == 0 {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component":  "database",
            "operation":  "update_product",
            "product_id": p.ID,
        }).Warn("Nenhum produto atualizado - ID não encontrado")
        
        // Retorna sql.ErrNoRows para indicar NotFound
        return sql.ErrNoRows
    }

    logrus.WithContext(ctx).WithFields(logrus.Fields{
        "component":  "database",
        "operation":  "update_product",
        "product_id": p.ID,
    }).Debug("Produto atualizado com sucesso")
    
    return nil
}
```

##### deleteProduct - Deleção com Verificação:
```go
func (p *product) deleteProduct(ctx context.Context, db *sql.DB) error {
    logrus.WithContext(ctx).WithFields(logrus.Fields{
        "component":  "database",
        "operation":  "delete_product",
        "product_id": p.ID,
    }).Debug("Iniciando deleteProduct")
    
    query := "DELETE FROM products WHERE id =?"
    
    result, err := db.ExecContext(ctx, query, p.ID)
    if err != nil {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component":  "database",
            "operation":  "delete_product",
            "product_id": p.ID,
            "error":      err.Error(),
        }).Error("Erro ao executar ExecContext")
        return fmt.Errorf("erro ao excluir produto %d: %w", p.ID, err)
    }

    // Verifica se algum registro foi deletado
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component":  "database",
            "operation":  "delete_product",
            "product_id": p.ID,
            "error":      err.Error(),
        }).Error("Erro ao obter RowsAffected")
        return fmt.Errorf("erro ao verificar linhas afetadas para produto %d: %w", p.ID, err)
    }
    
    if rowsAffected == 0 {
        logrus.WithContext(ctx).WithFields(logrus.Fields{
            "component":  "database",
            "operation":  "delete_product",
            "product_id": p.ID,
        }).Warn("Nenhum produto excluído - ID não encontrado")
        
        return sql.ErrNoRows
    }

    logrus.WithContext(ctx).WithFields(logrus.Fields{
        "component":  "database",
        "operation":  "delete_product",
        "product_id": p.ID,
    }).Debug("Produto excluído com sucesso")
    
    return nil
}
```

##### countProducts - Função para Métricas:
```go
func countProducts(ctx context.Context, db *sql.DB) (int, error) {
    var count int
    query := "SELECT COUNT(*) FROM products"
    
    // QueryRowContext para queries que retornam uma única linha
    err := db.QueryRowContext(ctx, query).Scan(&count)
    if err != nil {
        logrus.WithContext(ctx).WithError(err).Error("Erro ao executar QueryRowContext em countProducts")
        return 0, fmt.Errorf("erro ao contar produtos: %w", err)
    }
    
    return count, nil
}
```

#### Características Importantes da Implementação:

##### 1. **Context Propagation Universal:**
- **Todas as funções** recebem `context.Context` como primeiro parâmetro
- **Cancelamento automático** via context timeout/cancellation
- **Trace correlation** automática através do context
- **Preparação para circuit breakers** e rate limiting

##### 2. **Error Handling Robusto:**
- **Error wrapping** com `fmt.Errorf` preserva stack trace
- **Tipo de erro específico** (`sql.ErrNoRows`) para NotFound scenarios
- **RowsAffected validation** diferencia erro de execução vs. registro não encontrado
- **Logging estruturado** de todos os erros para debugging

##### 3. **Logging Estruturado Completo:**
- **Campos padronizados**: component, operation, product_id
- **Context correlation**: trace_id/span_id automáticos via WithContext
- **Níveis apropriados**: Debug para operações normais, Error para falhas, Warn para NotFound
- **Informações úteis**: Número de registros, IDs específicos

##### 4. **Instrumentação SQL Automática:**
- **otelsql** captura automaticamente todas as queries
- **SQLCommenter** adiciona trace context nas queries
- **Spans automáticos** para cada operação SQL
- **Métricas de performance** coletadas automaticamente

**Justificativa Geral:**
Esta implementação demonstra **best practices** para camada de dados em Go:
- **Context-aware** para cancelamento e tracing
- **Error handling** robusto e informativo
- **Logging estruturado** para observabilidade
- **Instrumentação automática** para performance monitoring
- **Separação de responsabilidades** clara entre handlers e dados

---

## 📦 Análise de Dependências (go.mod)

### Dependências Principais:

#### **Observabilidade Completa:**
```go
// OpenTelemetry - Distributed Tracing
"go.opentelemetry.io/otel" v1.36.0
"go.opentelemetry.io/otel/sdk" v1.36.0  
"go.opentelemetry.io/otel/trace" v1.36.0
"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc" v1.35.0

// Instrumentação Automática
"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux" v0.61.0
"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp" v0.60.0
"github.com/XSAM/otelsql" v0.38.0                    // SQL instrumentation

// Métricas Prometheus  
"github.com/prometheus/client_golang" v1.21.1

// Profiling Contínuo
"github.com/grafana/pyroscope-go" v1.1.2

// Logging Estruturado
"github.com/sirupsen/logrus" v1.9.3
"github.com/uptrace/opentelemetry-go-extra/otellogrus" v0.3.2  // Trace correlation
```

#### **Web Framework e Banco:**
```go
// HTTP Router
"github.com/gorilla/mux" v1.8.1

// MySQL Driver
"github.com/go-sql-driver/mysql" v1.8.1

// gRPC para OTLP
"google.golang.org/grpc" v1.71.1
```

### Justificativas de Escolha:

#### **OpenTelemetry (OTEL) - v1.36.0:**
- **Padrão da indústria** para observabilidade distribuída
- **Vendor-neutral** - funciona com Jaeger, Tempo, SigNoz, etc.
- **Instrumentação automática** reduz boilerplate code
- **Context propagation** nativo para Go
- **Semantic conventions** padronizam metadados

#### **Gorilla Mux - v1.8.1:**
- **Router maduro** e estável para HTTP
- **Path variables** com regex support (`{id:[0-9]+}`)
- **Middleware chain** flexível
- **Instrumentação OTEL** nativa disponível

#### **Prometheus Client - v1.21.1:**
- **Métricas padrão** da indústria (RED metrics)
- **Multiple metric types**: Counter, Gauge, Histogram, Summary
- **Label support** para agregação flexível
- **Performance otimizada** para high-throughput

#### **Pyroscope Go - v1.1.2:**
- **Profiling contínuo** com overhead < 2%
- **Multiple profile types**: CPU, Memory, Goroutine
- **Flame graphs** para análise visual
- **Push model** para centralização

#### **Logrus - v1.9.3:**
- **Structured logging** com JSON output
- **Hook system** para integração com OTEL
- **Context support** para trace correlation
- **Field consistency** para queries estruturadas

### Dependências Indiretas Importantes:

```go
// Compressão para melhor performance de rede
"github.com/klauspost/compress" v1.17.11

// UUID generation para trace IDs
"github.com/google/uuid" v1.6.0

// Backoff para retry logic no OTEL
"github.com/cenkalti/backoff/v4" v4.3.0

// HTTP utils para instrumentação
"github.com/felixge/httpsnoop" v1.0.4
```

### Versionamento Strategy:

#### **Compatibilidade OpenTelemetry:**
- **OTEL SDK v1.36.0** - versão estável mais recente
- **Contrib packages v0.61.0** - compatíveis com SDK v1.36.x
- **OTLP exporters v1.35.0** - compatível com receivers modernos

#### **Go Version Requirements:**
```go
go 1.23.0
toolchain go1.23.8
```

**Justificativa:**
- **Go 1.23** inclui melhorias de performance para HTTP
- **Context improvements** para melhor tracing
- **Garbage collector** otimizado para aplicações instrumentadas

## 🏗️ Justificativas Arquiteturais

### 1. **Separação de Responsabilidades**

**Decisão:** Dividir em 3 arquivos com responsabilidades distintas
- `main.go`: Infraestrutura e observabilidade
- `app.go`: HTTP e middlewares
- `module.go`: Dados e persistência

**Benefícios:**
- Código mais organizado e testável
- Fácil manutenção e evolução
- Responsabilidades bem definidas

### 2. **Stack de Observabilidade Completa**

**Decisão:** Implementar os 4 pilares da observabilidade
- **Métricas**: Prometheus para monitoramento RED
- **Logs**: Logrus estruturado com correlação de traces
- **Traces**: OpenTelemetry para observabilidade distribuída
- **Profiling**: Pyroscope para análise de performance

**Benefícios:**
- Visibilidade completa da aplicação
- Debugging eficiente em produção
- Detecção proativa de problemas
- Análise de performance detalhada

### 3. **Dois TracerProviders Separados**

**Decisão:** TracerProvider separado para HTTP e SQL

**Benefícios:**
- Melhor organização visual no Tempo
- Traces SQL aparecem como serviço `my-inventory-mysql`
- Traces HTTP aparecem como serviço `inventory-app`
- Facilita análise de performance por camada

### 4. **Context Propagation**

**Decisão:** Usar `context.Context` em todas as operações

**Benefícios:**
- Cancelamento de operações
- Timeout configurável
- Trace correlation automática
- Preparação para rate limiting e circuit breakers

### 5. **Middleware Centralizado**

**Decisão:** Middleware único para métricas e logging

**Benefícios:**
- Consistência na coleta de métricas
- Logging padronizado
- Fácil adição de novas funcionalidades (auth, rate limiting)

