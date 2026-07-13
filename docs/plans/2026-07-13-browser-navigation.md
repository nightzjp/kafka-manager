# Browser Navigation and State Recovery Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make cluster, page, Topic and Topic tab restorable through browser URLs.

**Architecture:** Use a small typed route model and the native History API. Keep route parsing independent from React, then make App derive all navigation state from the current route.

**Tech Stack:** React 18, TypeScript, Vitest, browser History API, existing Go SPA fallback.

---

### Task 1: Route model

**Files:**
- Create: `web/src/navigation/routes.ts`
- Create: `web/src/navigation/routes.test.ts`

1. Write failing parse/build round-trip tests for page and Topic routes.
2. Implement strict route parsing, safe decoding and canonical path generation.
3. Run focused tests.

### Task 2: Route-driven application shell

**Files:**
- Modify: `web/src/App.tsx`

1. Initialize route from `location.pathname` and listen to `popstate`.
2. Replace page, cluster and Topic tab navigation state with route updates.
3. Restore Topic metadata for direct deep links and show loading/error recovery.
4. Update document title from route context.

### Task 3: Verification

1. Run frontend tests, typecheck and production build.
2. Run Go tests and vet.
3. Request a nested URL over HTTP and verify it returns the SPA document.
4. Commit the batch.
