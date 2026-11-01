# MTL first release: personal trade journaling and setup performance

Status: resolved

The product design and public Go API testing seam are approved. This
specification is published to the local Markdown issue tracker.

## Problem Statement

I trade cryptocurrency futures and hold positions for hours, days, weeks, or
months. I need a reliable journal that records what I traded, why I traded it,
how I managed it, and what happened after costs, so I can discover which setups
work for me and improve my decisions.

The existing library records a single entry and exit per position, derives
quantity from margin and leverage, and reports price profit without deducting
fees. It cannot faithfully represent scaling in, partial exits, optional funding,
changing setup rules, or the history of my plans and corrections. Its aggregate
statistics do not answer which setups performed well under the rules I actually
used.

## Solution

Build a personal trading journal as a Go library that another application can
use for its UI and service integrations. Initially support manually recorded
USDT/USDC-settled perpetual futures, with each trade representing one trading idea
and containing its actual executions.

Calculate results using weighted-average entry accounting, mandatory execution
fees supplied by exchange settings or actual-fee overrides, and optional recorded
funding. Preserve setup versions, original plans, revisions, and correction
history. Produce filtered reports of completed trades on a recorded-cost basis,
keeping settlement currencies separate and exposing incomplete records clearly.

Keep both CSV and PostgreSQL storage and provide an explicit, source-preserving
migration path. The trader remains responsible for strategy and trading
decisions.

## User Stories

