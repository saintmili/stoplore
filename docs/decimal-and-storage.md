# Financial values and journal storage

New journal APIs use `mtl.Decimal`, an alias of `shopspring/decimal.Decimal`.
Construct values from strings with `mtl.ParseDecimal` (returns an error) or
`mtl.MustDecimal` (panics on invalid input). Plain signed base-ten integer or
fractional digits are accepted, with no exponent, whitespace, NaN, or infinity.
Financial inputs must not pass through binary floats. Decimal zero values are
valid; optional fields use `*Decimal`, with `nil` meaning unknown and a pointer
to zero meaning a known zero.

Exchange `EntryFeeRate` and `ExitFeeRate` use percentage units: `0.04` means
0.04 percent. For example, a 6,000 USDT execution at this rate costs 2.40 USDT.
These defaults must be nonnegative; actual execution overrides may represent
rebates independently. `Exchange.Edit` replaces both defaults and the name;
a nil rate clears that default. Maker/taker fields are retained for legacy API
compatibility and never substitute for entry/exit defaults.

Entered finite decimals retain their exact numerical value in storage. Canonical
string formatting removes insignificant trailing zeroes (`0.0400` reads as
`0.04`). Addition and multiplication remain exact. Journal calculations that
require division use `DivRound(divisor, 18)`, rounding to 18 fractional places
with ties away from zero. No currency-specific cent rounding is implicit.
Callers may format returned values for display without modifying journal data.

Both constructors return a service that must be initialized with `Init` before
use. `NewWithCSV` takes a directory and stores new journal documents in
`mtl_journal.csv`. The file is a versioned CSV record containing a JSON document,
so optional values and future histories retain their meaning. Writes use a
filesystem lock and atomic replacement; failed domain operations leave the
previous document intact. PostgreSQL stores the equivalent JSONB document in
`mtl_journal_state`, with a row lock around each change. Mutations read current
storage rather than trusting a cached snapshot, allowing reopened service
instances to see each other's updates.

Existing `positions.csv`, `exchanges.csv`, and PostgreSQL `trades` records remain
in their legacy schema. New configuration writes do not alter those records.
Migration is a separate, explicit operation; no maker/taker rate is inferred as
an entry/exit default.

The shared public API behavior suite runs CSV by default. Set
`MTL_TEST_POSTGRES_DSN` to a PostgreSQL connection string to run the same suite
against PostgreSQL. Each test creates and drops its own random schema, so the
configured database user needs schema creation privileges. Tests reopen the
library and verify records through public operations.

`Service.RecordTrade` records a single trading idea with a mandatory first filled
entry. `AccountID` identifies the exchange account, and `SetupVersionID` identifies
the immutable rules version, rather than the setup's latest rules. The caller
supplies `LONG` or `SHORT`, an instrument symbol, and `USDT` or `USDC` settlement.
`GetTrade` and `ListTrades` expose the recorded executions and remaining position.
Filled quantity and price must be positive, and the execution timestamp must be
present. Optional margin must be nonnegative and optional leverage positive;
neither determines quantity. Execution instants retain nanosecond precision.

The first entry snapshots its exchange's entry percentage in `Execution.Fee.Rate`
and its calculated charge in `Execution.Fee.Amount`. Its default charge is
`quantity * price * rate / 100`, rounded to 18 fractional places using the division
contract above. A configured zero rate produces a known zero fee. An absent rate
rejects the request; changing exchange settings subsequently leaves the saved
execution's rate and amount intact. Inputs and averages for a single entry retain
the entered decimal value without rounding. `ListTrades` orders by journal
creation time, with UUID as a deterministic tie breaker.

`Service.CloseTrade` records a complete exit with its actual filled quantity,
price, and timestamp. Quantity must equal the remaining position and the timestamp
must not precede earlier executions. An already completed trade rejects further
closure. The exit snapshots the exchange's current exit percentage and calculates
its fee from its own quantity and price, including a configured known zero.
Missing exit defaults reject closure without changing the trade.

Trades expose `RealizedPriceProfitLoss`, `TotalExecutionFees`, `RecordedFunding`,
and `NetProfitLoss` in their settlement currency. Directional price profit is
exit minus average entry for long trades and average entry minus exit for shorts,
multiplied by the exited quantity. Net result subtracts all recorded execution
fees and adds signed recorded funding. With no funding entries, recorded funding
is zero; `CostBasis` is `RecordedCosts`, which does not assert that no unrecorded
payments occurred. `CompletedAt` is the final exit instant and remaining quantity
is zero. Before closure, the first entry's fee is already included in the trade's
recorded costs and net result.

`Service.RecordEntry` adds a filled entry to an ongoing trade. Its weighted
average is `(remaining quantity * average entry + added quantity * entry price)
/ (remaining quantity + added quantity)`, rounded to 18 fractional places under
the division rule above. Subsequent additions use this recorded average. Entries
average against only remaining exposure, rather than quantities already exited.

`Service.RecordExit` records either a partial or a complete exit. An exit cannot
exceed remaining quantity. Partial exits retain the average entry price and
realize directional price profit against that average for the exited quantity.
Each entry or exit snapshots the current applicable exchange fee; all recorded
execution fees contribute to net result immediately, including entry fees for
remaining exposure. No unrealized market valuation is included. Executions must
be recorded in nondecreasing timestamp order; equal instants retain recording
order. A complete exit sets `CompletedAt`; completed trades reject additional
entries and exits. The last average entry price remains available after closure.
`CloseTrade` continues to require an explicitly supplied complete exit quantity.
