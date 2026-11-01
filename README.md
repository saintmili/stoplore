# MTL - My Trade Logs

A Go library for recording and analyzing a personal trading journal.

## Design documentation

The first-release design is accepted; it describes the target behavior, which is
not yet fully implemented in the current library.

- [First-release design](docs/first-release-design.md): scope, behavior, reporting,
  migration, and acceptance examples.
- [First-release specification](.scratch/first-release-trading-journal/spec.md):
  published with status `ready-for-agent`, including user stories, implementation
  decisions, and the approved testing seam.
- [Domain glossary](GLOSSARY.md): the language used throughout the design.
- [Architecture decisions](docs/adr/): the reasoning behind lasting design choices.

## Exchange accounts and fees

```go
service := mtl.NewWithCSV("./journal")
if err := service.Init(); err != nil {
    return err
}
entry, exit := mtl.MustDecimal("0.04"), mtl.MustDecimal("0.06")
exchange, err := service.Exchange.Add(ctx, mtl.AddExchangeRequest{
    Name: "My exchange", EntryFeeRate: &entry, ExitFeeRate: &exit,
})
if err != nil {
    return err
}
account, err := service.CreateAccount(ctx, mtl.CreateAccountRequest{
    ExchangeID: exchange.ID, Name: "Main",
})
```

The same API works with `NewWithPostgres(pool, key)`. See the
[decimal and storage contract](docs/decimal-and-storage.md) for exact decimal
units, missing-value behavior, rounding, and backend test setup.
