#!/bin/bash

mysql -u root -p"${MYSQL_ROOT_PASSWORD}" <<EOF
CREATE DATABASE IF NOT EXISTS inventory;
USE inventory;

CREATE TABLE IF NOT EXISTS products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    quantity INT NOT NULL
);

-- Insere os produtos apenas se a tabela estiver vazia
INSERT INTO products (name, price, quantity)
SELECT * FROM (SELECT 'Notebook', 3500.00, 10 UNION ALL
               SELECT 'Mouse', 150.00, 25 UNION ALL
               SELECT 'Teclado', 200.00, 15 UNION ALL
               SELECT 'Monitor', 1200.00, 8 UNION ALL
               SELECT 'Cadeira Gamer', 800.00, 5) AS tmp
WHERE NOT EXISTS (SELECT 1 FROM products LIMIT 1);
EOF