# --- Estágio 1: Gerar o código semconv ---
    FROM golang:1.23-alpine AS semconv-generator

    WORKDIR /gen
    
    # Instalar o Git (ESSENCIAL!)
    RUN apk update && apk add --no-cache git
    
    # Clonar o repositório opentelemetry-go-contrib (que contém o semconvgen)
    RUN git clone https://github.com/open-telemetry/opentelemetry-go-contrib.git
    
    # Mudar para o diretório do semconvgen
    WORKDIR /gen/opentelemetry-go-contrib/cmd/semconvgen
    
    # Fazer checkout de uma tag específica (compatível com sua versão do SDK)
    # Use v0.49.0 (compatível com SDK v1.35.0, build-tools v0.98.0).
    RUN git checkout v0.49.0  # <---  CORREÇÃO AQUI: checkout simples da tag
    
    # Compilar o semconvgen manualmente
    RUN go build -o /go/bin/semconvgen .
    
    
    # Gerar o código (agora usando o binário compilado manualmente)
    COPY semconv /gen/semconv
    RUN /go/bin/semconvgen generate --config="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/trace.yaml" --output="/gen/semconv"
    RUN /go/bin/semconvgen generate --config="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/resource.yaml" --output="/gen/semconv"
    RUN /go/bin/semconvgen generate --config="https://raw.githubusercontent.com/open-telemetry/opentelemetry-specification/v1.35.0/semantic_conventions/metrics.yaml" --output="/gen/semconv"
    
    # --- Estágio 2: Construir a aplicação ---
    FROM golang:1.22-alpine AS builder
    
    WORKDIR /app
    
    # Copiar go.mod e go.sum
    COPY go.mod go.sum ./
    
    # Baixar dependências
    RUN go mod download
    
    # Copiar o código gerado do estágio anterior
    COPY --from=semconv-generator /gen/semconv /app/semconv
    
    # Copiar o restante do código fonte
    COPY . .
    
    # Permissões para o script (dentro do builder)
    COPY --chmod=755 docker-entrypoint-initdb.d/setup.sh /app/docker-entrypoint-initdb.d/
    RUN chmod +x /app/docker-entrypoint-initdb.d/setup.sh
    
    # Compilar a aplicação
    RUN go build -o main .
    
    # --- Estágio 3: Imagem Final (Runtime) ---
    FROM alpine:latest
    
    # Copiar o binário
    COPY --from=builder /app/main /usr/local/bin/main
    
    # Copiar o script de inicialização (se necessário)
    COPY --from=builder /app/docker-entrypoint-initdb.d/setup.sh /docker-entrypoint-initdb.d/
    
    EXPOSE 10000
    
    CMD ["/usr/local/bin/main"]
    
    # --- LABELS ---
    LABEL version="1.0" \
          description="Versão 1.0 - Aplicação e BD mySQL sem telemetria alguma" \
          changelog="Versão inicial da aplicação"
    
    LABEL version="2.0" \
          description="Versão 2.0 - Suporte a Log em nível INFO para aplicação e BD" \
          changelog="Suporte a log via logrus"
    
    LABEL version="2.1" \
          description="Versão 2.1 - Suporte a Log e métrica http_requests_total" \
          changelog="Suporte a métrica via prometheus"
    
    LABEL version="2.2" \
          description="Versão 2.2 - Suporte a Log e várias métricas" \
          changelog="Suporte a métrica via prometheus"
    
    LABEL version="2.3" \
          description="Versão 2.3 - Suporte a Log, métricas e traces via Open Telemetry" \
          changelog="Suporte as 3 telemtrias"