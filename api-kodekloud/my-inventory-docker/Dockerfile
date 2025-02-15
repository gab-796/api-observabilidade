# Estágio de build
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod tidy

RUN go build -o main .

# Estágio final
FROM scratch

COPY --from=builder /app/main /app/

COPY --chmod=755 docker-entrypoint-initdb.d/setup.sh /docker-entrypoint-initdb.d/

EXPOSE 10000

CMD ["/app/main"]

# A imagem gerada por esse Dockerfile tem 8MB.