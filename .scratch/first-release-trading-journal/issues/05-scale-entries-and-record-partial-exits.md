# 05: Scale entries and record partial exits

**What to build:** Add multiple entries, exit gradually, add again, and inspect remaining quantity, weighted-average entry, and realized results before final closure.

**Blocked by:** 04: Fully close a trade and calculate its net result.

**Status:** resolved

**Spec coverage:** User stories 7, 8, 9, 12, 45, 59, 60, 61.

- [x] Explicitly assign multiple entries and exits to the same trading idea.
- [x] Entries average against the remaining position; partial exits reduce quantity and realize directional profit against that average.
- [x] Entries of one unit at 100 and one at 120 give an average of 110; exiting one at 130 realizes 20 before costs; adding one at 90 then leaves an average of 100.
- [x] Open and partially closed trades expose recorded realized results separately from completed state, without live valuation.
- [x] Verify long and short accounting through the public API, including the final complete exit.
- [x] Both backends retain the execution sequence, fees, remaining quantity, and results across reopening.


## Answer

Added `RecordEntry` and `RecordExit` for explicitly assigned fills, with
weighted-average accounting against remaining exposure and directional realized
results after partial exits. Full exits complete the trade; `CloseTrade` keeps
its strict full-quantity contract. Entries and exits reject completed trades,
chronologically invalid fills, unavailable fees, and excessive exits without
changing stored state. Existing planning and review context remain intact.

The shared public API suite verifies long and short scaling, the 100/120 entry,
130 partial-exit and 90 addition example, final closure, fee snapshots, reopening,
invalid sequences, and 18-place average rounding on CSV and PostgreSQL.
