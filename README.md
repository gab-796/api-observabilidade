# 🔭 API Observabilidade

> **Plano de Desenvolvimento Individual (PDI)** - Projeto completo de observabilidade com Kubernetes, Grafana Stack e Go

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.29+-326CE5?style=flat&logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![ArgoCD](https://img.shields.io/badge/ArgoCD-GitOps-EF7B4D?style=flat&logo=argo&logoColor=white)](https://argoproj.github.io/cd/)
[![Grafana](https://img.shields.io/badge/Grafana-Stack-F46800?style=flat&logo=grafana&logoColor=white)](https://grafana.com/)

---

## 📋 Índice

- [Sobre o Projeto](#-sobre-o-projeto)
- [Arquitetura](#-arquitetura)
- [Stack Tecnológica](#-stack-tecnológica)
- [Estrutura do Repositório](#-estrutura-do-repositório)
- [Quick Start](#-quick-start)
- [Roadmap](#-roadmap)
- [Documentação](#-documentação)

---

## 🎯 Sobre o Projeto

Projeto de observabilidade end-to-end para uma **API REST de inventário** construída em Go, deployada em Kubernetes local (Kind) com GitOps (ArgoCD), instrumentação completa (traces, logs, métricas, profiling) e stack de observabilidade Grafana (Loki, Tempo, Mimir, Pyroscope).

### 🎓 Objetivos de Aprendizado

- ✅ Desenvolvimento de APIs REST em Go
- ✅ Observabilidade com **4 pilares** (Logs, Traces, Metrics, Profiling)
- ✅ Kubernetes local (Kind) com infraestrutura completa
- ✅ GitOps com ArgoCD e Helm Charts
- ✅ Grafana Stack distribuído (Loki, Tempo, Mimir, Pyroscope)
- ✅ Testes de carga com K6
- ✅ Gestão de secrets com HashiCorp Vault

### 🏗️ Premissas

- Repositório pessoal para aprendizado hands-on
- Cluster Kubernetes local (Kind)
- GitOps com ArgoCD para deployment
- Helm Charts templatizados e reutilizáveis
- Makefiles para automação de build/deploy

---

## 🏛️ Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster (Kind)               │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────────┐  │
│  │   Ingress    │───▶│  API Go      │───▶│  MySQL DB       │  │
│  │  Controller  │    │  (Inventory) │    │                 │  │
│  └──────────────┘    └──────┬───────┘    └─────────────────┘  │
│                             │                                   │
│                             ▼                                   │
│         ┌───────────────────────────────────────────┐           │
│         │     Observability Stack (Grafana)        │           │
│         │                                           │           │
│         │  ┌─────────┐  ┌────────┐  ┌──────────┐  │           │
│         │  │  Tempo  │  │  Loki  │  │  Mimir   │  │           │
│         │  │(Traces) │  │ (Logs) │  │(Metrics) │  │           │
│         │  └─────────┘  └────────┘  └──────────┘  │           │
│         │                                           │           │
│         │  ┌──────────┐  ┌────────┐                │           │
│         │  │Pyroscope │  │ Alloy  │                │           │
│         │  │(Profile) │  │(Agent) │                │           │
│         │  └──────────┘  └────────┘                │           │
│         │                                           │           │
│         │         ┌────────────────┐                │           │
│         │         │  Grafana Web   │                │           │
│         │         │   (Dashboard)  │                │           │
│         │         └────────────────┘                │           │
│         └───────────────────────────────────────────┘           │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────────┐  │
│  │   Vault      │    │   ArgoCD     │    │   K6 (Load      │  │
│  │  (Secrets)   │    │   (GitOps)   │    │    Testing)     │  │
│  └──────────────┘    └──────────────┘    └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🛠️ Stack Tecnológica

### Backend & Infraestrutura
| Tecnologia | Versão | Função |
|-----------|--------|---------|
| **Go** | 1.21+ | Linguagem da API |
| **net/http** | - | Framework HTTP |
| **MySQL** | 8.0 | Banco de dados |
| **Docker** | - | Containerização |
| **Kubernetes (Kind)** | 1.29+ | Orquestração |
| **ArgoCD** | - | GitOps/CD |
| **Helm** | 3.x | Package manager K8s |

### Observabilidade
| Componente | Função |
|-----------|---------|
| **Grafana Tempo** | Distributed tracing |
| **Grafana Loki** | Log aggregation |
| **Grafana Mimir** | Metrics (Prometheus-compatible) |
| **Grafana Pyroscope** | Continuous profiling |
| **Grafana Alloy** | Telemetry pipeline |
| **OpenTelemetry** | Instrumentação (Go SDK) |

### DevOps & Tooling
| Ferramenta | Função |
|-----------|---------|
| **HashiCorp Vault** | Secret management |
| **K6** | Load testing |
| **External Secrets Operator** | Vault → K8s sync |
| **Ingress NGINX** | API Gateway |
| **HPA** | Horizontal Pod Autoscaler |

---

## 📁 Estrutura do Repositório

```
api-observabilidade/
├── 📱 api-kodekloud/              # Versões evolutivas da API
│   ├── v0-app-only-sem-docker/
│   ├── v1-com-docker/
│   ├── v2-logs/
│   ├── v3-metricas/
│   ├── v4-traces-com-k8s/
│   ├── v5-traces-com-bibliotecas-distintas/
│   ├── v6-profiling/
│   └── v7-WIP-trace-and-profile-correlated/
│
├── ☸️ api-k8s/                    # Manifestos Kubernetes
│   ├── 1.0/                       # Deploy básico
│   └── 2.0-Vault/                 # Deploy com Vault
│
├── 📦 helm-chart/                 # Helm Chart completo
│   ├── pdi-gabriel/               # Chart principal
│   │   ├── templates/
│   │   ├── dashboards/            # Grafana dashboards (JSON)
│   │   └── values.yaml
│   ├── argocd/                    # Configuração ArgoCD
│   │   ├── application.yaml
│   │   ├── values-argocd.yaml
│   │   └── README.md              # 📚 Guia de deploy ArgoCD
│   └── k6/                        # Testes de carga
│       ├── inventory-load-test.js
│       └── README.md              # 📚 Guia de testes K6
│
├── 📄 docs/                       # Documentação técnica
│   ├── api 2.0.md
│   └── ARQUITETURA_APLICACAO_GO.md
│
└── 🔧 EventRouter/                # Event logging para HPA
    └── values.yaml
```

---

## 🚀 Quick Start

### Pré-requisitos

```bash
# Instalar dependências
brew install kind kubectl helm argocd k6  # macOS
# ou
sudo apt install kubectl helm            # Linux
```

### 1. Criar Cluster Kubernetes Local

```bash
cd api-k8s/kind-cluster
make create-cluster
```

### 2. Deploy com ArgoCD

```bash
# Instalar ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Deploy da aplicação
kubectl apply -f helm-chart/argocd/application.yaml

# Acessar Grafana
# http://grafana-web.local (admin/admin)
```

### 3. Configurar Secrets no Vault

```bash
# Ver documentação completa em:
# helm-chart/argocd/README.md#5-aguardar-vault-e-configurar-secrets
```

### 4. Executar Teste de Carga

```bash
cd helm-chart/k6
k6 run inventory-load-test.js
```

📖 **Documentação Completa:**
- [Deploy com ArgoCD](helm-chart/argocd/README.md)
- [Testes de Carga K6](helm-chart/k6/README.md)
- [Configuração Vault](api-k8s/2.0-Vault/SETUP.md)

---

## 🗓️ Roadmap

### ✅ Q1/2025 - Fundamentos

- [x] Treinamento Go (KodeKloud)
- [x] API REST em Go (CRUD completo)
  - [x] GET, POST, PUT, DELETE
  - [x] Integração com MySQL
- [x] Containerização (Docker + Docker Compose)

### ✅ Q2/2025 - Kubernetes & Observabilidade Básica

- [x] Cluster local com Kind
- [x] Deploy em Kubernetes (Deployment, Service, Ingress)
- [x] Instrumentação da API:
  - [x] **Traces** (OpenTelemetry)
  - [x] **Logs** (Logrus)
  - [x] **Metrics** (Prometheus)
  - [x] **Profiling** (Pyroscope)

### ✅ Q3/2025 - Stack Distribuída & GitOps

- [x] Grafana Stack distribuído:
  - [x] Tempo (traces)
  - [x] Loki (logs)
  - [x] Mimir (metrics)
  - [x] Pyroscope (profiling)
- [x] ArgoCD + Helm Charts
- [x] Dashboards Grafana (C4 model)
- [x] Monitoramento de infra K8s:
  - [x] Ingress Controller, CoreDNS
  - [x] Nodes, Pods, HPA
  - [x] Control Plane (scheduler, controller-manager, etcd)
- [x] Kube State Metrics + cAdvisor

### ✅ Q4/2025 - Segurança & Otimizações

- [x] **HashiCorp Vault** para secrets
- [x] **External Secrets Operator**
- [x] Testes de carga com **K6**
- [x] HPA (Horizontal Pod Autoscaler)
- [x] Dashboard melhorado (antes/depois)

### 🔮 Backlog / Melhorias Futuras

- [ ] OTEL Collector para logs (OTLP gRPC/HTTP)
  - _Nota: SDKs de log ainda estão em desenvolvimento, Logrus continua sendo mais maduro_
- [ ] Aplicar chart no Karavela (cluster produção-like)
- [ ] Vagrant para provisionamento de VMs
- [ ] Correlação avançada logs ↔ traces
- [ ] CI/CD com GitHub Actions
- [ ] Multi-cluster deployment

---

## 📚 Documentação

### 🎯 Guias Principais

| Documento | Descrição |
|-----------|-----------|
| [📘 Deploy com ArgoCD](helm-chart/argocd/README.md) | Guia completo de deploy GitOps |
| [📗 Testes de Carga K6](helm-chart/k6/README.md) | Como executar testes de performance |
| [📕 Setup Vault](api-k8s/2.0-Vault/SETUP.md) | Configuração de secrets |
| [📙 Arquitetura Go](docs/ARQUITETURA_APLICACAO_GO.md) | Estrutura da aplicação |

### 🔧 Componentes

- **API Endpoints:**
  - `GET /products` - Listar produtos
  - `GET /product/:id` - Obter produto por ID
  - `POST /product` - Criar produto
  - `PUT /product/:id` - Atualizar produto
  - `DELETE /product/:id` - Deletar produto
  - `GET /health` - Health check

- **Observability Endpoints:**
  - `:10000/health` - API health
  - `:2113/metrics` - Prometheus metrics
  - Traces → Tempo (automatic via OTEL)
  - Logs → Loki (via Alloy)
  - Profiling → Pyroscope (continuous)

---

## 📊 Dashboards Grafana

O projeto inclui dashboards customizados para monitoramento:

### 📈 API Inventory Dashboard
- **RED Metrics** (Rate, Errors, Duration)
- **HTTP Requests** por método/path/status
- **Latência** (p50, p90, p95, p99)
- **Erros** e taxa de sucesso
- **Database connections**

### ☸️ Kubernetes Infrastructure Dashboard
- **Nodes**: CPU, Memória, Status
- **Pods**: Status, restarts, resource usage
- **HPA**: Autoscaling events e métricas
- **Ingress**: Request rate e latência
- **CoreDNS**: Query rate e erros

---

## 🎓 Lições Aprendidas

### ✅ O que funcionou bem

1. **External Secrets Operator** > Vault Agent Injector
   - Compatível com imagens scratch
   - Secrets como env vars nativas do K8s
   - Multi-provider (Vault, AWS, GCP, Azure)

2. **Grafana Alloy** para telemetria
   - Substituto moderno do Prometheus Agent
   - Pipeline unificado (logs + metrics + traces)
   - Configuração mais simples

3. **Kind para cluster local**
   - Mais leve que Minikube
   - Multi-node support
   - Suporte a Ingress nativo

4. **ArgoCD + Helm**
   - GitOps elimina `kubectl apply` manual
   - Self-healing e auto-sync
   - Rollback fácil

### ⚠️ Desafios Encontrados

1. **Logs de nodes do Kind**
   - Impossível capturar via DaemonSet usando Kind
   - Nodes são containers Docker, não VMs
   - Solução: aceitar limitação ou usar cluster real

2. **Dashboard Grafana em Helm templates**
   - JSON dentro de YAML causa conflitos
   - Solução: usar `.Files.Get` para JSON externo
   - Placeholders `{{}}` do Grafana conflitam com Helm

3. **Vault em dev mode**
   - Dados não persistentes (memória)
   - Token fixo `root` (inseguro)
   - OK para lab, produção precisa HA + backend persistente

---

## 📝 Notas Técnicas

### Limitações Conhecidas

- **Kind cluster**: Logs de nodes não capturáveis via DaemonSet
- **Vault dev mode**: Dados não persistem em restart
- **OTEL log SDK**: Ainda imaturo, Logrus é mais estável
- **Single namespace**: Vault + API no mesmo namespace (não recomendado para prod)

### Requisitos de Recursos

**Mínimo:**
- CPU: 4 cores
- RAM: 8GB
- Disk: 20GB

**Recomendado:**
- CPU: 8 cores
- RAM: 16GB
- Disk: 50GB

---

## 📜 Licença

Este projeto é de código aberto para fins educacionais.

---

## 👤 Autor

**Gabriel Rocha**  
📧 Email: gabriel.rocha(arroba)ufrj.br  
🔗 LinkedIn: https://www.linkedin.com/in/gabrielrocha14/  
🐙 GitHub: [@gab-796](https://github.com/gab-796)

---

## ⭐ Se este projeto te ajudou

Considere dar uma estrela ⭐ no repositório!

---

<div align="center">

**Construído com ❤️ para aprender observabilidade end-to-end**

</div>

