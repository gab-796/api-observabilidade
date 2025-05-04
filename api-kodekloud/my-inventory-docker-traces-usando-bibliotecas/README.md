# api-observabilidade em Docker
API de inventário em Go(v1.22) usando o mySQL v8.0 containerizado.
 Use o arquivo da collection Postman para poder fazer as chamadas de API e verificar o funcionamento da aplicação.

## Ideia de uso
Colocar a aplicação em um container Docker e deixar o container do mysql fora dele.
Aqui está o endereço da imagem buildada no dockerhub:
`gab796/inventory_app:v2.2`

### Uso localmente
Basta entrar na pasta e executar `go run .`
OBS: Não é recomendado executar localmente, devido as dependências do BD. Opte pelo uso em docker!

## Telemetria e versões

### Logrus - a partir da v2.0
Instalando o pacote logrus
`go get github.com/sirupsen/logrus`

### Métricas
Usaremos o client prometheus pra go
`go install github.com/prometheus/client_golang/prometheus`
`go install github.com/prometheus/client_golang/prometheus/promhttp`
`go install github.com/prometheus/client_golang/prometheus/promauto`

- **prometheus**: Para usar o cliente Prometheus básico, como métricas, registradores e outras utilidades.
- **promauto**: Facilita a criação e registro de métricas, permitindo uma maneira mais simples de criar métricas de contador, histograma, etc.
- **promhttp**: Fornece a funcionalidade necessária para expor as métricas via HTTP, como o promhttp.Handler() que você usa no seu servidor de métricas.

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


## Manipulando a imagem docker

### Criação da imagem e as coloca pra rodar - Usado exaustivamente para testes
`docker compose up --build`
Ou criando sem uso de cache e atribuindo uma nova imageID
`docker compose build --no-cache`

> Esse comando é essencial ao se criar uma nova feature. Sempre execute ele e depois execute o docker compose up para ficar testando se a feature está funcional ou não. Quando estiver tudo ok, execute o comando abaixo pra salvar ela no docker hub.

### Quando a imagem estiver estável, aplique a tag
`docker tag <nome_da_imagem ou imageID> gab796/inventory_app:v2.0`

> Para obter o nome da imagem criada pelo docker compose: `docker compose images`

### Rodando a imagem em segundo plano, liberando o terminal - Não cria a imagem!
`docker compose up -d`
Porém o ideal é rodar a imagem segurando o terminal, pois assim teremos os logs exibidos diretamente na tela, para isso execute:
`docker compose up`

### Terminando a aplicação e removendo todos os containers
`docker compose down -v`

## Subindo a imagem no dockerhub
1. Login na dockerhub
`docker login`

2. Build da imagem, sem rodar ela, apenas pra enviar pro dockerhub
`docker build --tag gab796/inventory_app:vN.n .`

> Executado dentro da pasta com todos os arquivos go e dockerfile e docker compose.

3.  Enviando a imagem para o docker hub
`docker push gab796/inventory_app:vN.n`

#####################################################################################################


## Acessando a primeira métrica no /metrics - http_requests_total
Quando vc subir a aplicação, basta entrar em
 `localhost:2113/metrics`

O resultado é esse após acessar algumas rotas via Postman.
//# HELP http_requests_total Número total de requisições HTTP recebidas
//# TYPE http_requests_total counter
http_requests_total{method="GET",path="/product/2",status="200"} 1
http_requests_total{method="GET",path="/products",status="200"} 1
http_requests_total{method="POST",path="/product",status="201"} 1

//# HELP sql_errors_total Número total de erros de SQL
//# TYPE sql_errors_total counter
sql_errors_total 0

//# HELP products_in_db Número de produtos no banco de dados
//# TYPE products_in_db gauge
products_in_db 6

//# HELP http_requests_total Número total de requisições HTTP recebidas
//# TYPE http_requests_total counter
http_requests_total{method="GET",path="/product/5",status="200"} 1
http_requests_total{method="GET",path="/products",status="200"} 2

// # HELP http_active_connections Número de conexões HTTP ativas
// # TYPE http_active_connections gauge
http_active_connections 0

// # HELP http_request_duration_seconds Duração das requisições HTTP em segundos
// # TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/product/5",le="0.005"} 1
http_request_duration_seconds_bucket{method="GET",path="/product/5",le="0.01"} 1
http_request_duration_seconds_bucket{method="GET",path="/product/5",le="0.025"} 1
http_request_duration_seconds_bucket{me# HELP go_gc_duration_seconds A summary of the wall-time pause (stop-the-world) duration in garbage collection cycles.

---

## Traces de forma automatizada das requisições HTTP de entrada

1. Começando executando esses comandos:
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/trace
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/otel/sdk
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
go get go.opentelemetry.io/otel/propagation
go get google.golang.org/grpc

2. Use o container do otelcollector - está no docker compose!
Ele usará o arquivo .env com a chave de API do DD usado no momento.
> Esse arquivo está no .gitignore, portanto não vazará a chave.
> O arquivo terá essa estrutura DD_API_KEY_GO_LAB=<SUA-API-KEY>

### Mas em caso de k8s
Leia o Readme.md correspondente.

1. Execute qualquer uma das rotas e veja os traces aparecerem na aba de traces do DD correspondente.

Os traces costumam ter os seguintes valores no Docker:
GET 600micros
GET em product 1.58ms
POST 8.86ms
PUT 3.5ms
DELETE 7.46ms

## Lista de instrumentadores de trace para pacotes usados nessa aplicação

1. instrumenta trace no pacote gorilla mux
https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux

2. instrumenta logrus com trace
https://github.com/uptrace/opentelemetry-go-extra/tree/main/otellogrus
Vai injetar o trace_id e o span_id nos logs do logrus!

3. instrumenta no pacote net/http
https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http#pkg-overview
OU
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 --> Faz parte da instrumentação automatizada, já implementada.

4. instrumenta trace no pacote grpc
https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/google.golang.org/grpc/otelgrpc/example
go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc

5. instrumenta no pacote mysql
Anterior: go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql
Atualizado para : go get github.com/XSAM/otelsql
