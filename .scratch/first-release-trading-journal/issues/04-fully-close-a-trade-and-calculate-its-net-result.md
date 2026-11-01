# 04: Fully close a trade and calculate its net result

**What to build:** Record a complete exit, calculate directional price profit and both execution fees, and expose the completed trade's decimal net result.

**Blocked by:** 03: Record and inspect a trade with its first entry.

**Status:** ready-for-agent

**Spec coverage:** User stories 28, 29, 30, 32, 33, 38, 59, 60, 61, 62.

- [ ] A complete exit records actual filled quantity, price, timestamp, and a mandatory exit fee.
- [ ] Calculate the exit default from its own quantity and price using the configured exit rate and preserve the fee snapshot.
- [ ] Fully closed trades expose directional price profit, total recorded execution fees, and net result for long and short examples.
- [ ] With no funding entries, net result uses zero recorded funding without claiming that no payments occurred.
- [ ] Both backends retain the completed result across reopening, verified through the shared public API suite.
- [ ] Update usage examples for the new complete-trade workflow.

