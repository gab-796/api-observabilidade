# Readme do chart

## Dependências
1. ArgoCD do cluster kind
2. Ingress e MetalLb do cluster kind
3. Grafana K6, caso queira executar o teste de carga.


## Como usar?
Leia o Readme dentro da pasta argocd e verifique o necessário para subir a aplicação localmente.

Caso não queira usar o ArgoCD para deployar, mas sim o Helm, só usar o commando  
`helm install api-observabilidade ./helm-chart/pdi-gabriel -f ./helm-chart/pdi-gabriel/values.yaml -n api-app-go --create-namespace`

## Teste de carga via Grafana K6
Entre na pasta k6 e execute o arquivo javascript:
`k6 run inventory-load-test.js`