# api-observabilidade em Docker
API simples em Go usando o mySQL em container

1. Pra rodar todos os arquivos .go ao mesmo tempo dê `go run .` na pasta my-inventory.

## Ideia de uso
Colocar a aplicação em um container Docker e deixar o container do mysql fora dele.

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

1. Ao reiniciar o PC, o container do sql pode ter ficado parado, então apenas reiniicie o container com `docker start mysql-container`

Para acessar o banco de dados, use o comando:
`docker exec -it mysql-container mysql -u root -padmin inventory`
O nome do BD é inventory.

Para ver as tabelas, use o comando
`show tables;`
Haverá uma tabela chamada products

Para ver os dados da tabela, use o comando:
`select * from products;`

Deverá ter 5 linhas, de acordo com o arquivo de `setup-inventory.sql`
