package main

import (
	"context"
	"fmt"
	"time"

	"github.com/saintmili/stoplore"
)

func main() {
	service := stoplore.NewWithCSV("data")
	must(service.Init())
	ctx := context.Background()
	rate := stoplore.MustDecimal("0.04")
	exchange, err := service.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: fmt.Sprintf("Example %d", time.Now().UnixNano()), EntryFeeRate: &rate, ExitFeeRate: &rate})
	must(err)
	account, err := service.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: exchange.ID, Name: "Main"})
	must(err)
	setup, err := service.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Breakout", Rules: "Enter after resistance breaks"})
	must(err)
	margin, leverage := stoplore.MustDecimal("100"), stoplore.MustDecimal("2")
	trade, err := service.RecordTrade(ctx, stoplore.RecordTradeRequest{
		AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT",
		FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("60000"), ExecutedAt: time.Now(), Margin: &margin, Leverage: &leverage},
	})
	must(err)
	closed, err := service.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: trade.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("62000"), ExecutedAt: time.Now()}})
	must(err)
	fmt.Println("Completed:", closed.CompletedAt, "Net result:", closed.NetProfitLoss, closed.SettlementCurrency, "Basis:", closed.CostBasis)
	saved, err := service.GetTrade(ctx, trade.ID)
	must(err)
	fmt.Println("Trade:", saved.ID, "Quantity:", saved.RemainingQuantity, "Entry price:", saved.AverageEntryPrice)
	fmt.Println("Entry fee:", saved.Executions[0].Fee.Amount, saved.SettlementCurrency)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
