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

### Traces
Opentelemetry

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

### Criação da imagem e as coloca pra rodar - Usado axaustivamente para testes
`docker compose up --build`
Ou criando sem uso de cache e atribuindo uma nova imageID  
`docker compose build --no-cache`

### Quando a imagem estiver estável, aplique a tag
`docker tag <nome_da_imagemou imageID> gab796/inventory_app:v2.0`

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


##########################################################################################################################################

## Traces

Vamos usar o OpenTelemetry!

- SDK (go.opentelemetry.io/otel/sdk): O kit de desenvolvimento do OpenTelemetry. Ele fornece as classes e funções para configurar o tracing (TracerProvider, exporters, etc.).
- API (go.opentelemetry.io/otel): A API define as interfaces e tipos básicos (como Tracer, Span, Context). Seu código usa a API para criar spans, propagar o contexto, etc.
- Exporter (go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp): Este é o componente que envia os dados de tracing para o backend (Jaeger, Zipkin, etc.). Você está usando o exporter OTLP sobre HTTP (otlptracehttp).
- Resource (go.opentelemetry.io/otel/sdk/resource): Permite que você defina atributos que se aplicam a todos os spans gerados pela sua aplicação (nome do serviço, ambiente, versão, etc.).
- Semantic Conventions (go.opentelemetry.io/otel/semconv): Define nomes de atributos padrão (como service.name, http.method, http.status_code, etc.). Usar convenções semânticas torna mais fácil correlacionar dados de diferentes serviços e ferramentas.

## Pacotes utilizados
go get go.opentelemetry.io/otel \
    go.opentelemetry.io/otel/trace \
    go.opentelemetry.io/otel/sdk \
    go.opentelemetry.io/otel/exporters/otlp/otlptrace \
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp \
    go.opentelemetry.io/otel/sdk/resource \
    go.opentelemetry.io/otel/semconv/v1.17.0

Logo após isso, `go mod tidy`


## Treta do semconv

Não vamos utilizar ele no go mod direto e importando, mas apenas o gerador de código:

1. go install go.opentelemetry.io/collector/cmd/semconvgen@latest

2. mkdir semconv (aqui na raiz)

3. executar os comandos abaixo
semconvgen generate --input="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/trace.yaml" --output="./semconv"
semconvgen generate --input="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/resource.yaml" --output="./semconv"
semconvgen generate --input="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/metrics.yaml" --output="./semconv"

Não deu, entao vamos apelar:
go run go.opentelemetry.io/collector/cmd/semconvgen@v0.97.0 generate --input="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/trace.yaml" --output="./semconv"
go run go.opentelemetry.io/collector/cmd/semconvgen@v0.97.0 generate --input="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/resource.yaml" --output="./semconv"
go run go.opentelemetry.io/collector/cmd/semconvgen@v0.97.0 generate --input="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/metrics.yaml" --output="./semconv"


Na boa, refaz essa porra toda de trace do inicio, pegando da pasta v3 e recriando essa pasta v4...


## Sobre a instrumentação do código
De acordo com a documentação oficial, de início, bastaria usar os 3 pacotes:
go get go.opentelemetry.io/otel \
go.opentelemetry.io/otel/trace \
go.opentelemetry.io/otel/sdk \

1. "go.opentelemetry.io/otel" v1.35.0
Pacote raiz do opentelemetry pra Go.
Daqui é definido o TracerProvider, que cria instâncias de Tracer, etc.
POde ser definido o MeterProvider global também, mas esse é só prá métricas
E também define o Propagator global, que propaga contexto, como IDs de rastreamento.
Obtém acesso ao tracer global padrão, via `otel.Tracer()`
Lida com erros globais do OpenTelemetry

É um ponto de controle pro OpenTelemetry na sua app.

2. go.opentelemetry.io/otel/sdk v1.35.0
SDK oficial pro Go.
Fornece implementação das interfaces dos pacotes otel e otel/trace

3. go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.35.0
Exporter para enviar trace OTLP via HTTP. Tem outro tb pra grpc!

4. "go.opentelemetry.io/otel/trace" --> Ainda não usado!
Pacote core API para tracing dentro do OpenTelemetry.
Ele contém as **interfaces** e tipos fundamentais para criar e manipular spans (unidades de trabalho), traces (conjuntos de spans conectados) e o contexto de rastreamento.

**Interface TracerProvider** que definie como criar instâncias de Tracer. 
Usualmente não usamos ela diretamente, mas uma implementação concreta dela como o `sdktrace.TracerProvider` do pacote `go.opentelemetry.io/otel/sdk/trace`

**Interface Tracer** que é usada para criar spans.

E também Interfaces para **amostragem** (sampling), que decide quais traces devem ser coletados e exportados.

Se não usarmos esse pacote, os exportadores padrão, propagadores ou qq outra ferramenta que dependa dessas interfaces não poderão ser utilizados, daí vc ta criando um sistema de rastreamento isolado do Opentelemetry, que não faz sentido algum.

PS: Existe também o pacote de métricas: `go.opentelemetry.io/otel/metrics`, que substituiria o Prometheus ao usar o opentelemetry pra criar e enviar métricas.

5. go.opentelemetry.io/otel/sdk/trace
Faz parte da SDK e é específico para traces.
Ele fornece as implementações concretas das interfaces definidas em `go.opentelemetry.io/otel/trace`, além de funcionalidades adicionais para configurar e controlar o comportamento do tracing.

Ela que traz a implementação concreta da interface `trace.TracerProvider`
Fornece tb `trace.WithBatcher(exporter)` para configurar `BatchSpanProcessor`, usado em produção.


	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"


### Procurar depois esses pacotes
go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql
Instrumentação pra interações com bancos de dados SQL.

go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
Instrumentação para servidores e clientes HTTP.