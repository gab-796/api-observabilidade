package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// --- Funções auxiliares (sendError, sendResponse)

func sendError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func sendResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json") // Define o content type
	w.WriteHeader(status)                              //Seta o status code antes de enviar a resposta
	if data != nil {                                   // Verifica se existe dados a serem retornados.
		err := json.NewEncoder(w).Encode(data) // escreve a resposta.
		if err != nil {
			log.WithError(err).Error("Erro ao codificar a resposta JSON")
			// Não chamamos sendError aqui para evitar recursão infinita;
			// apenas logamos e retornamos, o status code já foi setado.
		}
	}
}

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

func (app *App) Initialise() error {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")

	connectionString := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", dbUser, dbPassword, dbHost, dbName)
	var err error
	app.DB, err = sql.Open("mysql", connectionString)
	if err != nil {
		log.WithError(err).Error("Erro ao conectar com o banco de dados")
		return err
	}

	app.Router = mux.NewRouter().StrictSlash(true)
	app.HandleRequests()
	app.Router.Use(prometheusMiddleware) // Aplica o middleware a TODAS as rotas

	// Inicializa a métrica products_in_db (em uma goroutine)
	go app.updateProductsInDBMetric() //inicia a go routine

	log.Info("Aplicação inicializada com sucesso")
	return nil
}

func (app *App) HandleRequests() {
	app.Router.HandleFunc("/products", app.getProducts).Methods("GET")
	app.Router.HandleFunc("/product/{id}", app.getProduct).Methods("GET")
	app.Router.HandleFunc("/product", app.createProduct).Methods("POST")
	app.Router.HandleFunc("/product/{id}", app.updateProduct).Methods("PUT")
	app.Router.HandleFunc("/product/{id}", app.deleteProduct).Methods("DELETE")
}

func (app *App) Run(addr string) {
	log.Infof("Servidor iniciando na porta %s", addr)
	if err := http.ListenAndServe(addr, app.Router); err != nil {
		log.WithError(err).Fatal("Erro ao iniciar o servidor")
	}
}

func (app *App) getProducts(w http.ResponseWriter, r *http.Request) {
	products, err := getProductsFromDB(app.DB)
	if err != nil {
		log.WithError(err).Error("Erro ao obter produtos do banco de dados")
		sqlErrorsTotal.Inc() // Incrementa o contador de erros SQL
		sendError(w, http.StatusInternalServerError, err)
		return
	}
	log.WithField("num_products", len(products)).Info("Listando produtos")
	sendResponse(w, http.StatusOK, products)
}

func (app *App) getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.WithError(err).Warn("ID do produto inválido")
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return
	}

	p := product{ID: key}
	err = p.getProduct(app.DB)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.WithField("product_id", key).Info("Produto não encontrado")
			sendError(w, http.StatusNotFound, fmt.Errorf("produto não encontrado"))

		} else {
			log.WithError(err).Error("Erro ao buscar produto no banco de dados")
			sqlErrorsTotal.Inc() // Incrementa o contador de erros SQL
			sendError(w, http.StatusInternalServerError, err)
		}
		return
	}
	log.WithField("product_id", key).Info("Exibindo produto")
	sendResponse(w, http.StatusOK, p)
}

func (app *App) createProduct(w http.ResponseWriter, r *http.Request) {
	var p product
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		log.WithError(err).Warn("Payload de requisição inválido")
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid request payload"))
		return
	}
	defer r.Body.Close()

	err := p.createProduct(app.DB)
	if err != nil {
		log.WithError(err).Error("Erro ao criar produto no banco de dados")
		sqlErrorsTotal.Inc() // Incrementa o contador de erros SQL
		sendError(w, http.StatusInternalServerError, err)
		return
	}

	log.WithField("product_id", p.ID).Info("Produto criado")
	sendResponse(w, http.StatusCreated, p)

	// Atualiza a métrica de produtos no banco de dados
	app.updateProductsInDBMetric()
}
func (app *App) updateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.WithError(err).Warn("ID do produto inválido")
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return
	}

	var p product
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		log.WithError(err).Warn("Payload de requisição inválido")
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid request payload"))
		return
	}
	defer r.Body.Close()

	p.ID = key
	err = p.updateProduct(app.DB)
	if err != nil {
		log.WithError(err).Error("Erro ao atualizar produto")
		sqlErrorsTotal.Inc() // Incrementa o contador de erros SQL
		sendError(w, http.StatusInternalServerError, err)
		return
	}
	log.WithField("product_id", p.ID).Info("Produto atualizado")
	sendResponse(w, http.StatusOK, p)
}

func (app *App) deleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.WithError(err).Warn("ID do produto inválido")
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return
	}

	p := product{ID: key}
	err = p.deleteProduct(app.DB)
	if err != nil {
		log.WithError(err).Error("Erro ao deletar produto")
		sqlErrorsTotal.Inc() // Incrementa o contador de erros SQL
		sendError(w, http.StatusInternalServerError, err)
		return
	}
	log.WithField("product_id", key).Info("Produto deletado")
	sendResponse(w, http.StatusOK, map[string]string{"result": "success"})

	// Atualiza a métrica de produtos no banco de dados
	app.updateProductsInDBMetric()
}

// Função para atualizar a métrica de produtos no banco de dados
func (app *App) updateProductsInDBMetric() {
	count, err := countProducts(app.DB)
	if err != nil {
		log.WithError(err).Error("Erro ao contar produtos no banco de dados")
		sqlErrorsTotal.Inc() // Incrementa o contador de erros SQL
		return
	}
	productsInDB.Set(float64(count))
}
