---
status: accepted
---

# Preserve trading context and correction history

Overwriting setup rules or trade plans would make historical results appear to
come from decisions the trader did not make at the time. Bind trades to setup
versions, preserve original plans with dated revisions, and retain previous
values when correcting inaccurate records. This adds persistence requirements
compared with storing only current values, but supports review of the original
decisions and recalculation after corrections without prescribing event sourcing.

This records the accepted target design; implementation is pending. See the
[first-release design](../first-release-design.md#setups-plans-and-review).
