package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	http.Handle("/metrics", promhttp.Handler())  // Expor as métricas
	log.Fatal(http.ListenAndServe(":2112", nil)) // Servir as métricas na porta 2112
}

// Esse app apenas sobe um servidor de prometheus e o expõe na porta 2112.
// Para testar, basta rodar o app e acessar http://localhost:2112/metrics

/* Instruções para criação desse path
1. go mod init promtest.com/t
2. go mod tidy
3. go install github.com/prometheus/client_golang/prometheus/promauto@latest
4. go install github.com/prometheus/client_golang/prometheus/promhttp@latest
5. go install github.com/prometheus/client_golang/prometheus@latest
*/
