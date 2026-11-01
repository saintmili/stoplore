---
status: accepted
---

# Report recorded costs with optional funding

The first release uses manual journal entry, with mandatory execution fees
supplied by snapshotted exchange defaults or explicit actual-fee overrides, while
funding entries remain optional. Include recorded funding in net results and
treat absent funding entries as zero recorded funding without excluding the trade
or requiring confirmation of zero funding. This keeps the manual workflow usable
but means reports must identify their recorded-cost basis, since omitted funding
cannot be assumed not to have occurred.

This records the accepted target design; implementation is pending. See the
[first-release design](../first-release-design.md#fees-and-funding).
