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
	log.Debug("Iniciando getProductsFromDB") // Log de depuração
	query := "SELECT id, name, quantity, price FROM products"
	rows, err := db.Query(query) // Executa a query no banco de dados e coloca essa resposta como valor para a variavel rows.
	if err != nil {
		log.WithError(err).Error("Erro ao executar query")
		return nil, fmt.Errorf("erro ao buscar produtos: %w", err)
	}
	defer rows.Close() // Garante que as linhas serão fechadas após a execução da função, liberando assim espaço em memória.

	products := []product{}
	for rows.Next() { // Responsável por iterar sobre as linhas retornadas na consulta.
		var p product
		err := rows.Scan(&p.ID, &p.Name, &p.Quantity, &p.Price) // Caso alguma coluna tenha erro, ela será capturada aqui.
		if err != nil {
			log.WithError(err).Error("Erro ao ler os dados da linha")
			return nil, fmt.Errorf("erro ao ler dados do produto: %w", err)
		}
		products = append(products, p)
	}
	log.WithField("num_products", len(products)).Debug("Produtos encontrados") // Log de depuração com informações adicionais
	return products, nil
}

func (p *product) getProduct(db *sql.DB) error {
	log.WithField("product_id", p.ID).Debug("Iniciando getProduct") // Log de depuração com o ID do produto
	query := ("SELECT name, quantity, price FROM products WHERE id = ?")
	row := db.QueryRow(query, p.ID)
	err := row.Scan(&p.Name, &p.Quantity, &p.Price)
	if err != nil {
		log.WithError(err).WithField("product_id", p.ID).Error("Erro ao buscar produto") // Log de erro com detalhes
		return fmt.Errorf("erro ao buscar produto: %w", err)
	}
	log.WithField("product_id", p.ID).Debug("Produto encontrado") // Log de depuração com o ID do produto
	return nil
}

func (p *product) createProduct(db *sql.DB) error {
	log.WithField("product_name", p.Name).Debug("Iniciando createProduct") // Log de depuração com o nome do produto

	query := "INSERT INTO products(name, quantity, price) VALUES(?,?,?)"
	result, err := db.Exec(query, p.Name, p.Quantity, p.Price)
	if err != nil {
		log.WithError(err).Error("Erro ao inserir produto") // Log de erro com detalhes
		return fmt.Errorf("erro ao criar produto: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.WithError(err).Error("Erro ao obter ID do produto inserido") // Log de erro com detalhes
		return fmt.Errorf("erro ao obter ID do produto: %w", err)
	}
	p.ID = int(id)

	log.WithField("product_id", p.ID).Debug("Produto criado") // Log de depuração com o ID do produto
	return nil
}

func (p *product) updateProduct(db *sql.DB) error {
	log.WithField("product_id", p.ID).Debug("Iniciando updateProduct") // Log de depuração com o ID do produto

	query := "UPDATE products SET name =?, quantity =?, price =? WHERE id =?"
	result, err := db.Exec(query, p.Name, p.Quantity, p.Price, p.ID)
	if err != nil {
		log.WithError(err).Error("Erro ao atualizar produto") // Log de erro com detalhes
		return fmt.Errorf("erro ao atualizar produto: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.WithError(err).Error("Erro ao obter número de linhas afetadas") // Log de erro com detalhes
		return fmt.Errorf("erro ao obter número de linhas afetadas: %w", err)
	}
	if rowsAffected == 0 {
		log.WithField("product_id", p.ID).Warn("Nenhum produto atualizado") // Log de warning
		return errors.New("nenhum produto atualizado")
	}

	log.WithField("product_id", p.ID).Debug("Produto atualizado") // Log de depuração com o ID do produto
	return nil
}

func (p *product) deleteProduct(db *sql.DB) error {
	log.WithField("product_id", p.ID).Debug("Iniciando deleteProduct") // Log de depuração com o ID do produto

	query := "DELETE FROM products WHERE id =?"
	result, err := db.Exec(query, p.ID)
	if err != nil {
		log.WithError(err).Error("Erro ao excluir produto") // Log de erro com detalhes
		return fmt.Errorf("erro ao excluir produto: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.WithError(err).Error("Erro ao obter número de linhas afetadas") // Log de erro com detalhes
		return fmt.Errorf("erro ao obter número de linhas afetadas: %w", err)
	}
	if rowsAffected == 0 {
		log.WithField("product_id", p.ID).Warn("Nenhum produto excluído") // Log de warning
		return fmt.Errorf("produto não encontrado")
	}

	log.WithField("product_id", p.ID).Debug("Produto excluído") // Log de depuração com o ID do produto
	return nil
}

// countProducts conta o número total de produtos no banco de dados.
func countProducts(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar produtos: %w", err) // Usa %w
	}
	return count, nil
}
