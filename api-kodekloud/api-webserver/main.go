package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Product struct {
	Id       int
	Name     string
	Quantity int
	Price    float64
}

var Products []Product // Inicia um slice de Products.

func homepage(w http.ResponseWriter, r *http.Request) { // função que será chamada quando o endpoint for chamado.
	fmt.Fprint(w, "Welcome to Homepage")  // Escreve na tela Welcome to Homepage ao acessar localhost:10000.
	log.Println("Endpoint Hit: homepage") // Imprime no console que o endpoint foi chamado quando eu bato em localhost:10000.
}

func returnAllProducts(w http.ResponseWriter, r *http.Request) {
	log.Println("Endpoint Hit: returnAllProducts")
	json.NewEncoder(w).Encode(Products) // Escreve no ResponseWriter todos os produtos.
}

func HandleRequests() {
	http.HandleFunc("/products", returnAllProducts)
	http.HandleFunc("/", homepage)            // qq request que o / for chamado, a função homepage será chamada.
	err := http.ListenAndServe(":10000", nil) // Inicia o servidor na porta 10000,e por padrão dop Go em localhost.
	if err != nil {
		log.Fatal("Erro ao iniciar o servidor: ", err) // Imprime o erro e sai
	}
}

func main() {
	Products = []Product{
		Product{Id: 1, Name: "Laptop", Quantity: 100, Price: 50000.00},
		Product{Id: 2, Name: "Mouse", Quantity: 200, Price: 500.00},
	}
	HandleRequests()
}

// Vamos usar o método ListenAndServe para criar um servidor web.
