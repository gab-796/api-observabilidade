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
	"time" // Importar time para o ticker

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	// Import para o trace
	"github.com/XSAM/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// --- Funções sendError e sendResponse  ---
func sendError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func sendResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			log.WithError(err).Error("Erro ao codificar a resposta JSON")
		}
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
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
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
		log.WithError(err).Errorf("Erro ao conectar com o banco de dados (%s) usando otelsql.Open", dbName)
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
		log.Printf("MySQL ainda não disponível (%v). Tentando novamente em 2s...", err)
		time.Sleep(2 * time.Second)
	}
	// Fim do teste


	// Define timeout dentro do contexto do span
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// PingContext
	err = app.DB.PingContext(ctx)
	if err != nil {
		log.WithError(err).Errorf("Erro ao fazer ping no banco de dados (%s) após conexão otelsql", dbName)
		app.DB.Close()
		return fmt.Errorf("falha ao fazer ping no banco de dados (%s): %w", dbName, err)
	}

	log.Infof("Conexão com o banco de dados MySQL (%s@%s) instrumentada com OTEL (serviço: my-inventory-mysql) estabelecida com sucesso", dbName, dbHost)

	app.Router = mux.NewRouter().StrictSlash(true)
	app.Router.Use(prometheusMiddleware)
	app.HandleRequests()
	go app.startBackgroundProductCountUpdate()

	log.Info("Aplicação inicializada com sucesso")
	return nil
}

// --- Método HandleRequests  ---
func (app *App) HandleRequests() {
	app.Router.HandleFunc("/products", app.getProducts).Methods("GET")
	app.Router.HandleFunc("/product/{id:[0-9]+}", app.getProduct).Methods("GET")
	app.Router.HandleFunc("/product", app.createProduct).Methods("POST")
	app.Router.HandleFunc("/product/{id:[0-9]+}", app.updateProduct).Methods("PUT")
	app.Router.HandleFunc("/product/{id:[0-9]+}", app.deleteProduct).Methods("DELETE")
	app.Router.HandleFunc("/health", app.healthCheck).Methods("GET")
}

// --- Método Run  ---
func (app *App) Run(addr string) {
	log.Infof("Lógica de execução movida para main.go para integração com otelhttp.")
}

// --- Handlers da API  ---
func (app *App) getProducts(w http.ResponseWriter, r *http.Request) {
	// Passa o contexto da requisição para a função do banco de dados
	products, err := getProductsFromDB(r.Context(), app.DB) // Passando r.Context()
	if err != nil {
		log.WithError(err).Error("Erro ao obter produtos do banco de dados")
		sqlErrorsTotal.Inc()
		sendError(w, http.StatusInternalServerError, errors.New("failed to retrieve products"))
		return
	}
	log.WithField("num_products", len(products)).Info("Listando produtos")
	sendResponse(w, http.StatusOK, products)
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
			log.WithField("product_id", key).Info("Produto não encontrado")
			sendError(w, http.StatusNotFound, fmt.Errorf("product with ID %d not found", key))
		} else {
			log.WithError(err).WithField("product_id", key).Error("Erro ao buscar produto no banco de dados")
			sqlErrorsTotal.Inc()
			sendError(w, http.StatusInternalServerError, errors.New("failed to retrieve product"))
		}
		return
	}
	log.WithField("product_id", key).Info("Exibindo produto")
	sendResponse(w, http.StatusOK, p)
}

