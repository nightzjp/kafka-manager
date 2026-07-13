# Message Order and Sidebar Control Design

## Goal

Make message chronology predictable and place the desktop sidebar control where users expect it.

## Confirmed behavior

- `latest` queries return newest records first, ordered by timestamp, partition, then offset.
- `earliest`, `offset`, and `timestamp` queries retain Kafka's forward order.
- Live records continue to appear at the top of the list.
- The sidebar collapse control moves into the bottom-left utility area, below cluster health.
- Expanded navigation shows a label; collapsed navigation shows only the arrow with a tooltip.
- The saved collapse preference and mobile drawer behavior remain unchanged.

## Architecture

Ordering is normalized in the backend collector so API clients receive the same semantics; the React UI does not apply a second, potentially inconsistent sort. The sidebar control remains in `App` but becomes part of the sidebar footer flow instead of an absolutely positioned floating control.

## Validation

- A Go regression test proves unfiltered `latest` results are newest-first.
- Existing filtered-latest tests continue to pass.
- Frontend state tests, TypeScript compilation, full Go tests, `go vet`, production build, and runtime health check pass.
