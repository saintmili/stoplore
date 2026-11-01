# MTL Trading Journal

MTL records a trader's activity and decisions so they can review their execution
and compare the results of their trading setups.

## Language

### Trading activity

**Trade**:
One trading idea in one instrument and direction, associated with an exchange
account and containing its entries and exits.
_Avoid_: Order, execution, position when referring to the complete trading idea.

**Execution**:
A filled purchase or sale with a quantity, price, time, and fee that adds to or
reduces a trade's exposure.
_Avoid_: Order when referring to a completed fill.

**Entry**:
An execution that adds exposure to a trade.

**Exit**:
An execution that reduces exposure to a trade, either partially or completely.

**Position**:
The remaining market exposure of a trade after its entries and exits.
_Avoid_: Trade when referring only to remaining exposure.

**Trade direction**:
Whether the trading idea is long or short.

**Exchange account**:
An account belonging to the trader at a particular exchange.
_Avoid_: Exchange when identifying a particular account rather than the venue.

**Settlement currency**:
The currency in which a contract's profit and loss are settled.

**Completed trade**:
A trade whose position has been fully closed.
_Avoid_: Completed trade for a trade with only a partial exit.

### Trading decisions

**Setup**:
A named set of trading rules being evaluated through journaled trades.
_Avoid_: Strategy as a separate hierarchy of setups in the initial scope.

**Setup version**:
The particular rules of a setup used for a trade, preserved when the setup's rules
subsequently change.

**Tag**:
A contextual label attached to a trade, such as a broader strategy grouping or
market condition.

**Trade plan**:
The trader's intended entry reason, stop, and target for a trade.

**Plan revision**:
A dated change to a trade plan that preserves the original plan and earlier
revisions.
_Avoid_: Correction when the trader changed their decision rather than fixed a record.

**Correction**:
A change that fixes an inaccurate journal record rather than represents a new
trading decision.

**Rule adherence**:
The trader's assessment that a trade followed its rules, deviated from them, or
has not been assessed, accompanied by explanatory notes where needed.

### Costs and performance

**Execution fee**:
The charge or rebate associated with an execution, calculated from the exchange's
configured entry or exit rate unless an actual-fee override is recorded.

**Fee override**:
An explicitly recorded actual execution fee that replaces the default fee
calculated from exchange settings.

**Funding payment**:
A payment received or paid while holding a perpetual futures position.

**Realized price profit/loss**:
The result of exited quantity against its weighted-average entry price, adjusted
for trade direction, before execution fees and funding.

**Net profit/loss**:
Price profit/loss adjusted for recorded execution fees and recorded funding
payments.
_Avoid_: Gross profit/loss when referring to a result after recorded costs.

**Original planned risk**:
The optional monetary amount the trader originally planned to risk on the whole
trade, retained as a fixed comparison baseline.

**R multiple**:
A trade's net profit/loss divided by its original planned risk, when that risk is
available and greater than zero.

**Break-even trade**:
A completed trade whose net profit/loss is zero.

**Report-eligible trade**:
A completed trade with the essential information and fees required for
performance reporting and, for migrated records, confirmed missing or inferred
essentials.

**Recorded-cost basis**:
A performance calculation using recorded execution fees and funding, with absent
funding entries contributing zero recorded funding.
