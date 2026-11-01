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
