package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql" // Importa o driver mysql.
)

func checkError(err error) {
	if err != nil {
		log.Fatalln(err) // Se err for diferente de nil, o programa para.
	}
}

type Data struct {
	Id   int
	Name string
}

func main() {
	connectionString := fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s", DBUser, DBPassword, DBName) // Cria a string de conexão. 3306 é a porta padrão do MySQL.
	db, err := sql.Open("mysql", connectionString)                                              // Abre a conexão com o banco de dados.
	checkError(err)                                                                             // Verifica se houve erro na conexão.
	defer db.Close()                                                                            // Fecha a conexão com o banco de dados.

	result, err := db.Exec("INSERT INTO data values(4, 'xyz')") // Executa a query no banco de dados.
	checkError(err)                                             // Verifica se houve erro na query.
	lastInsertedId, err := result.LastInsertId()                // Pega o último id inserido.
	fmt.Println("Last inserted id: ", lastInsertedId)           // Imprime o último id inserido.
	checkError(err)                                             // Verifica se houve erro na query.
	RowsAffected, err := result.RowsAffected()                  // Pega o número de linhas afetadas.
	fmt.Println("Rows Affected: ", RowsAffected)                // Imprime o número de linhas afetadas.
	checkError(err)

	rows, err := db.Query("SELECT * FROM data") // Executa a query no banco de dados.
	checkError(err)                             // Verifica se houve erro na query.
	defer rows.Close()                          // Fecha as linhas após leitura

	for rows.Next() { // Rodará enquanto houverem linhas para serem lidas.
		var data Data                                        // Cria uma variável do tipo Data.
		err := rows.Scan(&data.Id, &data.Name)               // Lê os valores das colunas e armazena em data.
		checkError(err)                                      // Verifica se houve erro na leitura.
		fmt.Printf("ID: %d, Nome: %s\n", data.Id, data.Name) // Imprime a variável data.
	}
}

/* Preparando o mysql:
1. rodando o mysql dentro de um container docker:
docker run -d --name mysql-container -e MYSQL_ROOT_PASSWORD=admin -e MYSQL_DATABASE=learning -p 3306:3306 mysql:8.0

2. Executando a criação da tabela e a adição de algumas linhas nele:
`docker exec -i mysql-container mysql -u root -padmin learning < setup.sql`

3. rode como `go run .`` dentro da pasta onde estao os arquivos go.

4. Ao reiniciar o PC, o container do sql pode ter ficado parado, então apenas reiniicie o container com `docker start mysql-container`

5. Para acessar o banco de dados, use o comando `docker exec -it mysql-container mysql -u root -padmin learning`
O nome do BD é learning.

6. Para ver as tabelas, use o comando `show tables;`

7. Para ver os dados da tabela, use o comando `select * from data;`
*/
