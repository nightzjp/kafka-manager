# Kafka Message Filtering Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Filter scanned Kafka messages by Key, raw Value, and typed JSON field conditions in both bounded queries and live streams.

**Architecture:** Validate filters in the message service and apply them inside the Kafka fetch loop to keep memory bounded. Serialize the same typed filter model from the React query form into normal HTTP and SSE requests.

**Tech Stack:** Go, franz-go, encoding/json, React 18, TypeScript, Vitest.

---

### Task 1: Filtering semantics

**Files:**
- Create: `internal/kafka/message/filter.go`
- Create: `internal/kafka/message/filter_test.go`
- Modify: `internal/kafka/message/service.go`
- Modify: `internal/kafka/message/service_test.go`

Write failing tests, implement Key/Value/JSON matching and validate operators, paths, condition count and scan bounds.

### Task 2: Bounded Kafka scanning and API

**Files:**
- Modify: `internal/kafka/message/kafka_backend.go`
- Modify: `internal/api/server.go`
- Modify: `internal/kafka/message/service_test.go`

Return scan metadata, filter records inside fetch batches, parse query parameters, and pass filters into SSE queries.

### Task 3: Frontend filter model

**Files:**
- Create: `web/src/pages/message-filters.ts`
- Create: `web/src/pages/message-filters.test.ts`

Test and implement active condition counting and URL parameter serialization.

### Task 4: Message filter interface

**Files:**
- Modify: `web/src/pages/MessagesPage.tsx`
- Modify: `web/src/styles.css`
- Modify: `web/src/polish.css`

Add advanced filters, scan/result statistics, multiple JSON conditions, reset behavior, and identical SSE parameters.

### Task 5: Verification

Run focused red-green tests, full Go and frontend suites, typecheck, production build, vet, runtime health check, then commit.
