# Criação do helm chart
`helm create pdi-gabriel`


## v0
Todos os yaml estão na pasta template de forma que o chart sobe todos eles como se fosse um makefile.

## V1
Parametrizar os templates para que os valores a serem considerados no values.yaml é que sejam levados em conta.

## Dependências
1. KubeStateMetrics -> Métricas de CPU, Memória, HPA e etc do cluster todo
   1. Instalado no ns ksm
2. EventRouter -> Obtem eventos de HPA e os transforma em logs, assim o painel do dashboard de logs consegue carregar os logs do HPA.
   1. instalado no ns do keda
   2. `helm upgrade my-eventrouter krateo/eventrouter --namespace keda -f values.yaml`