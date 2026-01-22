package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	if err != nil { // Verifica se houve erro no preparo da conexão.
		return err
	}
	app.Router = mux.NewRouter().StrictSlash(true)
	return nil // Para saber mais: `go doc github.com/gorilla/mux.Router`
}

/*
o type error é uma struct(go doc builtin.error)
type error interface {
	Error() string
}
    The error built-in interface type is the conventional interface for
    representing an error condition, with the nil value representing no error.

app -> é uma variável onde guardamos a instância de App.
app.DB -> preenche o campo DB da struct App com a conexão ao banco de dados.
app.Router -> preenche o campo Router da struct App com uma nova instância do roteador Gorilla mux.

A função Open do pacote sql apenas valida a conexão e prepara o pool de conexões, mas não abre efetivamente a conexão com o banco.
Isso só acontece quando a primeira query é executada.

A função mux.NewRouter() cria uma nova instância do roteador Gorilla mux.
A struct Router funciona como um multiplexador de requisições HTTP, bem mais avançado que o padrão do Go (http.ServeMux).
Quando invocada, ela apenas aloca memória pra a struct Router e inicializa suas estruturas internas que armazenarão as definições de rotas.
As rotas só são conhecidas quando o método HandleRequests() é chamado.

O StrictSlash(true) trata /products e /products/ como a mesma rota, ou seja, redireciona automaticamente
com status 301 Moved Permanently, evitando 404s.
*/


// Método Run - Usado para iniciar o servidor HTTP e manter ele rodando(Listen and Serve)
func (app *App) Run(addr string) error {
	err := http.ListenAndServe(addr, app.Router)
	if err != nil {
		return err
	}
	return nil
}
/*
addr é uma string que especifica o endereço de rede e vai receber o valor ":10000" quando o método for chamado
no main.go `app.Run(":10000")`
Como está :10000, ele vai escutar em todas as interfaces de rede (0.0.0.0), mas poderia colocar um IP específico, como
192.168.1.100 ou até localhost.
Outra forma seria colocar :http, assim seria usada a porta 80.

http.ListenAndServe é uma função do pacote net/http da biblioteca padrão do Go.
Ele executa duas tarefas principais:
1. Listen: Abre um socket TCP e começa a escutar na porta especificada
2. Serve: Usa o segundo parâmetro (app.Router) como o handler para processar as requisições HTTP recebidas.
A função bloqueia indefinidamente (entra em loop infinito) processando requisições.
Cada requisição é tratada em uma goroutine separada, permitindo concorrência automática.

Função log.Fatal: É uma função especial que registra um erro fatal no log, equivalente a log.Print() e 
Encerra o programa como os.Exit(1), que seria o código de saída indicando erro.
Raramente vai pegar algum erro, mas caso aconteça, pode ser dos seguintes cenários:
1. porta já em uso
2. permissões insuficientes
3. addr inválido
4. problemas de rede

Ela é usada como padrão para falhas fatais na inicialização.

*/

Paramos aqui em 27-11-25 !
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
pois transforma estruturas de dados Go em um formato universalmente legível por diferentes clientes
(navegadores, aplicações móveis, etc.).
*/

// Função de Escrita de erro, reutilizável da mesma forma que a função acima.
func sendError(w http.ResponseWriter, statusCode int, err error) { // Função que envia um erro.
	error_message := map[string]string{"error": err.Error()} // Map que encapsula a mensagem de erro em JSON, usando a interface error.
	sendResponse(w, statusCode, error_message) // Usa a função de Response no formato {"error": "mensagem do erro"}
}
/*
error_message é uma variável que armazena um map[string]string, ou seja, um mapa onde as chaves e valores são strings.
map{string}string --> é um mapa onde o tipo de chaves é string(primeiro valor) e os valores(segundo valor) também são strings.
No caso a chave seria "error" e o valor seria err.Error()

O uso da função sendResponse evita duplicação de código!
Todo o Marshalling de json seria repetido caso não tivessemos usado ela, ate pq a lógica seria a mesma ainda.

*/

// Método getProducts com entrada de writer(w) e request(r). Executa a função getProductsFromDB
func (app *App) getProducts(w http.ResponseWriter, r *http.Request) { // Seguindo o jargão de Go: w pra writer e r pra request
	products, err := getProductsFromDB(app.DB)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err)
		return
	}
	sendResponse(w, http.StatusOK, products)
}
/*
getProducts é um Handler HTTP que processa requisição GET pra listar todos os produtos.
O receiver (app *App), que é o request, tem como primeiro parâmetro o writer(w) e o segundo a request(r).

w http.ResponseWriter (Writer - Escritor de Resposta): é uma interface que permite escrever a resposta
HTTP de volta ao cliente.

type ResponseWriter interface {
    Header() Header              // Acessa headers HTTP
    Write([]byte) (int, error)   // Escreve corpo da resposta
    WriteHeader(statusCode int)  // Define código de status
}

r *http.Request: é um ponteiro para struct que contém todos os dados da requisição HTTP

type Request struct {
    Method     string              // "GET", "POST", etc.
    URL        *url.URL            // URL da requisição
    Header     Header              // Headers HTTP
    Body       io.ReadCloser       // Corpo da requisição, mas não usado aqui
    Form       url.Values          // Dados de formulário parseados, tb não usado aqui
}

Nesse caso, usamos dessa forma:
r.Method  // "GET"
r.URL     // "/products"
r.Header  // map[string][]string com headers

products, err := getProductsFromDB(app.DB) é a chamada da função que busca os produtos no banco de dados.
products é o resultado
err é o erro retornado, se houver.
app.DB passa a conexão com o banco de dados para a função.

r foi declarado mas não foi usado, então o ideal seria omitir ele usando o underline ou deixar para manter a
consistência com outros handlers.
O return logo depois de sendError é o early return, que já para a execução da função
logo quando encontra um erro.
A interface vazia da função sendResponse agora é populada pelo payload products

O uso dela se dá dessa forma:
app.getProducts(w, r)

Caso não tenha erro, a resposta será
sendResponse(w, http.StatusOK, products), ou seja, o status code de 200 junto do slice de produtos, por exemplo:

products = []product{
    {ID: 1, Name: "Notebook", Quantity: 10, Price: 2500.99},
    {ID: 2, Name: "Mouse", Quantity: 50, Price: 29.90},
}
*/

