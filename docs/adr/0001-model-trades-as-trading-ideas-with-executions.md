---
status: accepted
---

# Model trades as trading ideas containing executions

The existing single-entry/single-exit position cannot represent scaling into a
trading idea and closing it gradually. Model a trade as one idea in one instrument
and direction on an exchange account, with explicitly assigned executions and
weighted-average accounting for its remaining position. This changes the public
API and both storage models, but keeps the complete idea reviewable without
splitting each fill into a separate reported trade.

This records the accepted target design; implementation is pending. See the
[first-release design](../first-release-design.md#trades-and-executions).
