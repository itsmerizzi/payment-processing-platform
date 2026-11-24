GO_IMAGE := golang:1.25-alpine
GO := docker run --rm -v $(CURDIR):/app -w /app -v ppp-gomod:/go/pkg/mod -v ppp-gobuild:/root/.cache/go-build $(GO_IMAGE)

.PHONY: test fmt vet up down

test:
	$(GO) go test ./...

fmt:
	$(GO) gofmt -l -w .

vet:
	$(GO) go vet ./...

up:
	docker compose up -d --wait

down:
	docker compose down