1. As a Go application developer, I want journaling and reporting available through a library, so that my application can provide its own UI and service connections.
2. As a trader, I want to journal USDT/USDC-settled perpetual futures, so that MTL covers my initial trading activity.
3. As a trader, I want trades to retain execution timestamps across hours or months, so that long holding periods remain reviewable.
4. As a trader, I want every trade associated with its exchange and account, so that activity in my different accounts remains distinguishable.
5. As a trader, I want one trade to represent one idea in one instrument and direction, so that I can review the whole idea together.
6. As a trader, I want to assign executions explicitly to a trade, so that executions are grouped according to my trading ideas.
7. As a trader, I want to add multiple entries to a trade, so that scaling in is recorded accurately.
8. As a trader, I want to record partial exits, so that taking profit or reducing exposure gradually remains part of the same trade.
9. As a trader, I want to add an entry after a partial exit, so that continued management of an idea updates its remaining position correctly.
10. As a trader, I want to record each execution's actual quantity, price, and time, so that my results reflect what was filled.
11. As a trader, I want margin and leverage recorded separately from filled quantity, so that they provide context without determining my recorded exposure.
12. As a trader, I want remaining quantity and weighted-average entry accounting, so that partial realized results can be explained from my executions.
13. As a trader, I want to enter trades manually, so that I can use the journal before building exchange integrations.
14. As a trader, I want to record historical trades with missing planning details, so that an imperfect memory does not prevent me from keeping the activity.
15. As a trader, I want one primary named setup per trade, so that performance has a consistent grouping.
16. As a trader, I want contextual tags on trades, so that I can distinguish market conditions and broader strategy groupings.
17. As a trader, I want setup rules to have versions, so that changing my rules does not change the context of earlier trades.
18. As a trader, I want setup versions to remain distinguishable in reports, so that results from different rules are understandable.
19. As a trader, I want to record my original entry reason, stop, target, and notes, so that I can review the plan I started with.
20. As a trader, I want dated plan revisions that retain the original plan, so that I can examine how my decisions changed.
21. As a trader, I want unknown historical planning details represented as missing, so that my journal does not fabricate what I intended.
22. As a trader, I want to assess whether I followed rules, deviated, or have not assessed a trade, so that execution discipline is reviewable.
23. As a trader, I want notes alongside my adherence assessment, so that I can explain what happened.
24. As a trader, I want to correct inaccurate executions and journal records, so that mistakes in data entry can be fixed.
25. As a trader, I want previous values retained when I correct a record, so that corrections remain traceable.
26. As a trader, I want corrections to recalculate affected trade results and reports, so that reports use the corrected information.
27. As a trader, I want separate entry and exit percentage fee rates configured for an exchange, so that executions get the appropriate defaults.
28. As a trader, I want fees calculated automatically from exchange settings, so that I do not have to enter a fee amount on every execution.
29. As a trader, I want each fee calculated using that execution's filled quantity and price, so that partial exits and changing prices have the right execution values.
30. As a trader, I want each execution to retain the fee settings and amount used when recorded, so that later exchange-rate changes preserve historical results.
31. As a trader, I want an actual-fee override, so that discounts, rebates, and other differences can replace the default calculation.
32. As a trader, I want an explicitly configured or entered zero fee accepted, so that a known zero remains distinct from an unknown fee.
33. As a trader, I want newly recorded executions to require a fee amount, so that missing fees do not silently inflate net results.
34. As a trader, I want to record funding payments received or paid with their timestamps, so that recorded funding affects my net results.
35. As a trader, I want funding entries to be optional, so that I can report trades without confirming every possible funding payment.
36. As a trader, I want absent funding entries to contribute zero recorded funding without excluding the trade, so that reporting remains usable with optional funding.
37. As a trader, I want reports identified as using recorded costs, so that I understand that omitted funding may not be reflected.
38. As a trader, I want net results after recorded execution fees and funding, so that price profit alone does not determine reported performance.
39. As a trader, I want performance grouped by setup, so that I can identify which setups made or lost money.
40. As a trader, I want report filters for symbol, direction, and tags, so that I can examine relevant subsets of my activity.
41. As a trader, I want report filtering by rule adherence, so that I can compare following a setup with deviating from it.
42. As a trader, I want completed trades selected by final closing date, so that reports use a consistent completion period.
43. As a trader, I want each selected trade's entire lifetime result included, so that positions held across months have their full outcome represented.
44. As an application developer, I want to supply explicit report start and end instants, so that my UI can translate dates using the chosen timezone.
45. As a trader, I want open and partially closed trades shown separately with realized results visible, so that I can review ongoing activity.
46. As a trader, I want completed-trade statistics to exclude ongoing trades, so that unfinished outcomes do not affect win rate or average completed-trade profit.
47. As a trader, I want eligible trade counts displayed alongside performance, so that I can see how many trades support a result.
48. As a trader, I want total and average net profit per trade, so that I can compare the monetary results of my setups.
49. As a trader, I want wins, losses, break-even counts, and win rate based on net results, so that outcome classification reflects recorded costs.
50. As a trader, I want average win, average loss, and profit factor, so that I can compare the size of gains and losses.
51. As a trader, I want break-even trades counted separately and included in the win-rate denominator, so that the reported rate covers all eligible completed trades.
52. As a trader, I want ratios with a zero denominator explicitly unavailable, so that undefined results are not mistaken for zero performance.
53. As a trader, I want incomplete trades visible in the journal and their exclusion count visible in reports, so that I know which records still need attention.
54. As a trader, I want USDT and USDC report amounts kept separate, so that unlike settlement currencies are not silently combined.
55. As a trader, I want to record an optional original monetary risk budget for the whole trade, so that different trade sizes can be compared against planned risk.
56. As a trader, I want the original planned risk baseline preserved when plans change, so that later decisions do not rewrite that comparison.
57. As a trader, I want an R multiple calculated from net profit and original planned risk, so that I can review results relative to what I intended to risk.
58. As a trader, I want trades without an available positive risk amount included in monetary reports with R unavailable, so that missing risk data does not hide otherwise usable results.
59. As an application developer, I want decimal financial values and explicit rounding rules, so that entered prices, quantities, rates, and amounts are preserved consistently.
60. As an application developer, I want both CSV and PostgreSQL supported, so that I can choose either existing persistence option.
61. As a trader, I want trades, histories, fee settings, and results retained when the library is reopened, so that persistence preserves the journal's meaning.
62. As an application developer, I want the API and examples to express the new domain model clearly, so that I can integrate the redesigned library even when Go API changes are breaking.
63. As a trader, I want existing CSV and PostgreSQL records migrated without modifying their source records, so that I retain the original information.
64. As a trader, I want migrated records visible immediately and missing or inferred essentials identified, so that I can review what the migration could establish.
65. As a trader, I want to confirm missing or inferred essentials before migrated records enter performance reports, so that uncertain account, currency, quantity, or fee information does not silently affect results.
66. As a trader, I want absent historical rules, plans, and costs left unknown, so that migration preserves evidence rather than inventing history.
67. As a trader, I want optional funding to remain optional after migration, so that imported trades do not require an additional zero-funding confirmation.
68. As an application developer, I want meaningful errors for invalid requests and unavailable operations, so that callers can explain and handle failures.

