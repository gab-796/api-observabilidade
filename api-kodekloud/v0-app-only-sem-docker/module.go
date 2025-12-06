package main

import (
	"database/sql"
	"errors"
	"fmt"
)

type product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// Toda a lógica de Banco de Dados está nesse arquivo

// Função que pega todos os produtos da tabela e coloca na lista de productos(slice de structs product)
func getProductsFromDB(db *sql.DB) ([]product, error) {
	query := "SELECT id, name, quantity, price FROM products"
	rows, err := db.Query(query) // Executa a query no banco de dados e coloca essa resposta como valor para a variavel rows.
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []product{} // Cria um slice vazio para armazenar os produtos que serão encontrados.
	for rows.Next() {
		var p product
		err := rows.Scan(&p.ID, &p.Name, &p.Quantity, &p.Price)
		if err != nil {
			return nil, err // Se houver um erro ao escanear a linha, para tudo.
		}
		products = append(products, p)
	}
	return products, nil
}
/*
db.Query retorna:
*sql.Rows: Um objeto que permite iterar sobre as linhas do resultado. Não são os dados ainda!
err: Caso a consulta falhe.

Exemplos de erros que podem acontecer:
// Banco offline
err = "dial tcp: connection refused"

// Tabela não existe
err = "Table 'mydb.products' doesn't exist"

// Sintaxe SQL errada
err = "You have an error in your SQL syntax"

// Permissão negada
err = "Access denied for user"

A Mágica do 'defer':
O 'defer' agenda a execução de rows.Close() para o momento EXATO em que a função estiver prestes a terminar,
não importa como ela termine (seja por um 'return' de sucesso ou de erro).
Isso garante que a conexão com o banco seja sempre liberada, prevenindo vazamento de recursos.

for rows.Next() -> Método que avança o cursor para a próxima linha(mas não le os dados!)
Loop só para quando não tem mais para onde pular.

var p product -> Cria uma struct product temporária para cada linha
Estado inicial, a saber:
p = product{
    ID:       0,      // Zero value de int
    Name:     "",     // Zero value de string
    Quantity: 0,      // Zero value de int
    Price:    0.0     // Zero value de float64
}

err := rows.Scan(&p.ID, &p.Name, &p.Quantity, &p.Price) -> Escaneamento dos Dados

Método Scan aplicado no objeto row (rows.Scan()):
1. Lê os valores da linha atual (onde o cursor está)
2. Converte os tipos SQL para tipos Go
3. Copia os valores para os endereços fornecidos (&) na struct p

A conversão dos tipos é dessa forma:
Banco de dados (linha atual)
┌────┬───────────┬──────────┬─────────┐
│ id │ name      │ quantity │ price   │
├────┼───────────┼──────────┼─────────┤
│ 5  │ Notebook  │ 10       │ 2500.99 │
└────┴───────────┴──────────┴─────────┘
  ↓       ↓          ↓          ↓
  │       │          │          │
Scan(&p.ID, &p.Name, &p.Quantity, &p.Price)
  │       │          │          │
  ↓       ↓          ↓          ↓
┌────┬───────────┬──────────┬─────────┐
│ ID │ Name      │ Quantity │ Price   │
├────┼───────────┼──────────┼─────────┤
│ 5  │ "Notebook"│ 10       │ 2500.99 │
└────┴───────────┴──────────┴─────────┘
Struct product em Go

Conversões automáticas
MySQL INT       → Go int
MySQL VARCHAR   → Go string
MySQL INT       → Go int
MySQL DECIMAL   → Go float64

Lista de erros possíveis do Scan:
1. Tipos incompatíveis
Banco tem VARCHAR, Go espera int
err = "sql: Scan error... converting string to int"

2. Número errado de argumentos
rows.Scan(&p.ID)  // Falta argumentos
err = "sql: expected 4 destination arguments in Scan, not 1"

3. Valor NULL no banco
Se coluna permite NULL e Go não trata
err = "sql: Scan error... converting NULL to int"
*/


// Método para obter apenas 1 produto com id específico
func (p *product) getProduct(db *sql.DB) error {
	query := ("SELECT name, quantity, price FROM products WHERE id = ?")
	row := db.QueryRow(query, p.ID)
	err := row.Scan(&p.Name, &p.Quantity, &p.Price)
	if err != nil { // Os tipos de erro são: erro de conexão, de conversão de tipo ou especial(sql.ErrNoRows)
		return err  // quando não tem linha com o ID fornecido.
	}
	return nil
}

