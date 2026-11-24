GO_IMAGE := golang:1.25-alpine
GO := docker run --rm -v $(CURDIR):/app -w /app -v ppp-gomod:/go/pkg/mod -v ppp-gobuild:/root/.cache/go-build $(GO_IMAGE)
COMPOSE_NETWORK := $(notdir $(CURDIR))_default
PG_USER ?= payments
PG_PASSWORD ?= payments
PG_DB ?= payments

.PHONY: test test-integration fmt vet tidy up down

test:
	$(GO) go test ./...

# Runs the integration tests in a golang container on the compose network,
# against the compose Postgres (migrated first).
test-integration:
	docker compose up -d --wait postgres
	docker compose run --rm migrate
	docker run --rm --network $(COMPOSE_NETWORK) -v $(CURDIR):/app -w /app \
		-v ppp-gomod:/go/pkg/mod -v ppp-gobuild:/root/.cache/go-build \
		-e DATABASE_URL=postgres://$(PG_USER):$(PG_PASSWORD)@postgres:5432/$(PG_DB)?sslmode=disable \
		$(GO_IMAGE) go test -tags integration -count=1 ./internal/infrastructure/...

fmt:
	$(GO) gofmt -l -w .

vet:
	$(GO) go vet ./... && $(GO) go vet -tags integration ./...

tidy:
	$(GO) go mod tidy

up:
	docker compose up -d --build --wait

down:
	docker compose down
