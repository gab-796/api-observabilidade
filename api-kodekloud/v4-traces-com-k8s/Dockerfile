# Estágio de build
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod tidy

RUN go build -o main .

# Estágio final
FROM scratch
# o scratch permite usar uma imagem docker completamente vazia, pegando apenas o binário compiladodo estagio de build para a imagem final.
# Isso torna a imagem extremamente pequena.

COPY --from=builder /app/main /app/

COPY --chmod=755 docker-entrypoint-initdb.d/setup.sh /docker-entrypoint-initdb.d/

EXPOSE 10000

CMD ["/app/main"]

# A imagem gerada por esse Dockerfile tem 8MB.

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
LABEL description="Versão 3.0 - Fix setando o otlp exporter endpoint dinamicamente pra k8s funcionar"
LABEL changelog="Na v3.0 só funcionava pra Docker, agora vai funcionar pra k8s"