/*
O ? é um placeholder, que será substituido pelo p.ID, da linha abaixo:
row := db.QueryRow(query, p.ID)

O uso do placeholder evita SQL Injection(caso fosse tentado concatenamento), como abaixo:
query := fmt.Sprintf("SELECT ... WHERE id = %d", p.ID)

row := db.QueryRow(query, p.ID) ->
row é objeto *.sql.Row
db -> ponteiro *sql.DB
QueryRow_ método QueryRow do pacote SQL(go doc sql.QueryRow)

db.QueryRow é um método do pacote database/sql para buscar uma única linha
estrutura interna de row:
row = &sql.Row{
    // Campos internos (não acessamos)
    err:   nil,  // Guardará erro se houver
    rows:  nil,  // Cursor interno
}


err := row.Scan(&p.Name, &p.Quantity, &p.Price) ->
row -> objeto *sql.Row
Scan -> método Scan
argumentos são os ponteiros

O Scan executa a query no banco, lê a linha retornada, convert tipos SQL em Go
preenche as variáveis através dos ponteiros e retorna erro se algo der errado.

ANTES do Scan:
p = product{
    ID:       5,      ← Já preenchido (usado na query)
    Name:     "",     ← Zero value
    Quantity: 0,      ← Zero value
    Price:    0.0     ← Zero value
}
DEPOIS do Scan:
p = product{
    ID:       5,
    Name:     "Notebook",   ← PREENCHIDO
    Quantity: 10,           ← PREENCHIDO
    Price:    2500.99       ← PREENCHIDO
}
*/

// Método que cria produto na tabela
func (p *product) createProduct(db *sql.DB) error {
	query := ("INSERT INTO products(name, quantity, price) VALUES(?, ?, ?)")
	result, err := db.Exec(query, p.Name, p.Quantity, p.Price)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = int(id)
	return nil
}

/*
Handler cria struct com dados do JSON (SEM ID)
p := product{
    ID:       0,           // ← Zero value (será preenchido depois)
    Name:     "Mouse",     // ← Do JSON
    Quantity: 50,          // ← Do JSON
    Price:    29.90        // ← Do JSON
}
MySQL gera um novo ID automaticamente (AUTO_INCREMENT), dai não precisa mencionar o id na query.

result, err := db.Exec(query, p.Name, p.Quantity, p.Price) ->
result: variavel que armazena metadados da operação
db -> ponteiro para conexão *.sql.DB
Exec -> método Exec do pacote SQL
db.Exec-> Método do pacote database/sql(go doc sql.Exec)) usado para executar queries que
modificam dados (INSERT, UPDATE, DELETE)
sql.Result -> interface que contém metadados da operação executada (não os dados em si).

Tratamento de erro, possíveis erros:
1. Banco offline
err = "dial tcp 127.0.0.1:3306: connect: connection refused"
2. Constraint violation (ex: nome duplicado se tiver UNIQUE)
err = "Error 1062: Duplicate entry 'Mouse' for key 'name'"
3. Coluna não existe
err = "Error 1054: Unknown column 'nome' in 'field list'"
4. Tipo incompatível
err = "Error 1366: Incorrect integer value: 'abc' for column 'quantity'"
5. Campo NOT NULL vazio
err = "Error 1364: Field 'name' doesn't have a default value"
6. Permissão negada
err = "Error 1142: INSERT command denied to user 'app'@'localhost'"
7. Tabela não existe
err = "Error 1146: Table 'mydb.produtos' doesn't exist"

Caso haja erro no insert, ele para a função.

id, err := result.LastInsertId() ->
id -> id gerado pelo bd(tipo int64)
err -> variável de erro reutilizada!
result
LastInsertId() -> método da interface sql.Result,que retorna o ID gerado pelo último INSERT
executado por aquela conexão.(ID do aunto_increment).
type Result interface {
    LastInsertId() (int64, error)
    //              └──┬─┘
    //          ID do AUTO_INCREMENT
    RowsAffected() (int64, error)
}

p.ID = int(id) -> Preenche ID na struct
p -> receiver (ponteiro *product)
ID -> Campo ID da struct product
int -> conversão de int64 para int
id -> variável id do LastInsertId() como tipo int64

Estado da struct
// ANTES de createProduct
p = product{
    ID:       0,          // ← Zero value
    Name:     "Mouse",
    Quantity: 50,
    Price:    29.90
}

// db.Exec() insere no banco
// MySQL gera AUTO_INCREMENT = 6
// LastInsertId() retorna 6

// DEPOIS de p.ID = int(id)
p = product{
    ID:       6,          // ← PREENCHIDO!
    Name:     "Mouse",
    Quantity: 50,
    Price:    29.90
}
*/

