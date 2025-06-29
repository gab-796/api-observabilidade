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
- A v3.0 conta com trace via OtelCollector enviando pro DD da forma mais simples possível, mas só funciona em Docker
- v3.1 está com o otelexporter endpoint corrigido na main.go para que funcione tanto no Docker quanto no k8s.
- v3.2 contém traces para biblioteca MySQL e envio pro Grafana Tempo via OtelCollector. Datadog foi deprecated.
- v4.0 contém Profiling com Pyroscope e traces correlacionados com Logs. Grafana Alloy coleta os logs e enviando pro Loki e Otel collector coletando as métricas e enviando pro Mimir e traces sendo enviados ao Tempo.

O arquivo a ser configurada a versão é o `api-deployment.yaml`

## Dependências
0. Ter o kind e o Docker instalado
1. Instalar o cluster kind de Observabilidade (Execute o Makefile na pasta kind-cluster)
2. É necessário ter o nginx ingress controller instalado e incluir o DNS `inventory.local` no seu `/etc/hosts` e também o `grafana-web.local` para que o ingress funcione.
3. Acesse o grafana direto no browser via `grafana-web.local`.

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

## Dependências da v3.0
- Crie a secret com a sua apikey do DD:
`kubectl create secret generic datadog-api-key --from-literal=DD_API_KEY_GO_LAB=sua_chave_de_api_aqui`

> V3.2 em diante não usamos mais Datadog, portanto não precisa desse passo.

## Nota sobre os ingress

Há um arquivo de ingress para a Grafana Stack chamado `ingress-grafana-stack.yaml`
Subimos ingress para o Mimir, alloy, Tempo e PYroscope, porém eles não estão sendo usados.
Achei melhor deixar o datasource configurado pegando direto do nome do service deles.

O único ingress usado realmente é do Grafana Web, pois precisamos digitar esse endereço no browser para poder ter acesso a UI do Grafana.

## Mimir no Grafana
Há Métricas da aplicação sendo expostas, bem como métricas do próprio Mimir(otel collector buscando) e do Alloy(Alloy enviando suas métricas ao Mimir).