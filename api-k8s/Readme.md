# API em Go em k8s

A ideia é usar a imagem docker criada anteriormente para criação da API em 2 pods, um pra app e outro pro BD, todos morando no ns api-app-go.

Há um configmap apenas para iniciar junto do pod do mysql onde ele cria o database, a tabela products e ainda inclui 3 itens lá.


## Dependências
É necessário ter o nginx ingress controller instalado e incluir o DNS `inventory.local` no seu `/etc/hosts` para que o ingress funcione.
O cluster kind local já vem com o ingress instalado, então só precisa adicionar o DNS junto dos outros nomes.

A versão do k8s usada na criação desses manifestos é a v1.29

## Instalação
Dentro da pasta, basta executar:
1. Criação do namespace: `kubectl create ns api-app-go`
2. Instalação dos manifestos: `kubectl apply -f .`


## Detalhes
Essa aplicação em Go não emite log algum, pois está na V0.
A v1 contará com telemetria!

## Collection do Postman

Todos os métodos da API estão gravados na collection chamada `api-k8s-collection.json`, basta importar no seu Postman.