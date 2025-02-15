# api-observabilidade em Docker
API simples em Go usando o mySQL em container

## Ideia de uso
Colocar a aplicação em um container Docker e deixar o container do mysql fora dele.
Aqui está o endereço da imagem buildada no dockerhub: `gab796/inventory_app:v1.0`

## Manipulando a imagem

### Criação da imagem
`docker compose up --build`
Ou criando sem uso de cache
`docker compose build --no-cache`

### Rodando a imagem em segundo plano, liberando o terminal
`docker compose up -d`

### Fechando o container
`docker compose down`

## Verificando o Mysql manualmente

Para acessar o banco de dados, use o comando:
`docker exec -it mysql-container mysql -u root -padmin inventory`
O nome do BD é inventory.

Para ver as tabelas, use o comando
`show tables;`
Haverá uma tabela chamada products

Para ver os dados da tabela, use o comando:
`select * from products;`

Deverá ter 5 linhas
