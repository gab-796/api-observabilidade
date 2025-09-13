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
/*
Gorilla mux é um multiplexador de requisições HTTP, ou seja, um roteador do Go.
Ele fica entre o servidor HTTP e as funções handler e decide qual handler será chamado pra cada request, baseado nas regras
de rota definidas.

multiplexador --> Direciona uma mesma entrada(requisições HTTP todas chegando na porta 10000) para  saídas diferentes(os handlers)
conforme o path, método, host, variáveis de rota, etc.

O padrão do Go é http.ServeMux, que é um multiplexador simples, mas o Gorilla mux é mais completo e flexível.
Porém ele está sendo substituído pelo Chi, que é mais leve e performático.

go doc github.com/gorilla/mux.Router  --> Afirma que Router é uma struct.
go doc sql.DB --> Afirma que DB é uma struct.
*/


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
/*
A função initialise pega a struct app e preenche os campos DB e Router dela, que antes disso só tinham o valor zero, que no caso de ponteiros é nil.
o type error também é uma struct! --> Yukio

go doc builtin.error --> Afirma que error é uma interface!

type error interface {
	Error() string
}
    The error built-in interface type is the conventional interface for
    representing an error condition, with the nil value representing no error.

*/


// Método Run - Usado para iniciar o servidor HTTP e manter ele rodando(Listen and Serve)
func (app *App) Run(addr string) { // Parâmetro adicional, que seria o :10000, como uma string, já que ela não está no Struct app.
	log.Fatal(http.ListenAndServe(addr, app.Router)) //	Inicia o servidor,e por padrão do Go, em 0.0.0.0
}

// Como todo Handler precisará de uma resposta, é melhor criar um genérico e reutilizável por todos os Handlers.
func sendResponse(w http.ResponseWriter, statusCode int, payload interface{}) { // Função que envia a resposta.
	response, _ := json.Marshal(payload)               //	Converte o payload para JSON, seja lá qual tipo de dado payload tenha.
	w.Header().Set("Content-Type", "application/json") // Define o cabeçalho da resposta, já em formato JSON.
	w.WriteHeader(statusCode)                          // Define o status code da resposta.
	w.Write(response)                                  // Escreve os dados JSON no corpo da resposta.
}
/*
Parâmetros de entrada da função acima:
- w http.ResponseWriter: o writer da resposta HTTP, que é do tipo interface do pacote http: http.ResponseWriter
	- Para saber mais: `go doc http.ResponseWriter`
- statusCode int: o código de status HTTP a ser retornado, que é do tipo inteiro
- payload interface{}: o payload a ser enviado na resposta, que é do tipo interface vazia,
que no Go significa que pode ser qualquer tipo de dado

response, _ --> Ignora qualquer erro que json.Marshal possa retornar.
O ideal seria tratar o erro:
    response, err := json.Marshal(payload)
    if err != nil {
        http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
        return
    }

O json.Marshal(payload) é o coração da função, que converte o payload para JSON.
Esta operação de marshalling é crucial para APIs REST,
pois transforma estruturas de dados Go em um formato universalmente legível por diferentes clientes (navegadores, aplicações móveis, etc.).
*/

// Função de Escrita de erro, reutilizável da mesma forma que a função acima.
func sendError(w http.ResponseWriter, statusCode int, err error) { // Função que envia um erro.
	error_message := map[string]string{"error": err.Error()} // Map que encapsula a mensagem de erro em JSON, usando a interface error.
	sendResponse(w, statusCode, error_message) // Usa a função de Response no formato {"error": "mensagem do erro"}
}

// Método getProducts com entrada de writer(w) e request(r). Executa a função getProductsFromDB
func (app *App) getProducts(w http.ResponseWriter, r *http.Request) { // Seguindo o jargão de Go: w pra writer e r pra request
	products, err := getProductsFromDB(app.DB)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err)
		return
	}
	sendResponse(w, http.StatusOK, products)
}
/* r foi declarado mas não foi usado, então o ideal seria omitir ele usando o underline.
O return logo depois de sendError é o early return, que já para a execução da função
logo quando encontra um erro.
A interface vazia da função sendResponse agora é populada pelo payload products
*/

