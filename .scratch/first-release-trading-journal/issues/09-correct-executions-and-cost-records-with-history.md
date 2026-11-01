# 09: Correct executions and cost records with history

**What to build:** Correct recorded executions, fees, and funding while retaining prior values and recalculating the trade's position, closure state, and results.

**Blocked by:** 05: Scale entries and record partial exits; 06: Record actual fee overrides and optional funding.

**Status:** ready-for-agent

**Spec coverage:** User stories 24, 25, 26, 59, 60, 61, 68.

- [ ] Public correction operations retain previous recorded values and distinguish corrections from new executions.
- [ ] Quantity, price, and execution-time corrections recalculate the affected execution sequence, remaining average, quantities, and completion information.
- [ ] Fee and funding corrections recalculate the recorded-cost net result without rewriting unrelated historical defaults.
- [ ] Reject invalid corrections with meaningful errors rather than persist contradictory trade data.
- [ ] Both backends preserve corrected records, their prior values, and recalculated results across reopening.
- [ ] Verify corrected results and visible history through public API reads; reporting consumes those corrected records in its own slice.

