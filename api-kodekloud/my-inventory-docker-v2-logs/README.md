# api-observabilidade em Docker
API de inventário em Go(v1.22) usando o mySQL v8.0 containerizado.  
 Use o arquivo da collection Postman para poder fazer as chamadas de API e verificar o funcionamento da aplicação.

## Ideia de uso
Colocar a aplicação em um container Docker e deixar o container do mysql fora dele.  
Aqui está o endereço da imagem buildada no dockerhub:  
`gab796/inventory_app:v2.0`

A versão 2.0 conta com log via logrus, mas poderia ser qualquer um desses:
- zap da Uber(https://github.com/uber-go/zap) --> requer a versão mais atualizada do Golang, mas é o [pacote mais rápido disponivel](https://betterstack.com/community/guides/logging/go/zap/)
- slog(https://go.dev/blog/slog)
- zerolog(https://pkg.go.dev/github.com/rs/zerolog#section-readme) --> apenas JSON

Foi escolhido o logrus por estar já bem estabelecido e pela farta documentação, apesar de não receber atualização mais.

## Manipulando a imagem

### Criação da imagem
`docker compose up --build`
Ou criando sem uso de cache
`docker compose build --no-cache`

### Rodando a imagem em segundo plano, liberando o terminal
`docker compose up -d`
Porém o ideal é rodar a imagem segurando o terminal, pois assim teremos os logs exibidos diretamente na tela, para isso execute:
`docker compose up`

### Terminando a aplicação e removendo todos os containers
`docker compose down -v`

## Verificando o Mysql manualmente

Para acessar o banco de dados, use o comando:
`docker exec -it mysql-container mysql -u root -padmin inventory`  
O nome do BD é inventory.

Para ver as tabelas, use o comando
`show tables;`  
Haverá uma tabela chamada products

Para ver os dados da tabela, use o comando  
`select * from products;`

Deverá ter 5 linhas.

ou apenas  
`docker exec -it mysql-container mysql -u root -padmin -e "USE inventory; SELECT * FROM products;"`

## Logrus

Instalando o pacote logrus  
`go get github.com/sirupsen/logrus`

## Processo de build e push da imagem

#### Login na dockerhub
`docker login`

### Criação da imagem
`docker build -t gab796/inventory_app:2.0 .`

> Executado dentro da pasta com todos os arquivos go e dockerfile e docker compose.

### Enviando a imagem para o docker hub
`docker push gab796/inventory_app:v2.0`