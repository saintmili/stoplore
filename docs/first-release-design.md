# MTL first-release design

Status: accepted design, confirmed on 2026-11-01. Implementation is pending.

This document records the decisions confirmed in the grilling session, including
the final clarification that exchange settings supply execution fees. It defines
the target behavior; it does not describe features as already implemented.

See the [domain glossary](../GLOSSARY.md) for terminology and the
[decision records](adr/) for the reasoning behind lasting choices.

## Purpose and boundary

MTL is a personal trading journal implemented as a Go library. It should help the
trader record activity, examine which setups made or lost money after recorded
costs, review whether they followed their rules, and use that evidence to develop
their trading strategy.

The calling application supplies the UI and connects to other services. The
trader makes strategy and trading decisions. Manual entry and correction are
sufficient for the first release.

The initial instruments are USDT/USDC-settled perpetual futures. Positions may
last hours, days, weeks, or months. Keep USDT and USDC amounts separate in reports;
no currency conversion policy has been agreed.

The first deliverable is the complete recording-and-review workflow described
here. Backtesting, automated strategy suggestions, live position valuation, and
account-equity analysis are later work. A UI and automatic exchange imports or
connections are outside this library's first-release deliverable; migration of
existing MTL records is included.

## Trades and executions

A trade represents one trading idea in one instrument and direction, associated
with an exchange account. The trader explicitly assigns executions to the trade.
A trade can contain multiple entries and partial exits.

Each execution records its actual filled quantity, price, time, and fee. Record
leverage and margin separately; use the executions themselves to determine
quantity and price profit/loss.

Use weighted-average entry accounting for the remaining position:

- Entries update the remaining quantity and its weighted-average entry price.
- Partial exits realize price profit/loss against that average, adjusted for
  direction, and reduce the remaining quantity.
- Adding another entry recalculates the average using the remaining position.
- A fully closed position completes the trade.

For example, buying one unit at 100 and one at 120 produces an average entry of
110. Selling one at 130 realizes 20 before costs and leaves one unit with an
average entry of 110. Adding one unit at 90 then produces a remaining average of
100. These are illustrative amounts in one settlement currency.

See [ADR 0001](adr/0001-model-trades-as-trading-ideas-with-executions.md).

## Setups, plans, and review

Each trade has one primary named setup, with additional tags for context. A setup
is the versioned set of trading rules being evaluated. Use tags for broader
strategy groupings initially, without a separate strategy hierarchy.

Changing a setup's rules creates a new version. Existing trades retain the rules
under which they were traded. Preserve setup-version identity in reporting so
results from different rules remain distinguishable.

Capture the entry reason, planned stop, planned target, and review notes. Retain
the original plan and record later changes as dated revisions. Do not replace the
original plan with the latest plan.

Support historical entries even when the trader no longer knows their original
stop, target, or reason. Represent absent planning details explicitly. Their
absence alone does not exclude a trade from monetary performance reports.

Rule adherence is an optional assessment with three values: followed rules,
deviated, or not assessed. Allow explanatory notes and report filtering by the
assessment.

A correction fixes an inaccurate journal record; a plan revision represents a
changed trading decision. Retain previous values for corrections as well as plan
history. Correcting an execution recalculates the affected trade and reports.
This history requirement does not select an event-sourcing architecture.

See [ADR 0002](adr/0002-preserve-trading-context-and-correction-history.md).

## Fees and funding

### Exchange defaults and execution fees

Configure separate entry and exit percentage fee rates for each exchange. Each
execution gets its default fee from the corresponding exchange setting:

```text
execution value = filled quantity × execution price
default execution fee = execution value × configured fee rate
```

Use each exit's own quantity and price when calculating its fee. For example, a
hypothetical 0.04% fee on an execution worth 6,000 USDT is 2.40 USDT; the same rate
on an execution worth 6,200 USDT is 2.48 USDT.

Every newly recorded execution must have a fee amount. An explicitly configured
or entered zero is valid; an unknown fee must not silently become zero. The caller
does not need to enter an actual amount on every execution when exchange settings
can supply the calculated amount.

Each execution preserves the fee settings and calculated amount used when it was
recorded. Later exchange-rate edits apply to future recordings and preserve
historical results. An explicit actual-fee override replaces the calculated fee
when the real charge differs, including discounts or rebates.

These defaults are entry/exit rates, as agreed. The existing code's maker/taker
fields do not establish the new fee model.

### Optional funding

Record funding amounts received or paid with their timestamps. Funding entries
are optional. No funding entries means zero recorded funding, and does not make a
trade ineligible for performance reports or require a separate confirmation that
funding was zero.

Report results on a recorded-cost basis. The absence of funding entries does not
establish that no funding payment occurred.

For a completed trade:

```text
net profit/loss = price profit/loss
                  − recorded execution fees
                  + recorded funding received
                  − recorded funding paid
```

For example, 200 USDT price profit, 4.88 USDT total execution fees, 5 USDT funding
paid, and 1 USDT funding received produce 191.12 USDT net profit. With no funding
recorded, the same trade reports 195.12 USDT on a recorded-cost basis.

See [ADR 0003](adr/0003-report-recorded-costs-with-optional-funding.md).

## Reports

### Selection and eligibility

Group performance by setup, preserving the setup-version information. Support
filters for final closing date, symbol, direction, tags, and rule adherence.

