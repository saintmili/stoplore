# 05: Scale entries and record partial exits

**What to build:** Add multiple entries, exit gradually, add again, and inspect remaining quantity, weighted-average entry, and realized results before final closure.

**Blocked by:** 04: Fully close a trade and calculate its net result.

**Status:** ready-for-agent

**Spec coverage:** User stories 7, 8, 9, 12, 45, 59, 60, 61.

- [ ] Explicitly assign multiple entries and exits to the same trading idea.
- [ ] Entries average against the remaining position; partial exits reduce quantity and realize directional profit against that average.
- [ ] Entries of one unit at 100 and one at 120 give an average of 110; exiting one at 130 realizes 20 before costs; adding one at 90 then leaves an average of 100.
- [ ] Open and partially closed trades expose recorded realized results separately from completed state, without live valuation.
- [ ] Verify long and short accounting through the public API, including the final complete exit.
- [ ] Both backends retain the execution sequence, fees, remaining quantity, and results across reopening.

