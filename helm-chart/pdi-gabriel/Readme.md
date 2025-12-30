# Criação do helm chart
`helm create pdi-gabriel`


## v0
Todos os yaml estão na pasta template de forma que o chart sobe todos eles como se fosse um makefile.

## V1
Parametrizar os templates para que os valores a serem considerados no values.yaml é que sejam levados em conta.

## Coleta de logs dos Nodes
Não é possível pro Alloy capturar esses logs dos nodes do Kind. Eu precisaria rodar um Alloy no meu host, pra assim pegar os logs do container que o Kind usa como nodes…

Porque no Kind, os containers não são máquinas reais, mas containers Docker. eles não tem um SO que gera logs de sistema como syslog e journald.

Os arquivos ficariam em /var/log ou acessíveis via journald, mas como não temos SO, apenas containers, nada disso é gerado. Os únicos logs relevantes são dos processos do k8s, que ficam dentro do container Docker.

O acesso dos logs do kubelet e containerd se dá via `docker logs <nome-do-node>`


## Coleta de Métricas no Otel Collector

### Métricas do CAdvisor
job_name: 'cadvisor'
- Memória usada por todos os containers
Captura direto do CAdvisor, que mora no Kubelet, na porta 10250 via HTTPS,  métricas detalhadas de uso de CPU, memória, disco e rede para cada contêiner em execução, como container_memory_usage_bytes
São métricas que iniciam com container_

### Métricas do Kube State Metrics
job_name: 'kube-state-metrics'
Gera métricas de estado dos objetos k8s como kube_deployment_spec_replicas e kube_node_status_condition

### Métricas do proprio Mimir
job_name 'mimir-metrics' no otel collector

### Métricas da aplicação api-app-go
job_name: 'inventory-app'

## Dependências
1. KubeStateMetrics -> Métricas de CPU, Memória, HPA e etc do cluster todo
   1. Instalado no ns ksm
2. EventRouter -> Obtem eventos de HPA e os transforma em logs, assim o painel do dashboard de logs consegue carregar os logs do HPA.
   1. instalado no ns do keda
   2. `helm upgrade my-eventrouter krateo/eventrouter --namespace keda -f values.yaml`
3. Vault e External Secret Operator

## Tempo
Estamos usando o init container para deletar todos os arquivos que ficam no WAL e nos blocks, afim de garantir que nada corrompido fique entre cada reinstalação que façamos no decorrer dos testes.

## Uso do k6 - Teste de carga
Uso mais rápido é com o k6 instalado, executando:
`k6 run k6/inventory-load-test.js`

Para mais detalhes, consulte o Readme dentro da pasta do k6.