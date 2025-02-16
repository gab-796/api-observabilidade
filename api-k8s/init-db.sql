CREATE DATABASE IF NOT EXISTS inventory;
USE inventory;

CREATE TABLE IF NOT EXISTS products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    quantity INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO products (name, price, quantity) VALUES
    ('Notebook', 3500.00, 10),
    ('Mouse', 150.00, 25),
    ('Teclado', 200.00, 15),
    ('Monitor', 1200.00, 8),
    ('Cadeira Gamer', 800.00, 5);