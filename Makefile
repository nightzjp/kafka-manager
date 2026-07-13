GO ?= go

.PHONY: dev-backend dev-frontend test test-e2e build

dev-backend:
	$(GO) run . --config $${CONFIG:-./config.yaml}

dev-frontend:
	pnpm --dir web dev

test:
	$(GO) test ./...
	$(GO) vet ./...
	pnpm --dir web test --run

test-e2e:
	pnpm --dir web test:e2e

build:
	pnpm --dir web build
	mkdir -p internal/webassets/dist-built
	cp -R web/dist/. internal/webassets/dist-built/
	mkdir -p build
	$(GO) build -o build/kafka-manager .
