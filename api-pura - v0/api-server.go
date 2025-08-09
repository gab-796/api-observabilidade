package main

import (
	"fmt"
	"log"
	"net/http"
)

func homepage(w http.ResponseWriter, r *http.Request) { // função que será chamada quando o endpoint for chamado.
	fmt.Fprint(w, "Welcome to Homepage")  // Escreve na tela Welcome to Homepage ao acessar localhost:10000.
	fmt.Println("Endpoint Hit: homepage") // Imprime no console que o endpoint foi chamado quando eu bato em localhost:10000.
}

func main() {
	http.HandleFunc("/", homepage)            // qq request que o / for chamado, a função homepage será chamada.
	http.ListenAndServe(":10000", nil)        // Inicia o servidor na porta 10000,e por padrão dop Go em localhost.
	err := http.ListenAndServe(":10000", nil) // Captura o erro
	if err != nil {
		log.Fatal("Erro ao iniciar o servidor: ", err) // Imprime o erro e sai
	}
}

// Vamos usar o método ListenAndServe para criar um servidor web.