## Implementation Decisions

- **Library boundary:** Retain a Go library for personal journaling and analysis.
  The calling application owns the UI, reporting-timezone selection, and external
  service connections. Manual entry and correction form the initial workflow.
- **Initial contracts:** Support USDT/USDC-settled perpetual futures. Preserve
  settlement currency on monetary results and keep currencies separate.
- **Trade model:** Replace the single-entry/single-exit model with one trade per
  trading idea, instrument, direction, and exchange account. Explicitly assign
  executions to trades; an execution is not itself the complete reported trade.
- **Execution inputs:** Record actual filled quantity, execution price, time, and
  fee. Store leverage and margin as separate context rather than derive filled
  quantity from them.
- **Accounting:** Maintain weighted-average entry accounting for the remaining
  position. Entries update the average, partial exits realize directional price
  profit against it, and subsequent additions average against the remaining
  quantity. Fully closing the position completes the trade.
- **Setup model:** Give each trade one primary named setup and contextual tags.
  Model setups as versioned trading rules. Broader strategy groupings use tags
  initially; do not introduce a separate strategy hierarchy.
- **Historical setup context:** Bind existing trades to the setup version under
  which they were traded and retain version identity in reports when rules
  change.
- **Plans:** Preserve the original entry reason, stop, target, and review notes.
  Record subsequent plan changes as dated revisions without replacing the
  original plan. Allow missing historical planning information explicitly.
- **Adherence:** Support followed rules, deviated, and not assessed, with notes
  and report filtering. Assessment is optional.
- **Corrections:** Distinguish correcting a record from changing a trading
  decision. Retain previous corrected values and recalculate affected trade
  results and reports. History requirements do not mandate event sourcing.
- **Exchange configuration:** Implement usable exchange configuration with
  separate entry and exit percentage fee rates. Existing maker/taker terminology
  does not define the agreed entry/exit default model.
- **Fee calculation:** Calculate each default fee from that execution's quantity
  multiplied by its price and the applicable configured rate. Calculate exit fees
  from the exit's own execution value.
- **Fee requirement:** Every new execution must have a fee amount, normally
  supplied automatically by exchange settings. Accept an explicit zero and keep
  unknown fees distinct from known zero amounts.
- **Fee history and overrides:** Preserve the settings and calculated fee used
  when an execution was recorded. Later fee-rate edits apply to future
  recordings. Allow an actual-fee override to replace the default charge,
  including discounts and rebates.
- **Funding:** Allow timestamped funding payments received or paid. Funding
  entries are optional; absent entries mean zero recorded funding and do not
  require zero-funding confirmation or cause report exclusion.
- **Net results:** Adjust directional price profit/loss for recorded execution
  fees, funding received, and funding paid. Reports identify this as a
  recorded-cost basis; an omitted payment is not evidence that none occurred.
- **Report cohort:** Select fully closed trades by final closing time and
  include their entire lifetime results and recorded costs. This is a
  completed-trade cohort rather than a period cash-movement statement.
- **Time boundary:** Accept explicit start and end instants from the caller and
  preserve precise execution and journal timestamps. The caller converts its
  reporting-timezone choices to those instants.
- **Report organization:** Group by setup while preserving setup-version
  information. Filter by closing time, symbol, direction, tags, and adherence.
- **Report eligibility:** Exclude incomplete required-fee records from
  net-performance statistics while keeping them visible and exposing the
  exclusion count. Apply confirmation requirements to missing or inferred
  migration essentials. Missing plans or funding alone do not cause exclusion.
- **Ongoing activity:** Show open and partially closed trades and their recorded
  realized results separately from completed-trade performance statistics.
