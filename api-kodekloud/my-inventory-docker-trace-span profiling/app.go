package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
	"github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	// Import para o trace
	"github.com/XSAM/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	// Trace para o mux
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

// --- Funções sendError e sendResponse  ---
func sendError(w http.ResponseWriter, r *http.Request, status int, err error) {
	logrus.WithContext(r.Context()).WithFields(logrus.Fields{
		"component": "http_handler",
		"status":    status,
		"error":     err.Error(),
	}).Error("Erro na requisição")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func sendResponse(ctx context.Context, w http.ResponseWriter, status int, data interface{}) {
	logger := logrus.WithContext(ctx).WithFields(logrus.Fields{
		"component": "http_handler",
		"status":    status,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			logger.WithError(err).Error("Erro ao codificar a resposta JSON")
			return
		}
		logger.Debug("Resposta enviada com sucesso")
	}
}

// --- Estrutura App  ---
type App struct {
	Router *mux.Router
	DB     *sql.DB
}

// --- Método Initialise ---
func (app *App) Initialise(sqlTracerProvider trace.TracerProvider) error {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")

	if dbUser == "" || dbPassword == "" || dbName == "" || dbHost == "" {
		return errors.New("variáveis de ambiente do banco de dados (DB_USER, DB_PASSWORD, DB_NAME, DB_HOST) não configuradas")
	}

	var err error

	if err != nil {
		logrus.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	originalDriverName := "mysql"
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbName)

	app.DB, err = otelsql.Open(originalDriverName, connectionString,
		otelsql.WithTracerProvider(sqlTracerProvider),
		otelsql.WithAttributes(
			semconv.DBSystemMySQL,
			semconv.DBNameKey.String(dbName),
			semconv.NetPeerNameKey.String(dbHost),
			semconv.NetPeerPortKey.Int(3306),
		),
		otelsql.WithSQLCommenter(true),
	)
	if err != nil {
		logrus.WithError(err).Errorf("Erro ao conectar com o banco de dados (%s) usando otelsql.Open", dbName)
		return fmt.Errorf("falha ao abrir conexão com o banco de dados instrumentado: %w", err)
	}

	// Teste para subir o MySQL
	var db *sql.DB
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbName) // Definição da variável dsn

	for _ = range make([]struct{}, 10) { // Cria um slice de 10 elementos, não importa o tipo, só para iteração
		db, err = sql.Open(originalDriverName, dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		logrus.Printf("MySQL ainda não disponível (%v). Tentando novamente em 2s...", err)
		time.Sleep(2 * time.Second)
	}
	// Fim do teste

	// Define timeout dentro do contexto do span
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// PingContext
	err = app.DB.PingContext(ctx)
	if err != nil {
		logrus.WithError(err).Errorf("Erro ao fazer ping no banco de dados (%s) após conexão otelsql", dbName)
		app.DB.Close()
		return fmt.Errorf("falha ao fazer ping no banco de dados (%s): %w", dbName, err)
	}

	logrus.Infof("Conexão com o banco de dados MySQL (%s@%s) instrumentada com OTEL (serviço: my-inventory-mysql) estabelecida com sucesso", dbName, dbHost)

	app.Router = mux.NewRouter().StrictSlash(true)
	// ORDEM CORRETA DOS MIDDLEWARES: Tracing PRIMEIRO, depois Prometheus
	app.Router.Use(otelmux.Middleware("inventory-app")) // Tracing primeiro!
	app.Router.Use(prometheusMiddleware)                // Métricas depois
	app.HandleRequests()
	go app.startBackgroundProductCountUpdate()

	logrus.Info("Aplicação inicializada com sucesso")
	return nil
}

// --- Método HandleRequests  ---
func (app *App) HandleRequests() {
	// Registrar handlers com profiling contextual
	app.Router.HandleFunc("/products", ProfiledHTTPHandler("get_products", app.getProducts)).Methods("GET")
	app.Router.HandleFunc("/product/{id:[0-9]+}", ProfiledHTTPHandler("get_product", app.getProduct)).Methods("GET")
	app.Router.HandleFunc("/product", ProfiledHTTPHandler("create_product", app.createProduct)).Methods("POST")
	app.Router.HandleFunc("/product/{id:[0-9]+}", ProfiledHTTPHandler("update_product", app.updateProduct)).Methods("PUT")
	app.Router.HandleFunc("/product/{id:[0-9]+}", ProfiledHTTPHandler("delete_product", app.deleteProduct)).Methods("DELETE")
	app.Router.HandleFunc("/health", ProfiledHTTPHandler("health_check", app.healthCheck)).Methods("GET")
}

// --- Método Run  ---
func (app *App) Run(addr string) {
	logrus.Infof("Lógica de execução movida para main.go para integração com otelhttp.")
}

// --- Handlers da API  ---
func (app *App) getProducts(w http.ResponseWriter, r *http.Request) {
	// Extrai span do contexto para logs com trace_id e span_id
	span := trace.SpanFromContext(r.Context())
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

	products, err := getProductsFromDB(r.Context(), app.DB)
	if err != nil {
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
		logEntry.Error("Erro ao obter produtos do banco de dados")
		sqlErrorsTotal.Inc()
		sendError(w, r, http.StatusInternalServerError, errors.New("failed to retrieve products"))
		return
	}

	successEntry := logrus.WithContext(r.Context()).WithField("num_products", len(products)).WithFields(logrus.Fields{
		"component": "http_handler",
		"operation": "get_products",
	})
	if span.SpanContext().IsValid() {
		successEntry = successEntry.WithFields(logrus.Fields{
			"trace_id": span.SpanContext().TraceID().String(),
			"span_id":  span.SpanContext().SpanID().String(),
		})
	}
	successEntry.Info("Listando produtos")
	sendResponse(r.Context(), w, http.StatusOK, products)
}

func (app *App) getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, _ := strconv.Atoi(vars["id"])

	p := product{ID: key}
	// Passa o contexto da requisição para a função do banco de dados
	err := p.getProduct(r.Context(), app.DB) // Passando r.Context()
	if err != nil {
		// Agora podemos confiar mais no erro retornado pela função getProduct
		if errors.Is(err, sql.ErrNoRows) {
			logrus.WithContext(r.Context()).WithField("product_id", key).Info("Produto não encontrado")
			sendError(w, r, http.StatusNotFound, fmt.Errorf("product with ID %d not found", key))
		} else {
			logrus.WithContext(r.Context()).WithError(err).WithField("product_id", key).Error("Erro ao buscar produto no banco de dados")
			sqlErrorsTotal.Inc()
			sendError(w, r, http.StatusInternalServerError, errors.New("failed to retrieve product"))
		}
		return
	}
	logrus.WithContext(r.Context()).WithField("product_id", key).Info("Exibindo produto")
	sendResponse(r.Context(), w, http.StatusOK, p)
}

