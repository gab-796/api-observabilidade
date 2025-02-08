# api-observabilidade
API simples em Go

1. Executamos `go get github.com/gorilla/mux`
2. Instalando o pacote do mysql: `go get github.com/go-sql-driver/mysql`
3. Pra rodar todos os arquivos .go ao mesmo tempo dê `go run .` na pasta my-inventory.

## Configurando o Mysql

1. rodando o mysql dentro de um container docker:
`docker run -d --name mysql-container -e MYSQL_ROOT_PASSWORD=admin -e MYSQL_DATABASE=learning -p 3306:3306 mysql:8.0`

2. Executando a criação da tabela e a adição de algumas linhas nele:
`docker exec -i mysql-container mysql -u root -padmin learning < setup-inventory.sql`

3. Ao reiniciar o PC, o container do sql pode ter ficado parado, então apenas reiniicie o container com `docker start mysql-container`

4. Para acessar o banco de dados, use o comando `docker exec -it mysql-container mysql -u root -padmin learning`
O nome do BD é learning.

5. Para ver as tabelas, use o comando `show tables;`
Haverá uma tabela chamada data

6. Para ver os dados da tabela, use o comando `select * from data;`
Deverá ter 5 linhas, de acordo com o arquivo de `setup-inventory.sql`
