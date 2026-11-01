# 10: Report completed-trade performance by setup

**What to build:** Produce filtered setup reports from corrected completed trades, with lifetime net results, agreed metrics, explicit unavailable ratios, and separate currencies and ongoing activity.

**Blocked by:** 08: Tag trades and review rule adherence; 09: Correct executions and cost records with history.

**Status:** ready-for-agent

**Spec coverage:** User stories 18, 26, 37, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 59, 60, 61.

- [ ] Group by setup while preserving setup-version identity and filter by final closing instants, symbol, direction, tags, and adherence.
- [ ] Select fully closed trades by final closing time and include lifetime results, including costs recorded outside the selected period.
- [ ] Expose eligible and excluded counts, win/loss/break-even counts, total and average net profit, win rate, average win/loss, and profit factor.
- [ ] Classify outcomes by net results and include break-even trades in the win-rate denominator; zero-denominator ratios are explicitly unavailable.
- [ ] Compute profit factor from total positive net results divided by the magnitude of total negative net results; no losses makes it unavailable.
- [ ] Apply required-fee completeness and migration-confirmation eligibility rules without excluding trades solely for missing plans or funding.
- [ ] Keep USDT and USDC results separate and show ongoing trade realized results outside completed-trade statistics.
- [ ] Reports reflect corrected executions and costs, including corrected final-close times changing cohort membership.
- [ ] Run the same externally asserted metric and filtering cases against both backends, including empty cohorts and caller-supplied instants for local reporting dates.
- [ ] Migration slices add incomplete-import scenarios through the public migration workflow.

