# Payment Processing Platform

Plataforma **simulada** de processamento de pagamentos em Go (API, workers, PostgreSQL, SQS, Outbox, webhooks), construída por etapas para estudo. O plano completo está em [`docs/plano.md`](docs/plano.md); decisões em [`docs/decisions`](docs/decisions).

Estado atual: etapa 1 — domínio (`internal/domain`) e PostgreSQL local.

## Requisitos

Apenas Docker (com Compose) e `make`. Go não precisa estar instalado: tudo roda na imagem `golang:1.25-alpine`.

## Comandos

```sh
make test   # go test ./... em container
make fmt    # gofmt
make vet    # go vet
make up     # sobe o PostgreSQL e espera ficar healthy
make down   # derruba o PostgreSQL (o volume com os dados é mantido)
```

O Postgres sobe em `localhost:5433` (usuário/senha/banco `payments` por padrão; sobrescreva com `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_PORT`).