// Método getProduct
func (app *App) getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, err := strconv.Atoi(vars["id"]) // Converte o id da URL, que é sempre uma string, para inteiro.
	if err != nil { // Tratamento de erro da conversão acima.
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return // Para a função imediatamente, em caso de erro.
	}

	p := product{ID: key} // Definie a struct product com o ID em inteiro que foi convertido.
	err = p.getProduct(app.DB) // Chama o método getProduct da struct product, que preenche o restante dos campos dela.
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

/*
mux.Vars(r) --> Função do pacote mux que extrai as variáveis da URL da requisição r.
O roteador gorilla/mux pega a URL (ex: /product/123) e, como a rota foi definida com /{id},
ele cria um mapa onde a chave é "id" e o valor é a string "123".

Se o usuário enviar um ID inválido, o tratamento de erro http.StatusBadRequest já retorna um 400,
indicando problema na requisição do cliente.


*/

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

/*
decoder := json.NewDecoder(r.Body), onde r.Body é um io.ReadCloser que contém o corpo de requisição HTTP.
Já o json.NewDecoder cria um decodificador que vai ler desse io.Reader

decoder.Decode(&p) lê o JSON do body e preenche a struct product
O &p passa o endereço da variável para que o decoder possa modificá-la

Exemplo --> Requisição chega assim:
POST /product
Content-Type: application/json

{
    "name": "Notebook",
    "quantity": 10,
    "price": 2500.99
}
Agora p contém:
 - p.Name = "Notebook"
 - p.Quantity = 10
 - p.Price = 2500.99

O defer r.Body.Close() garante que o corpo da requisição seja fechado adequadamente, 
independentemente de como a função termine, evitando vazamentos de recursos.

Se ela for criada com sucesso, é retornado 201 e o produto nasce em JSON.
*/

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
/*
O processo começa com a extração do identificador do produto a partir da URL.
A função mux.Vars(r) captura as variáveis de rota definidas no padrão da URL (como /products/{id}), retornando um mapa onde a chave "id" contém o valor como string. 
Como IDs são normalmente numéricos, strconv.Atoi() converte essa string para inteiro.
Se a conversão falhar (por exemplo, se alguém enviar /products/abc), a função retorna imediatamente um erro 400 Bad Request com a mensagem "invalid product ID".
*/


func (app *App) deleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r) // A função mux.Vars() captura as variáveis de rota da URL.
	key, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return
	}
	p := product{ID: key}
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
/*
p := product{ID: key} --> cria uma instância mínima da struct product contendo apenas o ID: p := product{ID: key}.
Esta abordagem é eficiente pois não requer buscar todos os dados do produto no banco apenas para deletá-lo - o ID é suficiente para a operação de exclusão. 
Isso demonstra um uso otimizado dos recursos, evitando consultas desnecessárias.

Já o tratamento de erro é específico para o caso de "product not found", retornando 404.
E para qualquer outro, retorna 500.

A função de resposta envia um 200 OK com um JSON simples confirmando a exclusão.
Poderia ser um 204 No Content tb.
*/

func (app *App) HandleRequests() {
	app.Router.HandleFunc("/products", app.getProducts).Methods("GET")
	app.Router.HandleFunc("/product/{id}", app.getProduct).Methods("GET")
	app.Router.HandleFunc("/product", app.createProduct).Methods("POST")
	app.Router.HandleFunc("/product/{id}", app.updateProduct).Methods("PUT")
	app.Router.HandleFunc("/product/{id}", app.deleteProduct).Methods("DELETE")
}

/*
Funções handler que tem como receiver (app *App), ou seja, são métodos da struct App.
Cada rota associa um path e um método HTTP.
A primeira, por exemplo, pega o path localhost:10000/products com o Handler app.getProducts.
O método Methods("GET") limita a rota para aceitar apenas requisições GET.

É usada a função para cada produto, definido acima.
*/