# 08: Tag trades and review rule adherence

**What to build:** Attach contextual tags and review whether a trade followed its rules, deviated, or remains unassessed, with notes and traceable corrections.

**Blocked by:** 03: Record and inspect a trade with its first entry.

**Status:** resolved

**Spec coverage:** User stories 16, 22, 23, 24, 25, 60, 61.

- [x] Public operations attach and retrieve contextual tags without adding a separate strategy hierarchy.
- [x] Support followed rules, deviated, and not assessed, with optional explanatory notes.
- [x] Unassessed activity remains valid journal data.
- [x] Retain previous values when correcting recorded review or tagging information.
- [x] Both backends preserve tags, assessments, notes, and correction history across reopening, verified through the public API.


## Answer

Added `SetTradeReview` to replace a trade's contextual tags, rule adherence, and
optional notes together. `GetTrade` and `ListTrades` expose normalized,
deduplicated tags and the current assessment. The zero-value assessment is
unassessed, so a new trade remains valid without a review. Followed and deviated
assessments can be corrected or reset to unassessed.

Each successful update preserves the previous tags and review in
`ReviewHistory` with its UTC correction timestamp, including the original
unassessed context. Invalid requests leave both current values and history
unchanged. No separate strategy hierarchy was introduced.

Verified through the approved public Go API seam with three red/green slices:
context and optional assessments, correction history across reopening, and
validation without mutation. `MTL_TEST_POSTGRES_DSN=... go test ./...` passes
against CSV and isolated PostgreSQL schemas, including reopen and list reads.