Select completed trades by their final closing time and include their entire
lifetime result. A trade opened in January and fully closed in March contributes
its complete result, including its recorded costs, to March's completed-trade
report. This is a completed-trade cohort report, not a statement of cash movements
during March.

The calling application chooses its reporting timezone and supplies explicit
start/end instants. Preserve timestamps as precise instants; the UI can translate
a choice such as "March in Tehran" into the requested boundaries.

Use fully closed trades for setup win rate and average-profit statistics. Show
open and partially closed trades separately, with their recorded realized results
visible. Live unrealized valuation is outside this release.

A trade with missing required fees is incomplete and excluded from net-performance
statistics. Show excluded records in the journal and expose their count in
reports. Migrated records also require confirmation of missing or inferred
essentials before inclusion. Optional funding and missing planning details do not
cause exclusion.

### Metrics

For each settlement currency, expose:

- Eligible trade count and the excluded count.
- Win, loss, and break-even counts.
- Total net profit/loss and average net profit/loss per eligible trade.
- Win rate: winning trades divided by all eligible completed trades, including
  break-even trades.
- Average win and average loss for their respective groups.
- Profit factor: total positive net results divided by the magnitude of total
  negative net results.

Classify wins, losses, and break-even trades using net results after recorded
costs. Always show trade counts alongside performance. Ratios with a zero
denominator are explicitly unavailable; for example, profit factor is unavailable
when there are no losses. Do not represent an unavailable ratio as zero.

### Original planned risk and R

Optionally record the monetary amount originally planned to be risked on the
whole trade. Keep this baseline fixed when the plan later changes.

```text
R multiple = completed trade's net profit/loss ÷ original planned risk
```

For example, 100 USDT net profit against 50 USDT original planned risk is 2R.
Trades without an available positive risk amount remain eligible for monetary
reports, with R explicitly unavailable.

## Precision, storage, and compatibility

Use decimal representations and arithmetic for prices, quantities, rates, fees,
funding, and monetary calculations, with explicit rounding rules. Preserve the
decimal values entered rather than introducing binary floating-point artifacts.
The decimal dependency, public value types, precision, and specific rounding
parameters have not been selected by this design.

Support both CSV and PostgreSQL for the new model. Breaking Go API changes are
accepted so trades, executions, setups, and histories can be expressed clearly.
Update usage examples with the redesigned API.

Provide an explicit migration path for existing CSV and PostgreSQL records,
preserving source records and their original information. Imported records are
immediately visible in the journal. Identify missing or inferred information and
require its confirmation before performance reporting; examples include account,
settlement currency, quantities, and fees.

Do not fabricate absent historical setup rules, plans, or costs. The funding rule
still applies to migrated records: no funding entries means zero recorded
funding, without making funding confirmation mandatory.

## Current implementation versus accepted design

The current library provides a starting point, but differs from this design:

| Area | Current implementation | Accepted target |
| --- | --- | --- |
| Trading activity | One entry and one exit in [`Position`](../models.go) | One trade with multiple executions |
| Quantity and price profit | Quantity derived from margin and leverage in [`position.go`](../position.go) | Explicit filled quantity; weighted-average accounting |
| Costs | Fee helpers separate from price profit; no funding in [`position.go`](../position.go) | Mandatory fee amounts from snapshotted defaults or overrides; optional recorded funding included in net results |
| Setups and history | Strategy string and mutable position fields in [`models.go`](../models.go) | Versioned setup rules, original plans, revisions, and correction history |
| Exchange settings | Maker/taker fields and stub service methods in [`exchange.go`](../exchange.go) | Working exchange configuration with entry/exit fee defaults |
| Reporting | Aggregate price-profit statistics in [`journal.go`](../journal.go) | Filtered setup reports using completed, eligible trades and recorded costs |
| Numeric values | `float64` fields in [`models.go`](../models.go) | Decimal values and explicit rounding |
| Persistence | Position records in [`csv_position.go`](../csv_position.go) and [`postgres.go`](../postgres.go) | Both backends support the new model, with explicit migration |

## Acceptance examples

The following summarize agreed behavior; they are not executable tests:

1. Scale in, partially exit, add again, and fully close a trade. Its quantities,
   remaining average entry, realized price profit, and final costs remain
   traceable to its executions.
2. Edit an exchange's fee rates. Existing execution fees stay unchanged; newly
   recorded executions use the new defaults, and explicit actual-fee overrides
   remain supported.
3. Compare a trade with recorded funding and one without funding entries. Both
   remain eligible when otherwise complete; each result reflects recorded costs.
4. Change setup rules and revise a stop. Earlier trades retain their setup
   versions, and the original plan remains visible alongside dated revisions.
5. Correct a mistyped execution. Previous values remain traceable and affected
   trade results and reports are recalculated.
6. Run a March report for a trade opened in January and fully closed in March.
   Include its lifetime result once in that completed-trade cohort. Keep a trade
   still partially open outside the completed-trade statistics.
7. Report a break-even trade, a group without losses, and a trade without planned
   risk. Preserve the break-even denominator rule and explicit unavailability of
   undefined profit factor or R.
8. Migrate legacy records while preserving their source. Show imported records
   immediately, exclude those awaiting confirmation of essential information,
   and show the exclusion count.