// Método getProduct - Handler que processa requisições GET /product/{id}
func (app *App) getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, err := strconv.Atoi(vars["id"]) // Converte o id da URL, que é sempre uma string, para inteiro.
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Errorf("invalid product ID"))
		return // Para a função imediatamente, em caso de erro.
	}

	p := product{ID: key} // Cria instância da struct product com apenas o campo ID preenchido
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

/*
mux.Vars(r) --> Função do pacote mux que extrai as variáveis da URL da requisição r.
O roteador gorilla/mux pega a URL (ex: /product/123) e, como a rota foi definida com /{id},
ele cria um mapa onde a chave é "id" e o valor é a string "123".

Se o usuário enviar um ID inválido, o tratamento de erro http.StatusBadRequest já retorna um 400,
indicando problema na requisição do cliente.

A conversão de string pra inteiro é necessária pq BD espera sempre tipo numérico.
A verificação de erro abaixo valida a conversão.
O return pode ser 400(cliente enviou dados invalidos), 404(recurso nao encontrado) ou 500(erro interno do servidor).

A struct product apos o preenchimento do ID apenas tem essa cara:
p = product{
    ID:       5,      // ← Preenchido explicitamente
    Name:     "",     // ← Zero value de string
    Quantity: 0,      // ← Zero value de int
    Price:    0.0     // ← Zero value de float64
}

err = p.getProduct(app.DB) -> Busca no BD via método getProduct(func (p *product) getProduct(db *sql.DB) error)
definido em module.go. Assim é encontrado e populado os outros campos da struct product.

sql.ErrNoRows -> Erro sentinal do pacote database/sql.
// Definido em database/sql/sql.go
var ErrNoRows = errors.New("sql: no rows in result set")

Qualquer outro erro, a sendError(w, http.StatusInternalServerError, err) entra como erro 500.
*/

// Handler que processa requisições POST /product, criando um novo produto no BD.
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
var p product -> Novamente é criado a variável p do tipo product(que tem como base uma struct) com valores nulos.

decoder := json.NewDecoder(r.Body)-> r.Body é do tipo io.ReadCloser, que contém o corpo de requisição HTTP.
Já o json.NewDecoder é uma função do pacote enncoding/json, que cria o decodificador que vai ler desse io.Reader

decoder.Decode(&p) -> lê o JSON do r.Body, parseia o JSON e preenche a struct product usando os ponteiros dos campos
O &p passa o endereço da variável para que o decoder possa modificá-la na struct original.

defer r.Body.Close() ->  garante que o corpo da requisição seja fechado adequadamente,
independentemente de como a função termine.
Fechamos r.Body para:
liberar recursos do sistema, como conexões de rede.
Evitar vazamentos(resource leaks)
Por ser boa prática fechar io.Closer

Se ela for criada com sucesso, é retornado 201 e o produto nasce em JSON.
*/

// Handler que processa requisições PUT /product/{id}, atualizando um produto existente no BD.
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

// Handler que processa requisições DELETE /product/{id}, deletando um produto existente no BD.
func (app *App) deleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
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
Aqui houve diarreia mental na Pryanka, pois ela não usou o erro sentinela sql.ErrNoRows, mas comparou string 
de erros distintas...

p := product{ID: key} --> cria uma instância mínima da struct product contendo apenas o ID.
Esta abordagem é eficiente pois não requer buscar todos os dados do produto no banco apenas para deletá-lo
- o ID é suficiente para a operação de exclusão.
Isso demonstra um uso otimizado dos recursos, evitando consultas desnecessárias.

Já o tratamento de erro é específico para o caso de "product not found", retornando 404.
E para qualquer outro, retorna 500.

A função de resposta envia um 200 OK com um JSON simples confirmando a exclusão.
Poderia ser um 204 No Content tb.

Estrutura Geral
1. Extrai ID da URL
2. Valida se ID é número válido
3. Cria struct product com apenas ID
4. Deleta do banco via método deleteProduct
5. Trata erros específicos (404 vs 500)
6. Retorna sucesso com mensagem
*/

// Configurador de rotas - Definição de todas as rotas da API
func (app *App) HandleRequests() {
	app.Router.HandleFunc("/products", app.getProducts).Methods("GET")
	app.Router.HandleFunc("/product/{id}", app.getProduct).Methods("GET")
	app.Router.HandleFunc("/product", app.createProduct).Methods("POST")
	app.Router.HandleFunc("/product/{id}", app.updateProduct).Methods("PUT")
	app.Router.HandleFunc("/product/{id}", app.deleteProduct).Methods("DELETE")
}

/*
O método HandleRequests():

1. Configura todas as rotas da API
2. Conecta paths aos handlers correspondentes
3. Especifica métodos HTTP permitidos
4. Define variáveis de rota ({id})
5. É chamado uma vez durante inicialização
6. Deixa tudo pronto para app.Run() começar a aceitar requisições

Exemplo da primeira rota:
Quando receber requisição GET para /products, chame app.getProducts(handler function)

Já o HandleFunc é um método do Gorillax Mux que reigstra uma rota.

*/