package main

import "log"

func main() {
	app := App{} // Cria uma instância da struct App, chamada de app, que nascerá com os valores dos ponteiros vazios: nil.
	err := app.Initialise() // Executa o método app.Initialise() e captura o valor de retorno dele por meio da variavel err.
	if err != nil { // Caso tenha erro, logue esse erro, usando a biblioteca padrão do go: log.
		log.Fatal(err)
	}
	app.HandleRequests() // Registra todas as rotas HTTP.
	app.Run(":10000") // Inicia o servidor na porta 10000. Go vai escutar em todas as interfaces de rede disponiveis na máquina(0.0.0.0)
}

/*
0. A chama app.Initialise() conecta ao banco, cria o roteador, ou seja, inicia a aplicação.

1. Ele funciona no POSTMAN tb, basta estar rodando aqui.

2. Para fazer o método POST funcionar, vc deve usar o endereço essa forma : `http://localhost:10000/product``, sem incluir o barra no final.
Isso é decorrente do uso do método StrictSlash(true) na criação do roteador.

3. Executando a criação da tabela e a adição de algumas linhas nele:
docker exec -i mysql-container mysql -u root -padmin learning < setup-inventory.sql

4. O Padrão usado foi Initialise --> Register Routes --> Run

5. Caso queiramos limitar a localhost, deveríamos colocar `app.Run("localhost:10000")`

---

Extras
Foi executado `go get github.com/gorilla/mux` e tb `go get github.com/go-sql-driver/mysql`

*/
