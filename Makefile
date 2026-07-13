GO ?= go

.PHONY: dev-backend dev-frontend test build

dev-backend:
	$(GO) run ./cmd/kafka-manager

dev-frontend:
	pnpm --dir web dev

test:
	$(GO) test ./...
	pnpm --dir web test --run

build:
	pnpm --dir web build
	$(GO) build -o build/kafka-manager ./cmd/kafka-manager
