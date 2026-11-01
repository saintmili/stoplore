# 03: Record and inspect a trade with its first entry

**What to build:** Manually record a long or short trade linked to its account, settlement currency, setup version, and first filled execution; inspect its quantity and snapshotted entry fee.

**Blocked by:** 01: Configure exchanges, accounts, and fee defaults; 02: Define and version trading setups.

**Status:** ready-for-agent

**Spec coverage:** User stories 1, 2, 3, 5, 6, 10, 11, 13, 15, 17, 28, 29, 30, 32, 33, 59, 60, 61, 62, 68.

- [ ] A trade represents one idea in one instrument and direction and references an exchange account, settlement currency, and primary setup version.
- [ ] Record explicit filled quantity, price, and timestamp; retain margin and leverage separately.
- [ ] Use decimal financial values with a documented precision and rounding contract for exposed calculations.
- [ ] The entry gets its mandatory fee from its exchange's entry rate and its own execution value; explicit zero is valid and unknown fees are not silently zero.
- [ ] Preserve the fee settings and calculated amount against later exchange-setting edits.
- [ ] Public get/list operations and reopening both backends preserve the trade and execution; invalid requests return meaningful errors.
- [ ] Update the public usage example to demonstrate recording and inspecting this workflow.

