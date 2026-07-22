# Chat Request Lease Renewal Fix

## Problem

A chat request can persist the user message and transition to `accepted` without
creating its generation task. The UI then receives repeated `accepted` replies
but no model content until the two-minute request lease expires and recovery
claims the request.

The failure occurs when the initial lease renewal writes the same millisecond
expiry that is already stored. MySQL reports zero changed rows for that no-op
update, and the backend incorrectly interprets zero as loss of lease ownership.

## Scope

- Fix lease renewal semantics in the Go backend.
- Preserve owner-token and completed-status protections.
- Add regression coverage for lease ownership and repeated renewal.
- Do not change the frontend, model adapters, request protocol, or lease duration.

## Design

`RenewChatRequestLease` will always advance `lease_expires_at` when the matching
owner still holds a non-completed request. The SQL assignment will choose the
normal two-minute candidate expiry when it is newer; otherwise it will increment
the stored expiry by one millisecond. A matching row therefore produces one
changed row on MySQL, while an owner mismatch or completed request still produces
zero rows.

This keeps `RowsAffected() == 1` as a valid ownership check without enabling
`clientFoundRows` globally or weakening the compare-by-owner condition.

## Verification

- Extend request lease tests to cover immediate repeated renewal, owner mismatch,
  and completed requests.
- Run `go test ./...` and `go build .`.
- Repeat the MySQL no-op probe against a temporary table to confirm the renewed
  SQL always changes the owned row.

## Rollback

The change is isolated to lease-renewal SQL and its tests. Reverting the code
commit restores the previous behavior without a schema or data migration.
