# 07: Preserve original plans and dated revisions

**What to build:** Record an optional original entry reason, stop, target, and notes; revise decisions with dates while keeping the original plan and traceable recording corrections.

**Blocked by:** 03: Record and inspect a trade with its first entry.

**Status:** resolved

**Spec coverage:** User stories 14, 19, 20, 21, 24, 25, 60, 61.

- [x] Capture original planning details through the public library API and retain them after later revisions.
- [x] Represent changed trading decisions as dated plan revisions with earlier plans still readable.
- [x] Distinguish a correction to recorded planning information from a new decision and retain previous corrected values.
- [x] Allow historical trades with missing stop, target, or reason without fabricating them.
- [x] Read original plans, revisions, and correction history after reopening either backend.
- [x] Tests assert public behavior without prescribing event sourcing.


## Answer

`RecordTradeRequest.OriginalPlan` optionally captures a `TradePlan` containing
entry reason, stop, target, and notes. Nil plans and nil prices preserve unknown
historical planning details. Supplied stop and target prices must be positive.

`ReviseTradePlan` records full plan snapshots with UUIDs and explicit decision
dates, retaining the original and earlier revisions. `CorrectTradePlan` updates
the recorded original (nil revision UUID) or a selected revision, retaining the
previous and corrected values, correction date, and explanation in
`Trade.PlanCorrections`. A nil corrected original restores an unknown plan;
corrections do not add decisions or change revision dates.

Public API tests were developed in red/green slices and verify optional and
partial originals, multiple revisions, corrections to originals and revisions,
validation failures, missing records, and persistence after CSV/PostgreSQL reopen.
All `go test ./...` checks pass with the isolated PostgreSQL DSN enabled.
Original planned risk remains a separate trade-level extension for ticket 11.
