# api-observabilidade
API simples em Go

1. Executamos `go get github.com/gorilla/mux`
2. Instalando o pacote do mysql: `go get github.com/go-sql-driver/mysql`
3. Pra rodar todos os arquivos .go ao mesmo tempo dê `go run .` na pasta my-inventory.

## Configurando o Mysql manualmente

1. Rodando o mysql com o database inventory dentro de um container chamado mysql-container:
`docker run -d --name mysql-container -e MYSQL_ROOT_PASSWORD=admin -e MYSQL_DATABASE=inventory -p 3306:3306 mysql:8.0`

2. Executando a criação da tabela products e a adição de algumas linhas nele:
`docker exec -i mysql-container mysql -u root -padmin inventory < setup-inventory.sql`

3. Ao reiniciar o PC, o container do sql pode ter ficado parado, então apenas reiniicie o container com `docker start mysql-container`

Para acessar o banco de dados, use o comando:
`docker exec -it mysql-container mysql -u root -padmin inventory`
O nome do BD é inventory.

Para ver as tabelas, use o comando
`show tables;`
Haverá uma tabela chamada products

Para ver os dados da tabela, use o comando:
`select * from products;`

Deverá ter 5 linhas, de acordo com o arquivo de `setup-inventory.sql`