# 01: Configure exchanges, accounts, and fee defaults

**What to build:** Create and retrieve exchange accounts and separate entry/exit percentage fee settings, using decimal values that survive reopening either backend.

**Blocked by:** None (can start immediately).

**Status:** resolved

**Spec coverage:** User stories 4, 27, 59, 60, 61, 68.

- [x] Public operations create and retrieve exchanges and their accounts, with meaningful validation and missing-record errors.
- [x] Separate entry and exit percentage rates preserve exact decimal values; an explicit zero remains distinguishable from missing configuration.
- [x] Both CSV and PostgreSQL preserve settings and identities across reopening.
- [x] Document the public decimal representation contract for these settings and preserve existing journal source records.
- [x] Establish the shared public API behavior suite while verifying this configuration workflow.


## Answer

Implemented exchange creation, lookup, listing, replacement edits and guarded
deletion, plus exchange accounts with stable UUID identity. Separate optional
entry/exit percentage defaults preserve exact decimal values and distinguish
known zero from missing rates. Invalid names/rates, duplicate names and missing
records return meaningful errors.

Added `Decimal` and string constructors, documenting percentage units, canonical
formatting and 18-place half-away-from-zero division in
`docs/decimal-and-storage.md`. New records use a versioned document in separate
CSV/PostgreSQL storage; legacy CSV files and PostgreSQL trade records remain
available through their original API and are not rewritten by configuration.

Established the shared public Go behavior harness, including isolated PostgreSQL
schemas, reopening each backend, exact fee values, account identity and validation
workflows. Recorded initial red compilation errors for missing public
capabilities, then green `go test ./...` against CSV and PostgreSQL using
`STOPLORE_TEST_POSTGRES_DSN`. Merged the latest integration branch including ticket02.
