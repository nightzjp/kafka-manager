# Stability Hardening Design

## Goal

Prepare Kafka Manager for long-running internal use by bounding disk growth, preventing writes to protected clusters, and covering core browser workflows with repeatable end-to-end tests.

## Disk retention

- Add `audit.configBackupRetentionDays` with a default of 30 days.
- Remove dated configuration-backup directories older than the configured period at startup and once per day.
- Keep the current date-directory backup layout and Web restore workflow.
- Configure Docker's local JSON log driver with `max-size: 10m` and `max-file: 3`.

This keeps configuration compact by placing both local-data retention values in the existing audit/data section instead of adding a new top-level section.

## Read-only clusters

- Add `readOnly` to each cluster configuration and cluster summary.
- Reject Topic creation/deletion/partition expansion/config changes, message production, Consumer Offset reset, and Consumer Group deletion with HTTP 403 before Kafka mutation.
- Audit denied write attempts.
- Display a read-only badge in the active cluster header and disable write controls while leaving all inspection, message reading, filtering, and live following available.
- Web configuration remains editable so an administrator can intentionally enable or disable read-only mode.

The API is the security boundary; frontend disabling is only explanatory UX.

## Browser end-to-end coverage

- Use Playwright with the installed local Chrome channel.
- Start Vite automatically and mock authenticated API responses, avoiding dependence on a live Kafka cluster.
- Cover login/session bootstrap, sidebar collapse persistence, Topic-to-message navigation, JSON display, read-only write controls, Consumer Group navigation, and theme switching.
- Keep E2E separate from the fast unit-test command and expose it as `pnpm --dir web test:e2e` / `make test-e2e`.

## Validation

Use red-green Go tests for retention cleanup and API read-only enforcement, existing Vitest coverage for frontend state, Playwright for browser workflows, then run Go tests, `go vet`, TypeScript build, production build, and runtime health checks.
