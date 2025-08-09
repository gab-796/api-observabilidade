package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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

func getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r) //
	key := vars["id"]

	id, err := strconv.Atoi(key) // Converte a string key para um inteiro
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest) // Retorna um erro se a conversão falhar
		return                                              // Importante: Sai da função após o erro
	}

	for _, product := range Products {
		if product.Id == id { // Agora a comparação é entre inteiros
			json.NewEncoder(w).Encode(product)
			return // Encontrou o produto, sai da função
		}
	}
	http.Error(w, "Produto não encontrado", http.StatusNotFound) // Retorna um erro se o produto não for encontrado
}

func HandleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true) // Cria uma nova instância de um roteador.
	myRouter.HandleFunc("/products", returnAllProducts)
	myRouter.HandleFunc("/product/{id}", getProduct)
	myRouter.HandleFunc("/", homepage)      // qq request que o / for chamado, a função homepage será chamada.
	http.ListenAndServe(":10000", myRouter) // Inicia o servidor na porta 10000,e por padrão do Go em localhost.
}

func main() {
	Products = []Product{
		Product{Id: 1, Name: "Laptop", Quantity: 100, Price: 50000.00},
		Product{Id: 2, Name: "Mouse", Quantity: 200, Price: 500.00},
	}
	HandleRequests()
}

// Vamos usar o método ListenAndServe para criar um servidor web.

// https://learn.kodekloud.com/user/courses/advanced-golang/module/483ddd82-96d2-43d5-a9a8-e27e8cdb064d/lesson/710dfc31-27ad-4ea2-b5a1-791a10dc1c60

// Mux Router finalzinho do video. mas não está dando certo o product id. vamos ver se com o mysql vai dar certo.