// Método para atualizar uma linha da tabela
func (p *product) updateProduct(db *sql.DB) error {
	query := ("UPDATE products SET name = ?, quantity = ?, price = ? WHERE id = ?")
	result, err := db.Exec(query, p.Name, p.Quantity, p.Price, p.ID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 { // Exibe a mensagem de erro quando tentamos alterar um id que não existe.
		return errors.New("no such rows to update")
	}
	return nil
}

/*
query := ("UPDATE products SET name = ?, quantity = ?, price = ? WHERE id = ?") -> Atualiza campos
específicos de uma linha que corresponde ao ID fornecido
Exemplo:
// Struct contém:
p.ID       = 5
p.Name     = "Mouse RGB"
p.Quantity = 100
p.Price    = 49.90

// SQL executado pelo driver:
UPDATE products
SET name = 'Mouse RGB', quantity = 100, price = 49.90
WHERE id = 5

result, err := db.Exec(query, p.Name, p.Quantity, p.Price, p.ID) ->
result -> Objeto sql.Result com metadados da operação UPDATE
type Result interface {
    LastInsertId() (int64, error)  // Não usado em UPDATE
    RowsAffected() (int64, error)  // ← USADO! Quantas linhas mudaram
}
db.Exec->  envia UPDATE para MySQL

Possíveis erros do Exec:
1. Banco offline
err = "dial tcp: connection refused"
2. Coluna não existe
err = "Unknown column 'nome' in 'field list'"
3. Tipo incompatível
err = "Incorrect integer value: 'abc' for column 'quantity'"
4. Permissão negada
err = "UPDATE command denied to user"
5. Tabela não existe
err = "Table 'mydb.produtos' doesn't exist"
6. Constraint violation (ex: nome duplicado com UNIQUE)
err = "Duplicate entry 'Mouse RGB' for key 'name'"

rowsAffected, err := result.RowsAffected() ->
rowsAffected -> número de linhas afetadas (tipo int64)
err -> variável de erro reutilizada!
result -> objeto sql.Result
RowsAffected() -> método da interface sql.Result que retorna o número de linhas afetadas(go doc sql.Result)

Possíveis erros de RowsAffected():
1. Driver não suporta (alguns drivers antigos)
    err = "RowsAffected is not supported by this driver"
2. Conexão perdida entre Exec e RowsAffected
    err = "invalid connection"

E finalmente, o tratamento de erro no caso de nenhuma linha ter sido alterada:
if rowsAffected == 0 { // Exibe a mensagem de erro quando tentamos alterar um id que não existe.
		return errors.New("no such rows to update")
	}

return errors.New ->
errors -> cria novo erro
New -> Função New do pacote errors(go doc errors.New)
"no such rows to update" -> msg de erro

É necessário pq update em ID inexistente não gera erro no Exec!
Sem essa verificação, o handler retornaria 200 OK quando deveria retornar 404 Not Found!

return nil -> Retorno de sucesso, ou seja, linha encontrada e atualizada.
*/

// Método para deletar linha na tabela
func (p *product) deleteProduct(db *sql.DB) error {
	query := ("DELETE FROM products WHERE id = ?")
	result, err := db.Exec(query, p.ID) // Usa placeholder para evitar SQL Injection
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

/*
Exatamente igual aos anteriores, porém é um caminho sem volta, deletou a linha, já era.

Possiveis erros no db.Exec:
1. Banco offline
err = "dial tcp: connection refused"
2. Tabela não existe
err = "Table 'mydb.products' doesn't exist"
3. Permissão negada
err = "DELETE command denied to user"
4. Foreign key constraint (se produto está em pedidos)
err = "Cannot delete or update a parent row: a foreign key constraint fails"
5. Sintaxe SQL errada (raro com placeholders)
err = "You have an error in your SQL syntax"


*/