# 04: Fully close a trade and calculate its net result

**What to build:** Record a complete exit, calculate directional price profit and both execution fees, and expose the completed trade's decimal net result.

**Blocked by:** 03: Record and inspect a trade with its first entry.

**Status:** resolved

**Spec coverage:** User stories 28, 29, 30, 32, 33, 38, 59, 60, 61, 62.

- [x] A complete exit records actual filled quantity, price, timestamp, and a mandatory exit fee.
- [x] Calculate the exit default from its own quantity and price using the configured exit rate and preserve the fee snapshot.
- [x] Fully closed trades expose directional price profit, total recorded execution fees, and net result for long and short examples.
- [x] With no funding entries, net result uses zero recorded funding without claiming that no payments occurred.
- [x] Both backends retain the completed result across reopening, verified through the shared public API suite.
- [x] Update usage examples for the new complete-trade workflow.


## Answer

Added `CloseTrade` with explicit filled quantity, price, time, and snapshotted exit
fee defaults. Complete closure records its final instant and exposes directional
realized price profit/loss, total execution fees, signed recorded funding, and
net profit/loss in the trade's settlement currency. Absent funding contributes
zero recorded funding; `RecordedCosts` identifies that calculation basis.

The shared public API suite verifies CSV and PostgreSQL reopening, long and short
winners and losers, distinct entry/exit rates, historical fee snapshots, known
zero fees, missing exit rates, invalid fills, and repeated closure rejection.
The worked 6,000-to-6,200 long returns 200 price profit, 4.88 fees, and 195.12 net.
Updated the usage example and decimal/storage documentation.

Validation: `MTL_TEST_POSTGRES_DSN=… go test ./...` passed for both backends;
`git diff --check` passed. Implemented from dependency integration commit 470ca90.