func (app *App) createProduct(w http.ResponseWriter, r *http.Request) {
	var p product
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&p); err != nil {
		logrus.WithContext(r.Context()).WithError(err).Warn("Payload de requisição inválido para criar produto")
		sendError(w, r, http.StatusBadRequest, errors.New("invalid request payload"))
		return
	}
	defer r.Body.Close()

	if p.Name == "" || p.Price < 0 || p.Quantity < 0 {
		logrus.WithContext(r.Context()).Warn("Tentativa de criar produto com dados inválidos")
		sendError(w, r, http.StatusBadRequest, errors.New("invalid product data: name is required, price and quantity cannot be negative"))
		return
	}

	// Passa o contexto da requisição para a função do banco de dados
	err := p.createProduct(r.Context(), app.DB) // <<< MODIFICADO: Passando r.Context()
	if err != nil {
		logrus.WithContext(r.Context()).WithError(err).Error("Erro ao criar produto no banco de dados")
		sqlErrorsTotal.Inc()
		sendError(w, r, http.StatusInternalServerError, errors.New("failed to create product"))
		return
	}

	logrus.WithContext(r.Context()).WithField("product_id", p.ID).Info("Produto criado")
	sendResponse(r.Context(), w, http.StatusCreated, p)
}

func (app *App) updateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, _ := strconv.Atoi(vars["id"])

	var p product
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&p); err != nil {
		logrus.WithContext(r.Context()).WithError(err).Warn("Payload de requisição inválido para atualizar produto")
		sendError(w, r, http.StatusBadRequest, errors.New("invalid request payload"))
		return
	}
	defer r.Body.Close()

	if p.Name == "" || p.Price < 0 || p.Quantity < 0 {
		logrus.WithContext(r.Context()).WithField("product_id", key).Warn("Tentativa de atualizar produto com dados inválidos")
		sendError(w, r, http.StatusBadRequest, errors.New("invalid product data: name is required, price and quantity cannot be negative"))
		return
	}

	p.ID = key
	// Passa o contexto da requisição para a função do banco de dados
	err := p.updateProduct(r.Context(), app.DB) // Passando r.Context()
	if err != nil {
		// Verifica o erro sql.ErrNoRows retornado pela função updateProduct
		if errors.Is(err, sql.ErrNoRows) {
			logrus.WithContext(r.Context()).WithField("product_id", key).Info("Produto não encontrado para atualização")
			sendError(w, r, http.StatusNotFound, fmt.Errorf("product with ID %d not found for update", key))
		} else {
			logrus.WithContext(r.Context()).WithError(err).WithField("product_id", key).Error("Erro ao atualizar produto")
			sqlErrorsTotal.Inc()
			sendError(w, r, http.StatusInternalServerError, errors.New("failed to update product"))
		}
		return
	}
	logrus.WithContext(r.Context()).WithField("product_id", key).Info("Produto atualizado")
	sendResponse(r.Context(), w, http.StatusOK, p)
}

