#!/bin/bash

# Script para monitorar status do Pyroscope
echo "🔍 Monitorando status do Pyroscope..."

while true; do
    response=$(curl -s http://localhost:4040/ready 2>/dev/null)
    
    if [[ $response == "ready" ]]; then
        echo "✅ $(date '+%H:%M:%S') - Pyroscope está PRONTO!"
        break
    elif [[ $response == *"Ingester not ready"* ]]; then
        echo "⏳ $(date '+%H:%M:%S') - Ingester aguardando estabilização (15s)..."
    elif [[ -z $response ]]; then
        echo "❌ $(date '+%H:%M:%S') - Pyroscope não está respondendo"
    else
        echo "⚠️  $(date '+%H:%M:%S') - Status: $response"
    fi
    
    sleep 2
done

echo "🎯 Pyroscope está funcionando em http://localhost:4040"