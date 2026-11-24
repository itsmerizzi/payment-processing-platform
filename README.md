# Payment Processing Platform

Plataforma **simulada** de processamento de pagamentos em Go (API, workers, PostgreSQL, SQS, Outbox, webhooks), construída por etapas para estudo. O plano completo está em [`docs/plano.md`](docs/plano.md); decisões em [`docs/decisions`](docs/decisions).

Estado atual: etapa 2 — API REST, persistência no PostgreSQL e idempotência.

## Requisitos

Apenas Docker (com Compose) e `make`. Go não precisa estar instalado: tudo roda na imagem `golang:1.25-alpine`.

## Comandos

```sh
make test              # testes unitários em container
make test-integration  # testes do repositório contra o Postgres do compose
make fmt               # gofmt
make vet               # go vet
make up                # constrói e sobe Postgres, migrations e API
make down              # derruba tudo (o volume com os dados é mantido)
```

A API fica em `http://localhost:8080` e o Postgres em `localhost:5433` (usuário/senha/banco `payments` por padrão; sobrescreva com `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_PORT`, `API_PORT`). As migrations em `migrations/` rodam sozinhas no `make up` (imagem `migrate/migrate`).

## API

`amount` é em centavos (inteiro) e `currency` é um código ISO 4217 de 3 letras. Erros têm o formato `{"error": {"code": "...", "message": "..."}}`.

```sh
# Criar pagamento (Idempotency-Key obrigatória) -> 201
curl -i -X POST localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: pedido-123' \
  -d '{"amount": 1500, "currency": "BRL"}'

# Mesma chave e mesmo corpo -> 200, mesmo pagamento
curl -i -X POST localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: pedido-123' \
  -d '{"amount": 1500, "currency": "BRL"}'

# Mesma chave e corpo diferente -> 422 idempotency_conflict
curl -i -X POST localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: pedido-123' \
  -d '{"amount": 999, "currency": "BRL"}'

# Consultar -> 200 (404 se não existir, 400 se o id não for UUID)
curl -i localhost:8080/payments/<id>
```

Veja [ADR 0002](docs/decisions/0002-idempotencia-na-api.md) para as regras de idempotência.
