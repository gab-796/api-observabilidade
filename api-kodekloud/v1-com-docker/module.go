package main

// Esse arquivo foca na lógica de Banco de Dados.

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

func getProductsFromDB(db *sql.DB) ([]product, error) {
	query := "SELECT id, name, quantity, price FROM products"
	rows, err := db.Query(query) // Executa a query no banco de dados e coloca essa resposta como valor para a variavel rows.
	if err != nil {
		return nil, err
	}
	defer rows.Close() // Garante que as linhas serão fechadas após a execução da função, liberando assim espaço em memória.

	products := []product{}
	for rows.Next() { // Responsável por iterar sobre as linhas retornadas na consulta.
		var p product
		err := rows.Scan(&p.ID, &p.Name, &p.Quantity, &p.Price) // Caso alguma coluna tenha erro, ela será capturada aqui.
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (p *product) getProduct(db *sql.DB) error {
	query := ("SELECT name, quantity, price FROM products WHERE id = ?")
	row := db.QueryRow(query, p.ID)
	err := row.Scan(&p.Name, &p.Quantity, &p.Price)
	if err != nil {
		return err
	}
	return nil
}

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

func (p *product) updateProduct(db *sql.DB) error {
	query := ("UPDATE products SET name = ?, quantity = ?, price = ? WHERE id = ?")
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

func (p *product) deleteProduct(db *sql.DB) error {
	query := ("DELETE FROM products WHERE id = ?")
	result, err := db.Exec(query, p.ID)
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
