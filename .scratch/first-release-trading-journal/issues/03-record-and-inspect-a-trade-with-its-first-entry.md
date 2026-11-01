# 03: Record and inspect a trade with its first entry

**What to build:** Manually record a long or short trade linked to its account, settlement currency, setup version, and first filled execution; inspect its quantity and snapshotted entry fee.

**Blocked by:** 01: Configure exchanges, accounts, and fee defaults; 02: Define and version trading setups.

**Status:** resolved

**Spec coverage:** User stories 1, 2, 3, 5, 6, 10, 11, 13, 15, 17, 28, 29, 30, 32, 33, 59, 60, 61, 62, 68.

- [x] A trade represents one idea in one instrument and direction and references an exchange account, settlement currency, and primary setup version.
- [x] Record explicit filled quantity, price, and timestamp; retain margin and leverage separately.
- [x] Use decimal financial values with a documented precision and rounding contract for exposed calculations.
- [x] The entry gets its mandatory fee from its exchange's entry rate and its own execution value; explicit zero is valid and unknown fees are not silently zero.
- [x] Preserve the fee settings and calculated amount against later exchange-setting edits.
- [x] Public get/list operations and reopening both backends preserve the trade and execution; invalid requests return meaningful errors.
- [x] Update the public usage example to demonstrate recording and inspecting this workflow.


## Answer

Added `Service.RecordTrade`, `GetTrade`, and `ListTrades` with explicit long/short
ideas, account and setup-version UUID references, USDT/USDC settlement, and a
first filled execution. Quantity and price remain independent of optional margin
and leverage. Entry fees snapshot the configured percentage and calculated
amount; unknown defaults reject recording while explicit zero remains valid.

The shared public API suite verifies both directions, exact decimal and timestamp
round trips, stable setup/fee history after edits, deterministic list/get/reopen
behavior, invalid requests, unknown/zero fees, and half-away-from-zero rounding
to 18 fractional places. CSV and PostgreSQL pass via `go test ./...` with
`STOPLORE_TEST_POSTGRES_DSN` configured. Updated `examples/main.go` and
`docs/decimal-and-storage.md`. Legacy sources remain untouched.
