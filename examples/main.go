package main

import (
	"context"
	"fmt"
	"time"

	"github.com/saintmili/mtl"
)

func main() {
	service := mtl.NewWithCSV("data")
	must(service.Init())
	ctx := context.Background()
	rate := mtl.MustDecimal("0.04")
	exchange, err := service.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: fmt.Sprintf("Example %d", time.Now().UnixNano()), EntryFeeRate: &rate, ExitFeeRate: &rate})
	must(err)
	account, err := service.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: exchange.ID, Name: "Main"})
	must(err)
	setup, err := service.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Breakout", Rules: "Enter after resistance breaks"})
	must(err)
	margin, leverage := mtl.MustDecimal("100"), mtl.MustDecimal("2")
	trade, err := service.RecordTrade(ctx, mtl.RecordTradeRequest{
		AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: mtl.Long, SettlementCurrency: "USDT",
		FirstEntry: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("0.1"), Price: mtl.MustDecimal("60000"), ExecutedAt: time.Now(), Margin: &margin, Leverage: &leverage},
	})
	must(err)
	closed, err := service.CloseTrade(ctx, mtl.CloseTradeRequest{TradeID: trade.ID, Exit: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("0.1"), Price: mtl.MustDecimal("62000"), ExecutedAt: time.Now()}})
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
