# 11: Compare results against original planned risk

**What to build:** Optionally record a fixed whole-trade risk budget and expose completed net results as R multiples without excluding trades missing risk data.

**Blocked by:** 07: Preserve original plans and dated revisions; 10: Report completed-trade performance by setup.

**Status:** resolved

**Spec coverage:** User stories 55, 56, 57, 58, 59, 60, 61.

- [x] Record original planned monetary risk for the whole trade and preserve it when plans are revised.
- [x] Expose R as net profit divided by an available positive original risk budget.
- [x] The agreed 100 USDT net-profit and 50 USDT risk example yields 2R.
- [x] Absent or nonpositive risk makes R unavailable while retaining otherwise valid monetary-report eligibility.
- [x] Both backends preserve the original budget and give matching public report results after reopening.


## Answer

`RecordTradeRequest.OriginalPlannedRisk` stores an optional monetary budget for
its whole trade, separate from plans and unchanged by decision revisions.
`ReportPerformance.Completed` exposes each eligible completed trade's net result,
settlement currency, setup version, original budget, and optional `RMultiple`.
Positive budgets divide net profit using the documented 18-place rounding rule;
missing, zero, or negative budgets leave R unavailable without exclusion.

`CorrectOriginalPlannedRisk` fixes erroneous data with a required timestamp and
reason, retaining before/after budgets in `Trade.RiskCorrections`. This remains
separate from changing a trading decision. Reports recalculate against the
corrected baseline.

Shared public API tests pass against CSV and isolated PostgreSQL, including
reopening, plan revision preservation, 100 USDT net / 50 USDT risk = 2R,
corrections and unavailable risk with retained monetary eligibility. Full suite:
`STOPLORE_TEST_POSTGRES_DSN=... GOCACHE=/tmp/stoplore-ticket11-cache go test ./... -count=1`.
