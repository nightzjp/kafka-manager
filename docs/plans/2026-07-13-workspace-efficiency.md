# Workspace Efficiency Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a persistent collapsible desktop sidebar, refresh Topic metadata in place, and restore message filters from shareable URLs.

**Architecture:** Keep UI state helpers as tested pure modules, while App owns sidebar and selected Topic state. MessagesPage initializes and synchronizes filters through the native URL API without adding a router dependency.

**Tech Stack:** React 18, TypeScript, Vitest, native History API, existing Go SPA server.

---

### Task 1: Persistent sidebar preference

**Files:**
- Create: `web/src/navigation/sidebar-state.ts`
- Create: `web/src/navigation/sidebar-state.test.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/Icon.tsx`
- Modify: `web/src/polish.css`

1. Write failing tests for safe storage reads and writes.
2. Implement the storage helper and run focused tests.
3. Add the toggle, accessible labels, custom collapsed hints and responsive layout.
4. Run typecheck and focused tests.

### Task 2: Message filter URL state

**Files:**
- Modify: `web/src/pages/message-filters.ts`
- Modify: `web/src/pages/message-filters.test.ts`
- Modify: `web/src/pages/MessagesPage.tsx`

1. Write failing tests for valid round-trip and malformed URL fallback.
2. Implement URL parsing and known-parameter replacement.
3. Initialize and synchronize MessagesPage filters.
4. Run focused tests and typecheck.

### Task 3: Topic in-place refresh

**Files:**
- Create: `web/src/pages/topic-selection.ts`
- Create: `web/src/pages/topic-selection.test.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/pages/TopicWorkspace.tsx`

1. Write a failing test for exact Topic selection.
2. Extract exact matching and reuse it for deep-link restoration and refresh.
3. Pass refresh into TopicWorkspace, add manual refresh, and await refresh after partition expansion.
4. Run focused and full tests.

### Task 4: Verification and commit

1. Run all Vitest tests and TypeScript typecheck.
2. Run production build, all Go tests, and Go vet.
3. Restart the local binary and verify health and embedded asset hashes.
4. Review the diff, commit, and confirm a clean worktree.
