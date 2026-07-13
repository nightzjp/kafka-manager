# Message Order and Sidebar Control Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Return recent Kafka messages newest-first and move the sidebar collapse action to the bottom-left utility area.

**Architecture:** Normalize `latest` ordering inside the Go record collector, leaving forward-reading modes unchanged. Recompose the existing React sidebar footer so health and collapse controls share a bottom utility section without changing stored state or mobile behavior.

**Tech Stack:** Go, React 18, TypeScript, CSS, Go testing, Vitest

---

### Task 1: Lock recent-message ordering with a regression test

**Files:**
- Modify: `internal/kafka/message/filter_test.go`
- Modify: `internal/kafka/message/filter.go`

**Step 1: Write the failing test**

Add a collector test with unfiltered `latest` records at increasing timestamps and assert the result is returned in decreasing timestamp/offset order.

**Step 2: Run test to verify it fails**

Run: `/Users/nightz/sdk/go1.25.1/bin/go test ./internal/kafka/message -run TestLatestCollectorReturnsNewestFirst -v`

Expected: FAIL because the collector currently preserves Kafka's ascending fetch order.

**Step 3: Write minimal implementation**

Sort every `latest` result with the existing `recordNewer` comparator. Keep the filtered collector's bounded newest-match selection and result-limit calculation intact.

**Step 4: Run focused and package tests**

Run: `/Users/nightz/sdk/go1.25.1/bin/go test ./internal/kafka/message -v`

Expected: PASS.

### Task 2: Move the sidebar control into the footer

**Files:**
- Modify: `web/src/App.tsx`
- Modify: `web/src/polish.css`
- Test: `web/src/navigation/sidebar-state.test.ts`

**Step 1: Recompose markup**

Create a bottom utility container containing cluster health and the existing collapse button. Show `收起侧栏` / `展开侧栏` text where space permits, and preserve ARIA labels and state persistence.

**Step 2: Replace floating styles**

Remove absolute positioning and style the control as a full-width footer action when expanded and a centered icon action when collapsed. Continue hiding it below 820px.

**Step 3: Verify frontend behavior contracts**

Run: `pnpm --dir web test --run && pnpm --dir web run typecheck`

Expected: all tests and TypeScript checks pass.

### Task 3: Full verification and delivery

**Files:**
- Verify all changed files

**Step 1: Run the complete test suite**

Run: `env GOCACHE=/private/tmp/codex-go-cache make test GO=/Users/nightz/sdk/go1.25.1/bin/go`

Expected: Go tests, `go vet`, and Vitest pass.

**Step 2: Build the production binary**

Run: `env GOCACHE=/private/tmp/codex-go-cache make build GO=/Users/nightz/sdk/go1.25.1/bin/go`

Expected: Vite and Go builds exit successfully.

**Step 3: Restart and check health**

Restart `build/kafka-manager --config ./config.yaml`, then request `/api/v1/health`.

Expected: `{"status":"ok"}`.

**Step 4: Commit**

Commit the implementation and tests with a focused bug-fix message.
