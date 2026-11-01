# 07: Preserve original plans and dated revisions

**What to build:** Record an optional original entry reason, stop, target, and notes; revise decisions with dates while keeping the original plan and traceable recording corrections.

**Blocked by:** 03: Record and inspect a trade with its first entry.

**Status:** ready-for-agent

**Spec coverage:** User stories 14, 19, 20, 21, 24, 25, 60, 61.

- [ ] Capture original planning details through the public library API and retain them after later revisions.
- [ ] Represent changed trading decisions as dated plan revisions with earlier plans still readable.
- [ ] Distinguish a correction to recorded planning information from a new decision and retain previous corrected values.
- [ ] Allow historical trades with missing stop, target, or reason without fabricating them.
- [ ] Read original plans, revisions, and correction history after reopening either backend.
- [ ] Tests assert public behavior without prescribing event sourcing.

