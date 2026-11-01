# 06: Record actual fee overrides and optional funding

**What to build:** Replace a default fee with an actual charge or rebate and record funding received or paid, with net results explicitly using recorded costs.

**Blocked by:** 04: Fully close a trade and calculate its net result.

**Status:** ready-for-agent

**Spec coverage:** User stories 31, 34, 35, 36, 37, 38, 60, 61.

- [ ] An actual-fee override replaces the default fee rather than adding another charge, while preserving the recorded default information.
- [ ] Support discounts and rebates without silently changing other executions.
- [ ] Funding received or paid has an amount and timestamp and adjusts the trade's net result.
- [ ] Absent funding entries mean zero recorded funding and require no zero-funding confirmation.
- [ ] Expose the recorded-cost basis without interpreting missing funding as evidence that no payment occurred.
- [ ] For 200 USDT price profit and 4.88 USDT fees, receiving 1 USDT and paying 5 USDT funding yields 191.12 USDT net; no recorded funding yields 195.12 USDT.
- [ ] Verify the workflow through the public API and reopen both backends to confirm persistence.

