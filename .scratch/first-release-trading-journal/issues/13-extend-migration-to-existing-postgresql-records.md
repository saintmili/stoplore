# 13: Extend migration to existing PostgreSQL records

**What to build:** Use the established migration review and confirmation workflow for existing StopLore PostgreSQL records, targeting either backend while preserving the source database records.

**Blocked by:** 12: Migrate and confirm existing CSV records.

**Status:** resolved

**Spec coverage:** User stories 53, 60, 61, 63, 64, 65, 66, 67.

- [x] Extend the public migration workflow with a source reader for the existing PostgreSQL record model.
- [x] Support importing into either CSV or PostgreSQL without rewriting source records or dropping available original information.
- [x] Imported records use the same immediate visibility, missing/inferred-information review, and confirmation-driven report eligibility as CSV imports.
- [x] Preserve available notes and historical information without inventing unavailable setup rules, plans, or costs.
- [x] Funding remains optional and absent funding requires no zero-funding confirmation.
- [x] Verify source preservation, public review operations, reopened records, and report results with isolated PostgreSQL fixtures and both target backends.


## Answer

Added `Service.ImportPostgres` with an explicit nonsecret source identity, a
caller-owned source pool, and a separate source notes key. A read-only repeatable
read transaction selects the legacy 26-field model without storage initialization
or source writes, then delegates to the shared normalized importer. Original
source fields and ciphertext remain evidence alongside decrypted review notes.
Stored float/timestamp precision is retained without inventing historical input
precision.

The shared public API acceptance workflow verifies plaintext/encrypted synthetic
sources into both CSV and PostgreSQL destinations, immediate visibility, persisted
missing/inferred essentials, unknown rules/plans/fees, review and explicit
confirmation, optional funding, reopened evidence and reports, and idempotency.
The worked result is 19.7 USDC after actual fees. Source row snapshots remain
unchanged. Additional cases reject wrong keys, malformed later rows without
partial imports, absent pools/identities, and missing source tables without
creating source storage.

Validation: red compilation before the public adapter existed; then
`STOPLORE_TEST_POSTGRES_DSN=<isolated local fixture> GOCACHE=/tmp/stoplore-ticket13-cache go test ./...`
passed for the full suite, including both target backends.