- **Metrics:** Expose eligible and excluded counts; win, loss, and break-even
  counts; total and average net profit; win rate; average win and average loss;
  and profit factor. Classify outcomes by net results after recorded costs.
- **Metric edge cases:** Include break-even trades in the win-rate denominator.
  Compute profit factor as total positive net results divided by the magnitude
  of total negative net results. Represent any ratio with a zero denominator as
  explicitly unavailable, including profit factor when there are no losses.
- **Planned risk:** Optionally record the original monetary risk budget for the
  whole trade and keep it fixed through plan changes. R is net profit/loss
  divided by that baseline. Without an available positive baseline, return R as
  unavailable while retaining monetary-report eligibility.
- **Financial representation:** Use decimal representations and arithmetic with
  explicit rounding rules for prices, quantities, rates, fees, funding, and
  monetary results. Preserve entered decimals without binary floating-point
  artifacts.
- **Module interfaces:** Update the public journaling, exchange configuration,
  setup, history, reporting, and persistence contracts to express the new model.
  Breaking Go API changes are accepted; update usage examples accordingly.
- **Persistence:** Keep CSV and PostgreSQL support. Both must preserve the new
  trade and execution records, account and currency context, setup versions,
  plans, corrections, funding, and historical fee information.
- **Migration:** Provide explicit conversion from existing CSV and PostgreSQL
  records while preserving source records and their original information. Make
  imported records immediately visible, identify missing or inferred essentials,
  and require confirmation before including those records in performance
  statistics. Preserve unknown historical rules and plans as unknown.
- **Implementation scope:** Use small, explicit domain modules, meaningful
  errors, and no global mutable state. No framework or code generation has been
  requested.

## Testing Decisions

The approved primary testing seam is the public Go library API, with one shared
behavior suite against CSV and PostgreSQL.

- **One primary seam:** Exercise caller-visible behavior through the public Go
  library API, starting from its existing service constructors and journal
  operations. Include exchange configuration, setup and plan history, reporting,
  and migration through the same public library boundary as those capabilities
  are added. Do not add separate private testing interfaces for each calculation
  or persistence helper.
- **Backend coverage:** Run a shared behavior suite against CSV in temporary
  directories and PostgreSQL in an isolated test database. Test persistence by
  reopening the library and reading records, histories, and reports through its
  API, rather than asserting raw CSV column ordering or SQL structure for the new
  model. Database provisioning details remain an implementation choice.
- **What makes a good test:** Drive a realistic sequence of public actions and
  assert returned values, visible histories, errors, eligibility, and report
  results. Use fixed timestamps and independently worked decimal examples. Avoid
  assertions about private calls, internal object layout, or expected values
  calculated by the production implementation itself.
- **Prior art:** Existing lifecycle tests cover creating and closing a journal
  record and missing-record errors. Existing analytics tests provide known-result
  assertions, and CSV round-trip and legacy-record tests establish persistence
  and historical-data scenarios. Preserve their useful behavior coverage at the
  public boundary; PostgreSQL behavior coverage must be added.
- **Execution accounting:** Cover long and short trades, multiple entries,
  partial exits, adding again after an exit, and final closure. Buying one unit
  at 100 and one at 120 gives an average of 110; exiting one at 130 realizes 20
  before costs; adding one at 90 then leaves an average of 100. Verify quantities
  and directional results through trade reads and reports.
- **Fees:** Verify distinct entry/exit defaults and per-execution values. A
  hypothetical 0.04% rate gives 2.40 USDT on 6,000 USDT and 2.48 USDT on 6,200
  USDT. Cover explicit zero, missing required fees, changing defaults without
  changing history, and actual-fee overrides including rebates.
- **Funding and net results:** Cover funding received, funding paid, and no
  funding entries. For 200 USDT price profit and 4.88 USDT fees, receiving 1 USDT
  and paying 5 USDT funding yields 191.12 USDT net; no recorded funding yields
  195.12 USDT. The latter remains eligible and carries the recorded-cost basis.
- **Historical context:** Change setup rules, revise a stop, and correct an
  execution. Verify the original setup version and plan remain visible, dated
  revisions and previous corrected values are retained, and corrected results
  appear in subsequent reports. Reopen each backend and verify the same history.
