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

// Define o método Initialise que tem como receiver (app *App), com app sendo o nome do receiver e *App é o ponteiro pro tipo do receiver.
func (app *App) Initialise() error {
	connectionString := fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s", DBUser, DBPassword, DBName) // String de conexão com o banco de dados
	var err error
	app.DB, err = sql.Open("mysql", connectionString) // Inicializa a conexão com o driver MySQL, armazenando no campo DB da struct app.
	if err != nil { // Verifica se houve erro no preparo da conexão. Lembrando que a função Open não abre efetivamente a conexao,
		return err
	}
    // StrictSlash redireciona de path/ para path
	app.Router = mux.NewRouter().StrictSlash(true) // Inicializa o roteador e armazena no campo Router da struct app.
	return nil // Para saber mais: `go doc github.com/gorilla/mux.Router`
}

// Método Run - Usado para iniciar o servidor HTTP e manter ele rodando(Listen and Serve)
func (app *App) Run(addr string) { // Parâmetro adicional, que seria o :10000, como uma string, já que ela não está no Struct app.
	log.Fatal(http.ListenAndServe(addr, app.Router)) //	Inicia o servidor,e por padrão do Go, em 0.0.0.0
}

// Como todo Handler precisará de uma resposta, é melhor criar um genérico e reutilizável por todos os Handlers.
func sendResponse(w http.ResponseWriter, statusCode int, payload interface{}) { // Função que envia a resposta.
	response, _ := json.Marshal(payload)               //	Converte o payload para JSON, seja lá qual tipo de dado payload tenha.
	w.Header().Set("Content-Type", "application/json") // Define o cabeçalho da resposta.
	w.WriteHeader(statusCode)                          // Define o status code da resposta.
	w.Write(response)                                  // Escreve a resposta.
}
/*
Parâmetros de entrada da função acima:
- w http.ResponseWriter: o writer da resposta HTTP, que é do tipo interface do pacote http: http.ResponseWriter
	- Para saber mais: `go doc http.ResponseWriter`
- statusCode int: o código de status HTTP a ser retornado, que é do tipo inteiro
- payload interface{}: o payload a ser enviado na resposta, que é do tipo interface vazia,
que no Go significa que pode ser qualquer tipo de dado

response, _ --> Ignora qualquer erro que json.Marshal possa retornar.

*/

// Função de Escrita de erro, reutilizável da mesma forma que a função acima.
func sendError(w http.ResponseWriter, statusCode int, err error) { // Função que envia um erro.
	error_message := map[string]string{"error": err.Error()} // Converte o erro para string.
	sendResponse(w, statusCode, error_message)               // Envia a resposta com o erro.
}

// Método getProducts com entrada de writer(w) e request(r). Executa a função getProductsFromDB!
func (app *App) getProducts(w http.ResponseWriter, r *http.Request) { // Seguindo o jargão de Go: w pra writer e r pra request
	products, err := getProductsFromDB(app.DB)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err)
		return
	}
	sendResponse(w, http.StatusOK, products)
}
// r foi declarado mas não foi usado, então o ideal seria omitir ele usando o underline.

// Método getProduct
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
