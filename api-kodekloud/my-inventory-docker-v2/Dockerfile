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
LABEL description="Versão 2.1 - Suporte a Log e métrica"
LABEL changelog="Suporte a métrica via prometheus"
