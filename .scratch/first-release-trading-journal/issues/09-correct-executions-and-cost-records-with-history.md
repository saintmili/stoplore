# 09: Correct executions and cost records with history

**What to build:** Correct recorded executions, fees, and funding while retaining prior values and recalculating the trade's position, closure state, and results.

**Blocked by:** 05: Scale entries and record partial exits; 06: Record actual fee overrides and optional funding.

**Status:** resolved

**Spec coverage:** User stories 24, 25, 26, 59, 60, 61, 68.

- [x] Public correction operations retain previous recorded values and distinguish corrections from new executions.
- [x] Quantity, price, and execution-time corrections recalculate the affected execution sequence, remaining average, quantities, and completion information.
- [x] Fee and funding corrections recalculate the recorded-cost net result without rewriting unrelated historical defaults.
- [x] Reject invalid corrections with meaningful errors rather than persist contradictory trade data.
- [x] Both backends preserve corrected records, their prior values, and recalculated results across reopening.
- [x] Verify corrected results and visible history through public API reads; reporting consumes those corrected records in its own slice.


## Answer

Added public `CorrectExecution`, `CorrectExecutionFee`, and `CorrectFundingPayment`
operations. Trade reads expose dated execution and funding correction histories
with complete previous records, preserving identities rather than adding fills.
Execution corrections replay a stable chronological sequence, retain exact first
entry prices, recompute remaining weighted averages and directional realized
results, and update or clear completion timestamps. Invalid chronological exits,
overexits, and executions after full closure fail atomically.

Corrected default fees use their execution's snapshotted rate. Existing actual
fee overrides remain effective unless explicitly replaced or reset through the
fee correction operation; resetting requires a known historical default.
Unrelated fees and exchange settings are unchanged. Signed funding corrections
update recorded-cost results and preserve prior amounts and timestamps.

Validation: shared public API tests against CSV and PostgreSQL cover reopen,
long/short results, quantity/price/time corrections, sequence reordering, closing
instant changes, returning a completed trade to an open position, fee overrides,
historical-rate edits, exact prices, optional funding, and atomic invalid requests.
`go test ./...` passed with the isolated PostgreSQL DSN configured.
