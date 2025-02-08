package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

func (app *App) Initialise() error { // Método que inicializa
	connectionString := fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s", DBUser, DBPassword, DBName)
	var err error
	app.DB, err = sql.Open("mysql", connectionString)
	if err != nil { // Verifica se houve erro na conexão.
		return err
	}

	app.Router = mux.NewRouter().StrictSlash(true)
	return nil
}

func (app *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, app.Router)) //	Inicia o servidor na porta 10000,e por padrão do Go em localhost.
}

func sendResponse(w http.ResponseWriter, statusCode int, payload interface{}) { // Função que envia a resposta.
	response, _ := json.Marshal(payload)               //	Converte o payload para JSON.
	w.Header().Set("Content-Type", "application/json") // Define o cabeçalho da resposta.
	w.WriteHeader(statusCode)                          // Define o status code da resposta.
	w.Write(response)                                  // Escreve a resposta.
}

func sendError(w http.ResponseWriter, statusCode int, err error) { // Função que envia um erro.
	error_message := map[string]string{"error": err.Error()} // Converte o erro para string.
	sendResponse(w, statusCode, error_message)               // Envia a resposta com o erro.
}

func (app *App) getProducts(w http.ResponseWriter, r *http.Request) {
	products, err := getProductsFromDB(app.DB)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err)
		return
	}
	sendResponse(w, http.StatusOK, products)
}

func (app *App) getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return
	}

	p := product{ID: key}
	err = p.getProduct(app.DB)
	if err != nil {
		if err == sql.ErrNoRows {
			sendError(w, http.StatusNotFound, fmt.Errorf("product not found"))
			return
		}
		sendError(w, http.StatusInternalServerError, err)
		return
	}

	sendResponse(w, http.StatusOK, p)
}

func (app *App) createProduct(w http.ResponseWriter, r *http.Request) {
	var p product
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid request payload"))
		return
	}
	defer r.Body.Close()

	if err := p.createProduct(app.DB); err != nil {
		sendError(w, http.StatusInternalServerError, err)
		return
	}

	sendResponse(w, http.StatusCreated, p)
}

func (app *App) updateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return
	}

	var p product
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid request payload"))
		return
	}
	defer r.Body.Close()

	p.ID = key
	if err := p.updateProduct(app.DB); err != nil {
		sendError(w, http.StatusInternalServerError, err)
		return
	}

	sendResponse(w, http.StatusOK, p)
}

func (app *App) deleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r) // Todo esse bloco é pra pegar o id do produto que será deletado.
	key, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return
	}

	p := product{ID: key} // Cria um produto com o id que foi passado.
	if err := p.deleteProduct(app.DB); err != nil {
		if err.Error() == "product not found" {
			sendError(w, http.StatusNotFound, err)
		} else {
			sendError(w, http.StatusInternalServerError, err)
		}
		return
	}
	sendResponse(w, http.StatusOK, map[string]string{"result": "successful deletion"})
}

func (app *App) HandleRequests() {
	app.Router.HandleFunc("/products", app.getProducts).Methods("GET")
	app.Router.HandleFunc("/product/{id}", app.getProduct).Methods("GET")
	app.Router.HandleFunc("/product", app.createProduct).Methods("POST")
	app.Router.HandleFunc("/product/{id}", app.updateProduct).Methods("PUT")
	app.Router.HandleFunc("/product/{id}", app.deleteProduct).Methods("DELETE")
}