- **Filtering and cohorts:** Verify each agreed filter and combined filters,
  preserve setup-version identity, and keep currencies separate. A trade opened
  in January and fully closed in March contributes its full lifetime result to
  the March cohort, while an ongoing partially closed trade is shown separately.
  Exercise explicit report instants corresponding to caller-chosen local dates.
- **Metrics and eligibility:** Cover winners, losers, break-even trades, empty
  cohorts, groups without losses, and incomplete records. Verify the win-rate
  denominator, averages, profit factor, exclusion counts, and explicit
  unavailable values for zero-denominator ratios. Missing planning or funding
  data must not independently exclude a trade.
- **Risk comparison:** Verify 100 USDT net profit against 50 USDT original risk
  produces 2R, that plan changes preserve the baseline, and that absent or
  nonpositive risk leaves R unavailable without hiding monetary performance.
- **Migration:** Use representative current and legacy CSV fixtures and existing
  PostgreSQL records through a public migration entry point at the library
  boundary. Verify source preservation, immediate visibility, missing/inferred
  information, confirmation-driven eligibility, and the optional-funding rule.
  Avoid using personal journal files as test fixtures.
- **Precision and failure behavior:** Verify exact decimal input round trips and
  observable results under the documented rounding policy once selected. Check
  meaningful caller-visible errors for invalid requests and missing records.
  Verify storage through public behavior rather than adding a second set of
  implementation-mirroring financial tests.

## Out of Scope

- A UI, terminal application, HTTP server, or hosted product.
- Automatic exchange synchronization, exchange-file imports beyond migration of
  existing MTL records, and external service integrations.
- Automated trading, automatic strategy discovery or suggestions, and backtesting.
- Live market data and unrealized position valuation.
- Account-equity analysis, portfolio returns, and equity drawdown reporting.
- Coin-settled inverse contracts, dated futures, and other markets beyond the
  agreed initial USDT/USDC-settled perpetuals.
- Automatic conversion or aggregation across settlement currencies.
- A multi-user service or separate strategy hierarchy.
- Mandatory funding collection or mandatory zero-funding confirmation.
- Guaranteed preservation of the old single-entry/single-exit Go API.
- A prescribed event-sourcing architecture, framework, or code generator.

## Further Notes

- The product scope was confirmed on 2026-11-01 after the grilling session. The
  final fee clarification supersedes the earlier suggestion that every caller
  must manually enter an actual fee: exchange entry/exit rates supply mandatory
  execution fees automatically, with actual-fee overrides available.
- Use the domain glossary and respect the accepted decisions on trades containing
  executions, preserved trading context and corrections, and recorded costs with
  optional funding. This specification synthesizes the accepted design; it does
  not claim the new behavior is already implemented.
- The current library's single-entry/single-exit records, gross-only analytics,
  floating-point financial values, and incomplete exchange service require
  redesign. Both existing persistence choices remain part of the deliverable.
- Decimal dependency, public decimal types, exact precision and rounding
  parameters, final API signatures, and persistence schemas remain implementation
  choices. Their observable contracts must implement the decisions above; no
  unselected package or storage architecture is an accepted requirement.
- Published to the configured local Markdown issue tracker with
  Status: ready-for-agent after approval of the tracker configuration, default
  triage vocabulary, and public Go API testing seam. The specification is ready
  for agent implementation.


## Answer

Implemented all thirteen tickets through the approved public Go API seam on CSV
and PostgreSQL. Trades preserve explicit executions, weighted-average positions,
versioned setups, original plans and revisions, review context, recorded fees and
funding, original risk, and dated correction histories. Reports group completed
trades by setup version and settlement currency and expose excluded imports.
Source-preserving CSV and PostgreSQL migration requires confirmation of missing
or inferred essentials, including selection of a named setup when absent.

Final review unified execution accounting with correction replay, added trade
context corrections without repricing historical fees, rejected incomplete stored
journal documents, and covered legacy CSV rows beneath current headers and mixed
row layouts. The complete test suite passed with both CSV and isolated PostgreSQL
backends; `go vet ./...` passed. See [the implementation map](map.md).
