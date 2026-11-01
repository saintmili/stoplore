# 10: Report completed-trade performance by setup

**What to build:** Produce filtered setup reports from corrected completed trades, with lifetime net results, agreed metrics, explicit unavailable ratios, and separate currencies and ongoing activity.

**Blocked by:** 08: Tag trades and review rule adherence; 09: Correct executions and cost records with history.

**Status:** resolved

**Spec coverage:** User stories 18, 26, 37, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 59, 60, 61.

- [x] Group by setup while preserving setup-version identity and filter by final closing instants, symbol, direction, tags, and adherence.
- [x] Select fully closed trades by final closing time and include lifetime results, including costs recorded outside the selected period.
- [x] Expose eligible and excluded counts, win/loss/break-even counts, total and average net profit, win rate, average win/loss, and profit factor.
- [x] Classify outcomes by net results and include break-even trades in the win-rate denominator; zero-denominator ratios are explicitly unavailable.
- [x] Compute profit factor from total positive net results divided by the magnitude of total negative net results; no losses makes it unavailable.
- [x] Apply required-fee completeness and migration-confirmation eligibility rules without excluding trades solely for missing plans or funding.
- [x] Keep USDT and USDC results separate and show ongoing trade realized results outside completed-trade statistics.
- [x] Reports reflect corrected executions and costs, including corrected final-close times changing cohort membership.
- [x] Run the same externally asserted metric and filtering cases against both backends, including empty cohorts and caller-supplied instants for local reporting dates.
- [x] Migration slices add incomplete-import scenarios through the public migration workflow.



## Answer

Added `Service.ReportPerformance` with half-open final-closing-time cohorts,
lifetime recorded costs, setup-version/currency groups, context filters, explicit
nil ratios, eligible/excluded counts, and ongoing activity outside the statistics.
Average loss remains signed; win rate is a fraction and includes break-even trades.
Known fees require an actual/default amount pointer, so imported unknown fees do
not become known zero. `Trade.PendingEssentials` reserves the confirmation gate
for ticket 12; unknown setup/currency groups remain visible and excluded.

Shared public API cases pass against CSV and isolated PostgreSQL, including reopen,
the 191.12 recorded-cost example, monetary and ratio metrics, net classification,
caller-chosen UTC+03:30 date boundaries, every filter and combined filters, empty
cohorts, separate currencies/versions, partial ongoing realized results, and
corrected final-close time and fees. Public incomplete-import and confirmation
scenarios will be added by ticket 12 through its migration workflow, as specified.
