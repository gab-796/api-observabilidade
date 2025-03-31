# Criação de uma API em Go

Passos usados para ter o projeto upado no meu git:

1. Vamos usar o gin: `go mod init github.com/gab-796/api-observabilidade/api`
Isso vai criar o arquivo go.mod com esse endereço nele
2. `go get github.com/gin-gonic/gin` --> Com esse vamos baixar efetivamente o gin e adicionar no go.mod as dependências bem como criar o go.sum
3. Já podemos rodar o main.go para ver o início da API respondendo na porta 3000, de acordo com o escolhido no codigo.

COntinuando apos escrever a parte do codigo do postres:
4. `go get github.com/jackc/pgx/v4`