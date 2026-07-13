# Reliability and Operation Feedback Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Prevent full-page crashes and replace silent/native operation feedback with accessible, consistent in-product interactions.

**Architecture:** Wrap the React application with a class error boundary and a lightweight feedback context. Keep confirmation dialogs controlled by each feature so generic UI remains separate from Kafka request logic.

**Tech Stack:** React 18, TypeScript, Vitest, existing CSS design tokens, Go embedded frontend.

---

### Task 1: Application error boundary

**Files:**
- Create: `web/src/components/AppErrorBoundary.tsx`
- Create: `web/src/components/AppErrorBoundary.test.ts`
- Modify: `web/src/main.tsx`
- Modify: `web/src/styles.css`

1. Write tests for deriving the failed state and invoking page reload.
2. Run `pnpm --dir web test --run web/src/components/AppErrorBoundary.test.ts` and verify failure because the component does not exist.
3. Implement the boundary and recovery screen.
4. Re-run the focused test and verify it passes.

### Task 2: Global feedback center

**Files:**
- Create: `web/src/components/Feedback.tsx`
- Create: `web/src/components/feedback-model.ts`
- Create: `web/src/components/feedback-model.test.ts`
- Modify: `web/src/main.tsx`
- Modify: `web/src/styles.css`

1. Write model tests for adding bounded messages and dismissing a message.
2. Run the focused test and verify failure because the model does not exist.
3. Implement the model, provider, hook and accessible toast stack.
4. Re-run focused tests and verify they pass.

### Task 3: Typed confirmation dialog

**Files:**
- Modify: `web/src/components/Common.tsx`
- Create: `web/src/components/confirmation-model.ts`
- Create: `web/src/components/confirmation-model.test.ts`
- Modify: `web/src/styles.css`

1. Write tests for exact typed confirmation matching.
2. Run the focused test and verify failure.
3. Implement the model and reusable confirmation dialog with pending/error states.
4. Re-run focused tests and verify they pass.

### Task 4: Integrate operation feedback

**Files:**
- Modify: `web/src/pages/TopicsPage.tsx`
- Modify: `web/src/pages/TopicWorkspace.tsx`
- Modify: `web/src/pages/ConsumersPage.tsx`
- Modify: `web/src/pages/SettingsPage.tsx`

1. Replace native prompt/confirm calls with controlled dialogs.
2. Add success/failure feedback to destructive actions, saves, restores and clipboard copy.
3. Verify `rg "\\b(confirm|prompt)\\(" web/src` returns no matches.

### Task 5: Full verification

1. Run `pnpm --dir web test --run`.
2. Run `pnpm --dir web typecheck`.
3. Run `pnpm --dir web build`.
4. Run `go test ./...` and `go vet ./...`.
5. Review `git diff --check`, status, and commit the completed batch.
