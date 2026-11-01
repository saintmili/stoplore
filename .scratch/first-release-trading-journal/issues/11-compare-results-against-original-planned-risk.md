# 11: Compare results against original planned risk

**What to build:** Optionally record a fixed whole-trade risk budget and expose completed net results as R multiples without excluding trades missing risk data.

**Blocked by:** 07: Preserve original plans and dated revisions; 10: Report completed-trade performance by setup.

**Status:** ready-for-agent

**Spec coverage:** User stories 55, 56, 57, 58, 59, 60, 61.

- [ ] Record original planned monetary risk for the whole trade and preserve it when plans are revised.
- [ ] Expose R as net profit divided by an available positive original risk budget.
- [ ] The agreed 100 USDT net-profit and 50 USDT risk example yields 2R.
- [ ] Absent or nonpositive risk makes R unavailable while retaining otherwise valid monetary-report eligibility.
- [ ] Both backends preserve the original budget and give matching public report results after reopening.

