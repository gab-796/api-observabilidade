package main

import "log"

func main() {
	app := App{}
	err := app.Initialise()
	if err != nil {
		log.Fatal(err)
	}
	app.HandleRequests()
	app.Run(":10000")
}

/* Foi executado go get github.com/gorilla/mux e tb go get github.com/go-sql-driver/mysql

1.Ele funciona no POSTMAN tb, basta estar rodando aqui.
2. Para fazer o método POST funcionar, vc deve usar o endereço essa forma : `http://localhost:10000/product``, sem incluir o barra no final.
Isso é decorrente do uso do método StrictSlash(true) na criação do roteador.


Parei no PUT Method:
https://learn.kodekloud.com/user/courses/advanced-golang/module/483ddd82-96d2-43d5-a9a8-e27e8cdb064d/lesson/59a5575f-7f37-4af9-aabb-b76da61db32a?autoplay=true

*/
