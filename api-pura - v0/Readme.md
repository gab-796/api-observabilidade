# Criação de uma API em Go

Passos usados para ter o projeto upado no meu git:

0. Vamos usar a biblioteca `gin-gonic` por ser a mais performática para o desenvolvimento de API Rest e serviços web.
   1. Na API do Kodekloud foi utilizada a biblioteca padrão net/http por ser mais fácil de usar.
1. Para criar o projeto em Go, execute: `go mod init github.com/gab-796/api-observabilidade/api`
Isso vai criar o arquivo go.mod com esse endereço nele
1. `go get github.com/gin-gonic/gin` --> Com esse vamos baixar efetivamente o gin e adicionar no go.mod as dependências bem como criar o go.sum
2. Já podemos rodar o main.go para ver o início da API respondendo na porta 3000, de acordo com o escolhido no codigo.

Continuando apos escrever a parte do codigo do postres:
4. `go get github.com/jackc/pgx/v4`


## Objetivo dessa pasta
Era criar a API em Golang usando Postgres sem ajudad alguma, ou seja, do 0.
Na época, não consegui fazer dessa forma e tive de apelas pro curso do Kodekloud.


### Detalhes técnicos do Gin
1. Ele já tem Middleware de Logger e Recovery(para capturar um panic e evitar que o servidor inteiro caia dando erro 500)