func (app *App) deleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, _ := strconv.Atoi(vars["id"])

	p := product{ID: key}
	// Passa o contexto da requisição para a função do banco de dados
	err := p.deleteProduct(r.Context(), app.DB) // Passando r.Context()
	if err != nil {
		// Verifica o erro sql.ErrNoRows retornado pela função deleteProduct
		if errors.Is(err, sql.ErrNoRows) {
			logrus.WithContext(r.Context()).WithField("product_id", key).Info("Produto não encontrado para deleção")
			sendError(w, r, http.StatusNotFound, fmt.Errorf("product with ID %d not found for deletion", key))
		} else {
			logrus.WithContext(r.Context()).WithError(err).WithField("product_id", key).Error("Erro ao deletar produto")
			sqlErrorsTotal.Inc()
			sendError(w, r, http.StatusInternalServerError, errors.New("failed to delete product"))
		}
		return
	}
	logrus.WithContext(r.Context()).WithField("product_id", key).Info("Produto deletado")
	sendResponse(r.Context(), w, http.StatusOK, map[string]string{"result": "success", "message": fmt.Sprintf("Product with ID %d deleted", key)})
}

// --- Health Check (sem alterações, já usava PingContext) ---
func (app *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.DB.PingContext(ctx); err != nil {
		logrus.WithError(err).Warn("Health check falhou (DB ping)")
		sendError(w, r, http.StatusServiceUnavailable, fmt.Errorf("database connection failed: %v", err))
		return
	}
	sendResponse(r.Context(), w, http.StatusOK, map[string]string{"status": "ok", "database": "connected"})
}

// --- Atualização da Métrica de Contagem de Produtos ---

// Função interna para buscar a contagem atual (agora passa contexto)
func (app *App) getCurrentProductCount() (int, error) {
	// Cria um contexto com timeout para esta chamada interna
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Passa o contexto criado para a função countProducts
	count, err := countProducts(ctx, app.DB) // Passando ctx
	if err != nil {
		logrus.WithError(err).Error("Erro ao contar produtos no banco de dados para métrica")
		sqlErrorsTotal.Inc()
		return 0, err
	}
	return count, nil
}

// Goroutine para atualizar periodicamente a métrica (sem alterações na lógica do ticker)
func (app *App) startBackgroundProductCountUpdate() {
	count, err := app.getCurrentProductCount()
	if err == nil {
		productsInDB.Set(float64(count))
		logrus.Infof("Métrica inicial 'products_in_db' definida para: %d", count)
	} else {
		logrus.Warn("Não foi possível definir a métrica inicial 'products_in_db'")
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	logrus.Info("Iniciando atualização periódica da métrica 'products_in_db' a cada 5 minutos")

	for range ticker.C {
		count, err := app.getCurrentProductCount()
		if err == nil {
			productsInDB.Set(float64(count))
			logrus.Debugf("Métrica 'products_in_db' atualizada para: %d", count)
		} else {
			logrus.Warn("Falha ao atualizar periodicamente a métrica 'products_in_db'")
		}
	}
}
