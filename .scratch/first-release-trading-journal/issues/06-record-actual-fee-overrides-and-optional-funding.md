# 06: Record actual fee overrides and optional funding

**What to build:** Replace a default fee with an actual charge or rebate and record funding received or paid, with net results explicitly using recorded costs.

**Blocked by:** 04: Fully close a trade and calculate its net result.

**Status:** resolved

**Spec coverage:** User stories 31, 34, 35, 36, 37, 38, 60, 61.

- [x] An actual-fee override replaces the default fee rather than adding another charge, while preserving the recorded default information.
- [x] Support discounts and rebates without silently changing other executions.
- [x] Funding received or paid has an amount and timestamp and adjusts the trade's net result.
- [x] Absent funding entries mean zero recorded funding and require no zero-funding confirmation.
- [x] Expose the recorded-cost basis without interpreting missing funding as evidence that no payment occurred.
- [x] For 200 USDT price profit and 4.88 USDT fees, receiving 1 USDT and paying 5 USDT funding yields 191.12 USDT net; no recorded funding yields 195.12 USDT.
- [x] Verify the workflow through the public API and reopen both backends to confirm persistence.


## Answer

Implemented optional `ExecutionRequest.ActualFee` to replace a default with an
actual charge, explicit zero, discount, or rebate. Execution fees retain default
rate and calculated amount independently from the actual and effective amounts;
a missing default stays unknown when the actual fee supplies the required cost.
`RecordFundingPayment` records signed settlement-currency payments with IDs and
precise timestamps, including historical payments entered after closure. Net
results include recorded funding, expose `RecordedCosts`, and need no absent
funding confirmation. Decimal calculations retain the documented 18-place
half-away-from-zero division rule; entered actual fees and funding remain exact.

Shared public API behaviors verify discounts/rebates, unknown defaults and explicit
zero, unchanged execution snapshots, funding receipts/payments/absence, meaningful
errors, and CSV/PostgreSQL reopening. The full suite passes with the isolated local
PostgreSQL DSN configured.
