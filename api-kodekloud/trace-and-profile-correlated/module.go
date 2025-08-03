package main

import (
	"context" // Importar o pacote context
	"database/sql"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
)


// Struct product (sem alterações)
type product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// getProductsFromDB busca todos os produtos, agora com contexto e profiling
func getProductsFromDB(ctx context.Context, db *sql.DB) ([]product, error) {
	return executeWithProfiling(ctx, "get_products", 0, func(profileCtx context.Context) ([]product, error) {
		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component": "database",
			"operation": "get_products",
		}).Debug("Iniciando getProductsFromDB")

		query := "SELECT id, name, quantity, price FROM products"
		rows, err := db.QueryContext(profileCtx, query)
		if err != nil {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component": "database",
				"operation": "get_products",
				"error":     err.Error(),
			}).Error("Erro ao executar QueryContext em getProductsFromDB")
			return nil, fmt.Errorf("erro ao buscar produtos: %w", err)
		}
		defer rows.Close()

		products := []product{}
		for rows.Next() {
			var p product
			err := rows.Scan(&p.ID, &p.Name, &p.Quantity, &p.Price)
			if err != nil {
				logrus.WithContext(profileCtx).WithFields(logrus.Fields{
					"component": "database",
					"operation": "get_products",
					"error":     err.Error(),
				}).Error("Erro ao ler os dados da linha em getProductsFromDB")
				return nil, fmt.Errorf("erro ao ler dados do produto: %w", err)
			}
			products = append(products, p)
		}

		if err = rows.Err(); err != nil {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component": "database",
				"operation": "get_products",
				"error":     err.Error(),
			}).Error("Erro durante a iteração das linhas em getProductsFromDB")
			return nil, fmt.Errorf("erro ao iterar sobre produtos: %w", err)
		}

		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":    "database",
			"operation":    "get_products",
			"num_products": len(products),
		}).Debug("Produtos encontrados em getProductsFromDB")
		return products, nil
	})
}

// getProduct busca um produto pelo ID, agora com contexto e profiling
func (p *product) getProduct(ctx context.Context, db *sql.DB) error {
	return ProfiledDatabaseOperation(ctx, "get_product", p.ID, func(profileCtx context.Context) error {
		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":  "database",
			"operation": "get_product",
			"product_id": p.ID,
		}).Debug("Iniciando getProduct")

		query := "SELECT name, quantity, price FROM products WHERE id = ?"
		row := db.QueryRowContext(profileCtx, query, p.ID)
		err := row.Scan(&p.Name, &p.Quantity, &p.Price)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logrus.WithContext(profileCtx).WithFields(logrus.Fields{
					"component":  "database",
					"operation": "get_product",
					"product_id": p.ID,
				}).Warn("Produto não encontrado")
				return sql.ErrNoRows
			}
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component":  "database",
				"operation": "get_product",
				"product_id": p.ID,
				"error":     err.Error(),
			}).Error("Erro ao buscar produto")
			return fmt.Errorf("erro ao buscar produto %d: %w", p.ID, err)
		}

		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":  "database",
			"operation": "get_product",
			"product_id": p.ID,
		}).Debug("Produto encontrado")
		return nil
	})
}

// createProduct cria um novo produto, agora com contexto e profiling
func (p *product) createProduct(ctx context.Context, db *sql.DB) error {
	return ProfiledDatabaseOperation(ctx, "create_product", 0, func(profileCtx context.Context) error {
		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":  "database",
			"operation": "create_product",
			"product_name": p.Name,
		}).Debug("Iniciando createProduct")
		query := "INSERT INTO products(name, quantity, price) VALUES(?,?,?)"
		// Usa ExecContext para passar o contexto
		result, err := db.ExecContext(profileCtx, query, p.Name, p.Quantity, p.Price)
		if err != nil {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component":  "database",
				"operation": "create_product",
				"error":     err.Error(),
			}).Error("Erro ao executar ExecContext em createProduct")
			return fmt.Errorf("erro ao criar produto: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			// Este erro geralmente não é relacionado ao contexto, mas logamos mesmo assim
			logrus.WithContext(profileCtx).WithError(err).Error("Erro ao obter ID do produto inserido em createProduct")
			return fmt.Errorf("erro ao obter ID do produto: %w", err)
		}
		p.ID = int(id)

		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":  "database",
			"operation": "create_product",
			"product_id": p.ID,
		}).Debug("Produto criado em createProduct")
		return nil
	})
}

