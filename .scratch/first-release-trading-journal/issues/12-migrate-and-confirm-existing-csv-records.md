# 12: Migrate and confirm existing CSV records

**What to build:** Import current and legacy MTL CSV records into either backend, preserve their source and original information, and review and confirm missing or inferred essentials before reporting.

**Blocked by:** 07: Preserve original plans and dated revisions; 10: Report completed-trade performance by setup.

**Status:** resolved

**Spec coverage:** User stories 53, 60, 61, 63, 64, 65, 66, 67.

- [x] A public migration operation accepts supported current and legacy MTL CSV records and can target either configured backend.
- [x] Preserve source records and all available original information; show imported records in the journal immediately.
- [x] Identify missing or inferred essentials such as account, currency, quantity, and fees and keep affected records out of performance statistics pending confirmation.
- [x] A public review-and-confirmation workflow makes completed records eligible when their essential information is confirmed and fees are complete.
- [x] Retain unknown historical setup rules and plans as unknown and keep missing funding optional, with no zero-funding confirmation required.
- [x] Expose exclusion counts and demonstrate that confirmation updates reports without modifying the source.
- [x] Use representative synthetic fixtures and verify reopened data and reports through the public API, establishing the shared migration workflow for additional sources.


## Answer

Implemented `ImportCSV`, `ReviewMigration`, and `ConfirmMigration` through the
public service API. The source reader validates current and legacy headers and
reads directly without initializing or writing the old storage. Imports are
atomic and retryable by source identity plus original record ID; changed source
evidence is rejected. A shared normalized-record importer supports the later
PostgreSQL source adapter.

Immediately visible trades retain every original field, source format/order,
explicit inferred-quantity provenance, and durable pending essentials. Historical
setup versions retain their names with unknown rules; original plan history stays
unknown while existing stop/target values remain evidence. Partial historical
exits remain ongoing. Nontrade source records require explicit classification.

Confirmation validates the account, supported settlement currency, complete
execution quantities/prices/times and explicit historical fees, and atomically
replays exposure and recorded costs. Current exchange defaults cannot substitute
for unknown historical fees. Source evidence survives confirmation and later
reopening; funding remains optional. Further edits use ordinary corrections.

Synthetic public API acceptance tests cover current/legacy sources into CSV and
isolated PostgreSQL targets, reopen, unknown rules/plans, inferred quantities,
partial and completed records, long/short direction, USDT/USDC, zero and unknown
fees, invalid batch/overexit atomicity, source conflicts/retries and byte-identical
source preservation. Report tests demonstrate excluded completed imports becoming
eligible with exact net results after confirmation.
