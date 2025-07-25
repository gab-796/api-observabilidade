# Estágio base para compartilhar código e dependências baixadas
FROM golang:1.24-alpine AS base

WORKDIR /app

# Copia go.mod e go.sum primeiro para aproveitar o cache do Docker
COPY go.mod go.sum ./
# Baixa dependências (go mod download é mais apropriado para um estágio base)
RUN go mod download
RUN go mod verify

# Copia o restante do código-fonte da sua aplicação
COPY . .

# ---- Estágio de Builder (para produção) ----
FROM base AS builder
# WORKDIR /app # Já definido no 'base'
RUN go mod tidy # Garante que o go.mod e go.sum estejam limpos
# Constrói o binário para produção, estático para a imagem scratch
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s' -o /app/main .

# ---- Estágio de Desenvolvimento (com air) ----
FROM base AS development
# WORKDIR /app # Já definido no 'base'

# Instala dependências do sistema operacional necessárias para 'air' (git)
# e CGO (gcc, musl-dev), caso seu projeto precise.
RUN apk add --no-cache git gcc musl-dev

# Instala o air
RUN go install github.com/air-verse/air@latest

# Expõe as portas que sua aplicação Go usa (10000 para app, 2113 para /metrics)
EXPOSE 10000
EXPOSE 2113

# Comando padrão para este estágio.
# O air usará o .air.toml para saber como construir e rodar sua app.
CMD ["air"]

# ---- Estágio final (Produção - a partir do 'scratch') ----
FROM scratch AS production
WORKDIR /app # Adicionado WORKDIR para consistência

# Copia apenas o binário compilado do estágio 'builder'
COPY --from=builder /app/main /app/main

# Copia e executa o script de inicialização do banco de dados
COPY --chmod=755 docker-entrypoint-initdb.d/setup.sh /docker-entrypoint-initdb.d/setup.sh

EXPOSE 10000

# Comando para executar o binário
CMD ["/app/main"]

LABEL version="1.0"
LABEL description="Versão 1.0 - Aplicação e BD mySQL sem telemetria alguma"
LABEL changelog="Versão inicial da aplicação"

LABEL version="2.0"
LABEL description="Versão 2.0 - Suporte a Log em nível INFO para aplicação e BD"
LABEL changelog="Suporte a log via logrus"

LABEL version="2.1"
LABEL description="Versão 2.1 - Suporte a Log e métrica http_requests_total"
LABEL changelog="Suporte a métrica via prometheus"

LABEL version="2.2"
LABEL description="Versão 2.2 - Suporte a Log e várias métricas"
LABEL changelog="Suporte a métrica via prometheus"

LABEL version="2.3"
LABEL description="Versão 2.3 - Suporte a traces via Otel Collector"
LABEL changelog="Suporte a envio de traces via Otel Collector para o DataDog"

LABEL version="3.0"
LABEL description="Versão 3.0 - OtelCollector Funcional para envio pro DD via Docker"
LABEL changelog="Versão inicial com suporte a Traces via Otel Collector pro DD"

LABEL version="3.1"
LABEL description="Versão 3.1 - Fix setando o otlp exporter endpoint dinamicamente pra k8s funcionar"
LABEL changelog="Na v3.0 só funcionava pra Docker, agora vai funcionar pra k8s"

LABEL version="3.2"
LABEL description="Versão 3.2 - Substitui o Datadog como recebedor de traces pelo Tempo e adiciona instrumentação a biblioteca MySQL"
LABEL changelog="Adiciona o container do Grafana Tempo e a instrumentação pra trace de biblioteca do container mySQL"

LABEL version="3.3"
LABEL description="Versão 3.3 - Diferencia o trace de http e mysql com uso de 2 tracer providers"
LABEL changelog="Adiciona mais logs e separa os traces de http e mysql com o uso de 2 tracer providers"

LABEL version="3.4"
LABEL description="Versão 3.4 - Adiciona o Air no Dockerfile para desenvolvimento e o trace do pacote mux e gRPC"
LABEL changelog="Possibilida o auto-reload da aplicação com o Air"

LABEL version="4.0"
LABEL description="Versão 4.0 - Adiciona o Grafana Pyroscope e correlação entre log e trace"
LABEL changelog="Pyscorope para profiling e correlação entre log e trace"