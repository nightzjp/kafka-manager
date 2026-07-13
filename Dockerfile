# syntax=docker/dockerfile:1

FROM node:22-alpine AS frontend
WORKDIR /src/web
RUN corepack enable && corepack prepare pnpm@10.15.1 --activate
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.25-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN rm -rf internal/webassets/dist-built && mkdir -p internal/webassets/dist-built
COPY --from=frontend /src/web/dist/ internal/webassets/dist-built/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/kafka-manager .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1000 -S kafka-manager \
    && adduser -u 1000 -S -D -H -G kafka-manager kafka-manager
WORKDIR /app
COPY --from=backend --chown=kafka-manager:kafka-manager /out/kafka-manager /app/kafka-manager
USER kafka-manager:kafka-manager
EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8080/api/v1/health >/dev/null || exit 1
ENTRYPOINT ["/app/kafka-manager"]
CMD ["--config", "/app/config.yaml"]
