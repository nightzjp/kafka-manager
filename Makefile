GO ?= go

.PHONY: dev-backend dev-frontend test build

dev-backend:
	$(GO) run ./cmd/kafka-manager --config $${CONFIG:-./config.yaml}

dev-frontend:
	pnpm --dir web dev

test:
	$(GO) test ./...
	$(GO) vet ./...
	pnpm --dir web test --run

build:
	pnpm --dir web build
	mkdir -p internal/webassets/dist-built
	cp -R web/dist/. internal/webassets/dist-built/
	mkdir -p build
	$(GO) build -o build/kafka-manager ./cmd/kafka-manager
