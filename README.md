# StopLore — The stories behind your stops.

A Go library for recording and analyzing a personal trading journal.

## Design documentation

The first-release journal supports executions, versioned setups, plans, corrections,
recorded costs, risk comparisons, and performance reports on CSV and PostgreSQL.

- [First-release design](docs/first-release-design.md): scope, behavior, reporting,
  migration, and acceptance examples.
- [First-release specification](.scratch/first-release-trading-journal/spec.md):
  resolved with user stories, implementation decisions, and the approved public
  Go API testing seam.
- [Domain glossary](GLOSSARY.md): the language used throughout the design.
- [Architecture decisions](docs/adr/): the reasoning behind lasting design choices.
- [Release policy and publishing instructions](docs/releasing.md).

## Exchange accounts and fees

```go
service := stoplore.NewWithCSV("./journal")
if err := service.Init(); err != nil {
    return err
}
entry, exit := stoplore.MustDecimal("0.04"), stoplore.MustDecimal("0.06")
exchange, err := service.Exchange.Add(ctx, stoplore.AddExchangeRequest{
    Name: "My exchange", EntryFeeRate: &entry, ExitFeeRate: &exit,
})
if err != nil {
    return err
}
account, err := service.CreateAccount(ctx, stoplore.CreateAccountRequest{
    ExchangeID: exchange.ID, Name: "Main",
})
```

The same API works with `NewWithPostgres(pool, key)`. See the
[decimal and storage contract](docs/decimal-and-storage.md) for exact decimal
units, missing-value behavior, rounding, and backend test setup.

## Migrate existing PostgreSQL records

Initialize the destination service with either backend, then pass a separate
`pgxpool.Pool` connected to the legacy database and schema:

```go
imported, err := service.ImportPostgres(ctx, stoplore.PostgresImportRequest{
    Source: legacyPool, SourceID: "my-old-journal", SourceKey: legacyNotesKey,
})
```

The caller owns `legacyPool`; its search path selects the source `trades` table.
The reader uses a read-only snapshot and never initializes source storage. Supply
a stable, nonsecret `SourceID` for retries and the old notes key only when source
notes are encrypted. The target's encryption configuration is independent.
Original fields, including source ciphertext, remain in migration evidence;
reviewable notes are decrypted. PostgreSQL's stored numeric and timestamp
precision is preserved.

Imported trades appear immediately through `ListTrades`, `GetTrade`, and
`ReviewMigration`. Use `ConfirmMigration` to supply the actual account, settlement
currency, executions, and actual fees before performance report eligibility.
For records without a strategy name, also select a named setup version with
`ConfirmMigrationRequest.SetupVersionID`. CSV imports support current and legacy
row layouts, including legacy rows beneath a current header.
Unknown historical rules and plans remain unknown, and funding stays optional.

Use `CorrectTradeContext` to correct the account, setup version, symbol, direction,
or settlement currency. It retains previous context and recalculates results while
preserving execution fee snapshots.