// updateProduct atualiza um produto, agora com contexto e profiling
func (p *product) updateProduct(ctx context.Context, db *sql.DB) error {
	return ProfiledDatabaseOperation(ctx, "update_product", p.ID, func(profileCtx context.Context) error {
		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":  "database",
			"operation": "update_product",
			"product_id": p.ID,
		}).Debug("Iniciando updateProduct")
		query := "UPDATE products SET name =?, quantity =?, price =? WHERE id =?"
		// Usa ExecContext para passar o contexto
		result, err := db.ExecContext(profileCtx, query, p.Name, p.Quantity, p.Price, p.ID)
		if err != nil {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component":  "database",
				"operation": "update_product",
				"product_id": p.ID,
				"error":     err.Error(),
			}).Error("Erro ao executar ExecContext em updateProduct")
			return fmt.Errorf("erro ao atualizar produto %d: %w", p.ID, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component":  "database",
				"operation": "update_product",
				"product_id": p.ID,
				"error":     err.Error(),
			}).Error("Erro ao obter número de linhas afetadas em updateProduct")
			// Pode não ser fatal, mas retornamos o erro para consistência
			return fmt.Errorf("erro ao obter linhas afetadas para produto %d: %w", p.ID, err)
		}
		if rowsAffected == 0 {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component":  "database",
				"operation": "update_product",
				"product_id": p.ID,
			}).Warn("Nenhum produto atualizado em updateProduct (ID não encontrado?)")
			// Retorna ErrNoRows para indicar que o ID não foi encontrado
			return sql.ErrNoRows
		}

		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":  "database",
			"operation": "update_product",
			"product_id": p.ID,
		}).Debug("Produto atualizado em updateProduct")
		return nil
	})
}

// deleteProduct deleta um produto, agora com contexto e profiling
func (p *product) deleteProduct(ctx context.Context, db *sql.DB) error {
	return ProfiledDatabaseOperation(ctx, "delete_product", p.ID, func(profileCtx context.Context) error {
		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":  "database",
			"operation": "delete_product",
			"product_id": p.ID,
		}).Debug("Iniciando deleteProduct")
		query := "DELETE FROM products WHERE id =?"
		// Usa ExecContext para passar o contexto
		result, err := db.ExecContext(profileCtx, query, p.ID)
		if err != nil {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component":  "database",
				"operation": "delete_product",
				"product_id": p.ID,
				"error":     err.Error(),
			}).Error("Erro ao executar ExecContext em deleteProduct")
			return fmt.Errorf("erro ao excluir produto %d: %w", p.ID, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component":  "database",
				"operation": "delete_product",
				"product_id": p.ID,
				"error":     err.Error(),
			}).Error("Erro ao obter número de linhas afetadas em deleteProduct")
			return fmt.Errorf("erro ao obter linhas afetadas para produto %d: %w", p.ID, err)
		}
		if rowsAffected == 0 {
			logrus.WithContext(profileCtx).WithFields(logrus.Fields{
				"component":  "database",
				"operation": "delete_product",
				"product_id": p.ID,
			}).Warn("Nenhum produto excluído em deleteProduct (ID não encontrado?)")
			// Retorna ErrNoRows para indicar que o ID não foi encontrado
			return sql.ErrNoRows
		}

		logrus.WithContext(profileCtx).WithFields(logrus.Fields{
			"component":  "database",
			"operation": "delete_product",
			"product_id": p.ID,
		}).Debug("Produto excluído em deleteProduct")
		return nil
	})
}

// countProducts conta os produtos, agora com contexto e profiling
func countProducts(ctx context.Context, db *sql.DB) (int, error) {
	return executeCountWithProfiling(ctx, "count_products", func(profileCtx context.Context) (int, error) {
		var count int
		query := "SELECT COUNT(*) FROM products"
		// Usa QueryRowContext para passar o contexto
		err := db.QueryRowContext(profileCtx, query).Scan(&count)
		if err != nil {
			logrus.WithContext(profileCtx).WithError(err).Error("Erro ao executar QueryRowContext ou Scan em countProducts")
			return 0, fmt.Errorf("erro ao contar produtos: %w", err)
		}
		return count, nil
	})
}

// --- Funções auxiliares para profiling ---

// executeWithProfiling executa uma função que retorna []product com profiling contextual
func executeWithProfiling(ctx context.Context, operation string, productID int, fn func(context.Context) ([]product, error)) ([]product, error) {
	var result []product
	var err error
	
	profileErr := ProfiledDatabaseOperation(ctx, operation, productID, func(profileCtx context.Context) error {
		result, err = fn(profileCtx)
		return err
	})
	
	if profileErr != nil {
		return nil, profileErr
	}
	
	return result, err
}

// executeCountWithProfiling executa uma função que retorna int com profiling contextual
func executeCountWithProfiling(ctx context.Context, operation string, fn func(context.Context) (int, error)) (int, error) {
	var result int
	var err error
	
	profileErr := ProfiledDatabaseOperation(ctx, operation, 0, func(profileCtx context.Context) error {
		result, err = fn(profileCtx)
		return err
	})
	
	if profileErr != nil {
		return 0, profileErr
	}
	
	return result, err
}
