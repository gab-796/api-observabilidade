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

// Função que pega todos os produtos da tabela e colocar na lista chamada products
func getProductsFromDB(db *sql.DB) ([]product, error) {
	query := "SELECT id, name, quantity, price FROM products"
	rows, err := db.Query(query) // Executa a query no banco de dados e coloca essa resposta como valor para a variavel rows.
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []product{} // Cria um slice para armazenar os produtos que forem sendo encontrados.
	for rows.Next() { // Responsável por iterar sobre as linhas retornadas na consulta.
		var p product // Cria um struct chamado product, temporário pra cada linha.
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
rows: Um objeto que permite iterar sobre as linhas do resultado. Não são os dados ainda!
err: Caso a consulta falhe.

A Mágica do 'defer':
O 'defer' agenda a execução de rows.Close() para o momento EXATO em que a função estiver prestes a terminar, 
não importa como ela termine (seja por um 'return'de sucesso ou de erro).
Isso garante que a conexão com o banco seja sempre liberada, prevenindo vazamento de recursos.

Método Scan aplicado no objeto row (rows.Scan()): escaneia os valores das colunas da linha atual( id, name, quantity, price) e 
os copia para os campos da struct p.
Usamos os ponteiros (&p.ID, &p.Name, etc.) para que a função Scan possa modificar os valores da struct.

*/

// Método para obter apenas 1 produto
func (p *product) getProduct(db *sql.DB) error {
	query := ("SELECT name, quantity, price FROM products WHERE id = ?") // O ? é um placeholder, que será substituido pelo p.ID
	row := db.QueryRow(query, p.ID) // Função db.QueryRow retorna apenas o objeto row.
	err := row.Scan(&p.Name, &p.Quantity, &p.Price) // A consulta é executada nesse ponto e caso haja erro, el retorna o erro
	if err != nil { // Os tipos de erro são: erro de conexão, de conversão de tipo ou especial(sql.ErrNoRows),
		return err  // quando não tem linha com o ID fornecido.
	}
	return nil
}


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

// Método para atualizar uma linha da tabela
func (p *product) updateProduct(db *sql.DB) error {
	query := ("UPDATE products SET name = ?, quantity = ?, price = ? WHERE id = ?") // placeholders evitando SQL Injection
	result, err := db.Exec(query, p.Name, p.Quantity, p.Price, p.ID) // Executa a query no banco de dados e pega o resultado.
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected() // Pega o número de linhas afetadas, para caso usemos o método PUT num id que não existe.
	if err != nil {
		return err
	}
	if rowsAffected == 0 { // Exibe a mensagem de erro quando tentamos alterar um id que não existe.
		return errors.New("no such rows to update")
	}
	return nil
}

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
db.Exec é um método do pacote database/sql que tem como retorno um objeto do tipo `sql.Result`
Esse objeto não contém dados das linhas, mas sim metadados sobre a operação que foi executada.
Para saber mais: `go doc sql.Exec`

O objeto result é utilizado com o método RowsAffected() para obter o número de linhas afetadas pela operação.
Como a operação foi DELETE, ele vai registrar quantas linhas foram deletadas.

O receiver de todos os métodos é p *product.
*/