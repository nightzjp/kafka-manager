# Stability Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Bound all local disk growth, enforce per-cluster read-only safety, and add deterministic browser workflow tests.

**Architecture:** Extend the existing YAML model without adding a top-level section, centralize Kafka mutation authorization in the API server, propagate read-only metadata to React, and run Playwright against Vite with mocked APIs. Preserve the single-binary runtime and real-Kafka independence of automated browser tests.

**Tech Stack:** Go 1.25, React 18, TypeScript, Vitest, Playwright, Docker Compose

---

### Task 1: Bound configuration backups and Docker logs

**Files:**
- Modify: `internal/config/model.go`
- Modify: `internal/config/load.go`
- Modify: `internal/config/validate.go`
- Modify: `internal/config/store.go`
- Modify: `internal/config/store_test.go`
- Modify: `internal/config/config_test.go`
- Modify: `main.go`
- Modify: `config.example.yaml`
- Modify: `web/src/lib/types.ts`
- Modify: `web/src/pages/SettingsPage.tsx`
- Modify: `docker-compose.yaml`
- Modify: `docs/configuration.md`
- Modify: `docs/deployment.md`

1. Write failing tests for the default/validation and dated backup-directory cleanup.
2. Run focused Go tests and confirm the expected failures.
3. Add `configBackupRetentionDays`, cleanup implementation, startup/daily scheduling, Web setting, documentation, and Compose log rotation.
4. Run config tests, frontend typecheck, and `docker compose config` when available.

### Task 2: Enforce and explain read-only clusters

**Files:**
- Modify: `internal/config/model.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/server_test.go`
- Modify: `config.example.yaml`
- Modify: `web/src/lib/types.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/pages/TopicsPage.tsx`
- Modify: `web/src/pages/TopicWorkspace.tsx`
- Modify: `web/src/pages/MessagesPage.tsx`
- Modify: `web/src/pages/ConsumersPage.tsx`
- Modify: `web/src/pages/SettingsPage.tsx`
- Modify: `web/src/polish.css`
- Modify: `docs/configuration.md`

1. Write a failing authenticated API test proving a read-only cluster rejects every mutation with HTTP 403 before contacting Kafka.
2. Run the focused test and confirm it fails for the current offline/400 behavior.
3. Add a centralized writable-cluster guard and apply it to all Kafka mutation handlers, recording rejected audits.
4. Propagate `readOnly` through summaries and React props; show the state and disable write controls.
5. Run focused API tests, all frontend unit tests, and TypeScript checks.

### Task 3: Add deterministic browser workflows

**Files:**
- Modify: `web/package.json`
- Modify: `pnpm-lock.yaml`
- Create: `web/playwright.config.ts`
- Create: `web/e2e/core-workflows.spec.ts`
- Modify: `Makefile`
- Modify: `docs/development.md`

1. Add Playwright as a development dependency and a Chrome-based configuration with Vite web-server startup.
2. Mock auth, dashboard, Topic, message, Consumer Group, configuration, and audit APIs in the browser test.
3. Test sidebar persistence, theme change, Topic message workflow/JSON rendering, read-only controls, and Consumer Group detail navigation.
4. Run `pnpm --dir web test:e2e` and fix only failures demonstrated by the workflow.

### Task 4: Complete verification and delivery

1. Run `git diff --check`.
2. Run `env GOCACHE=/private/tmp/codex-go-cache make test GO=/Users/nightz/sdk/go1.25.1/bin/go`.
3. Run `pnpm --dir web test:e2e`.
4. Run `env GOCACHE=/private/tmp/codex-go-cache make build GO=/Users/nightz/sdk/go1.25.1/bin/go`.
5. Restart the local binary, verify `/api/v1/health`, commit the implementation, and confirm a clean worktree.
