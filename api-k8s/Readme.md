# API em Go em k8s

A ideia é usar a imagem docker criada anteriormente para criação da API em 2 pods, um pra app e outro pro BD MySQL, todos morando no ns api-app-go.

Há um configmap apenas para iniciar junto do pod do mysql onde ele cria o database, a tabela products e ainda inclui 5 produtos.

## Versões da imagem docker da app
- A versão incial, v1.0 conta apenas com a aplicação e o BD, sem telemetria alguma.
- A versão 2.0 tem suporte a logs, via Logrus.
- Já a v2.1 tem suporte a logs e métrica http_requests_total em formato OpenMetrics.
- A v2.2 tem suporte a Log e várias métricas.
- A v2.2.1 contém a imagem alpine:latest com suporte a shell nos containers da aplicação.
- A v2.3 contém as 3 telemetrias, com envio de traces pelo Opentelemetry.


O arquivo a ser configurada a versão é o `api-deployment.yaml`

## Dependências
0. Ter o kind e o Docker instalado
1. Instalar o cluster kind de Observabilidade (Execute o Makefile na pasta kind-cluster)
2. É necessário ter o nginx ingress controller instalado e incluir o DNS `inventory.local` no seu `/etc/hosts` para que o ingress funcione.

A versão do k8s usada na criação desses manifestos é a `v1.29`

## Instalação
Dentro da pasta, basta executar:
1. `k apply -f namespace.yaml`
2. Instalação dos manifestos: `kubectl apply -f .`

OU

digite `make` para usar o makefile ;)

## Collection do Postman
Todos os métodos da API estão gravados na collection chamada `api-k8s-collection.json`, basta importar no seu Postman.

## Uso do app
Basta abrir o Postman e importar a collection.
Repare que elas usam o ingress como parte do path(inventory.local)

Para abrir o `/metrics`:  
`inventory.local/metrics`

### Checando a saúde do BD
`k exec -it <pod-do-mysql> -- mysql -u root -padmin -e "USE inventory; SELECT * FROM products;"`

exemplo: `k exec -it mysql-669586f559-dr7jw -- mysql -u root -padmin -e "USE inventory; SELECT * FROM products;"`

## Destruição do ambiente
Delete o ns e todos os seus recursos com `k delete ns api-app-go`

OU

`make destroy`