func (app *App) createProduct(w http.ResponseWriter, r *http.Request) {
	var p product
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&p); err != nil {
		log.WithError(err).Warn("Payload de requisição inválido para criar produto")
		sendError(w, http.StatusBadRequest, errors.New("invalid request payload"))
		return
	}
	defer r.Body.Close()

	if p.Name == "" || p.Price < 0 || p.Quantity < 0 {
		log.Warn("Tentativa de criar produto com dados inválidos")
		sendError(w, http.StatusBadRequest, errors.New("invalid product data: name is required, price and quantity cannot be negative"))
		return
	}

	// Passa o contexto da requisição para a função do banco de dados
	err := p.createProduct(r.Context(), app.DB) // <<< MODIFICADO: Passando r.Context()
	if err != nil {
		log.WithError(err).Error("Erro ao criar produto no banco de dados")
		sqlErrorsTotal.Inc()
		sendError(w, http.StatusInternalServerError, errors.New("failed to create product"))
		return
	}

	log.WithField("product_id", p.ID).Info("Produto criado")
	sendResponse(w, http.StatusCreated, p)
}

func (app *App) updateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, _ := strconv.Atoi(vars["id"])

	var p product
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&p); err != nil {
		log.WithError(err).Warn("Payload de requisição inválido para atualizar produto")
		sendError(w, http.StatusBadRequest, errors.New("invalid request payload"))
		return
	}
	defer r.Body.Close()

	if p.Name == "" || p.Price < 0 || p.Quantity < 0 {
		log.WithField("product_id", key).Warn("Tentativa de atualizar produto com dados inválidos")
		sendError(w, http.StatusBadRequest, errors.New("invalid product data: name is required, price and quantity cannot be negative"))
		return
	}

	p.ID = key
	// Passa o contexto da requisição para a função do banco de dados
	err := p.updateProduct(r.Context(), app.DB) // Passando r.Context()
	if err != nil {
		// Verifica o erro sql.ErrNoRows retornado pela função updateProduct
		if errors.Is(err, sql.ErrNoRows) {
			log.WithField("product_id", key).Info("Produto não encontrado para atualização")
			sendError(w, http.StatusNotFound, fmt.Errorf("product with ID %d not found for update", key))
		} else {
			log.WithError(err).WithField("product_id", key).Error("Erro ao atualizar produto")
			sqlErrorsTotal.Inc()
			sendError(w, http.StatusInternalServerError, errors.New("failed to update product"))
		}
		return
	}
	log.WithField("product_id", key).Info("Produto atualizado")
	sendResponse(w, http.StatusOK, p)
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
			log.WithField("product_id", key).Info("Produto não encontrado para deleção")
			sendError(w, http.StatusNotFound, fmt.Errorf("product with ID %d not found for deletion", key))
		} else {
			log.WithError(err).WithField("product_id", key).Error("Erro ao deletar produto")
			sqlErrorsTotal.Inc()
			sendError(w, http.StatusInternalServerError, errors.New("failed to delete product"))
		}
		return
	}
	log.WithField("product_id", key).Info("Produto deletado")
	sendResponse(w, http.StatusOK, map[string]string{"result": "success", "message": fmt.Sprintf("Product with ID %d deleted", key)})
}

// --- Health Check (sem alterações, já usava PingContext) ---
func (app *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.DB.PingContext(ctx); err != nil {
		log.WithError(err).Warn("Health check falhou (DB ping)")
		sendError(w, http.StatusServiceUnavailable, fmt.Errorf("database connection failed: %v", err))
		return
	}
	sendResponse(w, http.StatusOK, map[string]string{"status": "ok", "database": "connected"})
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
		log.WithError(err).Error("Erro ao contar produtos no banco de dados para métrica")
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
		log.Infof("Métrica inicial 'products_in_db' definida para: %d", count)
	} else {
		log.Warn("Não foi possível definir a métrica inicial 'products_in_db'")
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Info("Iniciando atualização periódica da métrica 'products_in_db' a cada 5 minutos")

	for range ticker.C {
		count, err := app.getCurrentProductCount()
		if err == nil {
			productsInDB.Set(float64(count))
			log.Debugf("Métrica 'products_in_db' atualizada para: %d", count)
		} else {
			log.Warn("Falha ao atualizar periodicamente a métrica 'products_in_db'")
		}
	}
}
