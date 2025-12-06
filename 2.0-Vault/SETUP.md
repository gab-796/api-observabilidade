# ...existing code...

## ⚠️ Diferença entre LAB e PRODUÇÃO

### LAB (este repositório)
O script `deploy.sh` **cria automaticamente** as secrets no Vault para facilitar testes.

### PRODUÇÃO (mundo real)
Você deveria:

1. **Vault já existe** (gerenciado por equipe de segurança)
2. **Criar secrets MANUALMENTE** antes do deploy:
   ```bash
   vault kv put secret/inventory-app/database \
     MYSQL_ROOT_PASSWORD=$(openssl rand -base64 32) \
     DB_PASSWORD=$(openssl rand -base64 32)
   ```
3. **Deploy apenas da aplicação** (sem tocar no Vault)

# ...existing code...