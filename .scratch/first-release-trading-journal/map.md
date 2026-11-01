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

- [03: Record and inspect a trade with its first entry](issues/03-record-and-inspect-a-trade-with-its-first-entry.md)
  resolved: public record/get/list API preserves explicit fills, account and setup
  version context, optional margin/leverage, and snapshotted entry fees. Shared
  CSV/PostgreSQL behaviors verify reopen, errors, decimal inputs, and 18-place
  half-away-from-zero calculated fees.

- [04: Fully close a trade and calculate its net result](issues/04-fully-close-a-trade-and-calculate-its-net-result.md)
  resolved: complete exits snapshot their own fee defaults and retain directional
  realized profit, execution costs, completion time, and recorded-cost net result.
  Shared CSV/PostgreSQL behavior verifies reopening, long/short results, distinct
  rates, explicit zero fees, and meaningful rejection without journal changes.

- [07: Preserve original plans and dated revisions](issues/07-preserve-original-plans-and-dated-revisions.md)
  resolved: optional original plans and dated full-plan snapshots preserve decision
  history; corrections keep before/after values and explanations without adding
  decisions. Shared public API tests cover CSV and PostgreSQL reopening.

- [08: Tag trades and review rule adherence](issues/08-tag-trades-and-review-rule-adherence.md)
  resolved: contextual tags and optional followed/deviated/unassessed reviews
  retain previous tags, assessments, notes, and UTC correction timestamps.
  Shared public API tests verify CSV/PostgreSQL reopening and invalid updates.

- [06: Record actual fee overrides and optional funding](issues/06-record-actual-fee-overrides-and-optional-funding.md)
  resolved: per-execution actual fees replace defaults while preserving unknown or
  known default snapshots; signed timestamped funding adjusts net results and
  remains optional. Shared CSV/PostgreSQL behaviors verify exact amounts,
  recorded-cost basis, meaningful errors, and reopening.

- [05: Scale entries and record partial exits](issues/05-scale-entries-and-record-partial-exits.md)
  resolved: multiple entries average against remaining exposure; partial exits
  realize directional profit and final exits complete the trading idea. Shared
  CSV/PostgreSQL API behaviors retain fills and fees across reopening, validate
  sequences, and verify the documented decimal rounding policy.
