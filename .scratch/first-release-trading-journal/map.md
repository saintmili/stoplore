# First-release implementation map

Integration branch: `integration/first-release-journal`
Baseline: `0145569`

## Task graph

- 01 and 02 have no blockers.
- 03 follows 01 and 02.
- 04, 07, and 08 follow 03.
- 05 and 06 follow 04.
- 09 follows 05 and 06.
- 10 follows 08 and 09.
- 11 and 12 follow 07 and 10.
- 13 follows 12.

## Decisions

The approved test seam is the public Go library API, shared across CSV and an
isolated PostgreSQL database. Each ticket is implemented in its own worktree and
merged into the integration branch after its dependencies are resolved.


- [02: Define and version trading setups](issues/02-define-and-version-trading-setups.md)
  resolved: named setups retain ordered rule versions, each with its own UUID.
  Public setup methods preserve earlier rules across CSV and PostgreSQL reopen;
  trades use setup-version identity for historical context.

- [01: Configure exchanges, accounts, and fee defaults](issues/01-configure-exchanges-accounts-and-fee-defaults.md)
  resolved: exact decimal entry/exit percentages, account UUIDs, meaningful
  validation, CSV/PostgreSQL reopen behavior and shared public API harness.
  New versioned storage preserves legacy source records. Decimal division is
  explicitly rounded to 18 fractional places, half away from zero.
