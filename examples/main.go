package main

import (
	"context"
	"fmt"
	"time"

	"github.com/saintmili/mtl"
)

func main() {
	service := mtl.NewWithCSV("data")
	err := service.Init()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	_, err = service.Journal.OpenPosition(
		context.Background(),
		&mtl.OpenPositionRequest{
			Symbol:         "BTCUSDT",
			Exchange:       "BITUNIX",
			PositionMargin: 12,
			OrderType:      "LIMIT",
			Side:           "BUY",
			EntryType:      "TRADE",
			TimeFrame:      "1H",
			Leverage:       2,
			EntryPrice:     1.2,
			Notes:          "note",
			OpenedAt:       time.Now(),
		},
	)
	if err != nil {
		panic(err)
	}

	// closeReq := &mtl.ClosePositionRequest{
	// 	ID:       uuid.MustParse("c7126881-e9a9-40fb-bd76-5cce80a6673f"),
	// 	ExitedAt: time.Now(),
	// 	ExitMarginPercent: 100,
	// 	ExitPrice: 1.5,
	// 	ExitFeeRate: 0.003,
	// 	ExitMentalState: "happy",
	// 	ExitImageUrl: "",
	// 	Notes: "testing",
	// }
	// _, err = service.Journal.ClosePosition(ctx, closeReq)
	// if err != nil {
	// 	panic(err)
	// }

	stats, err := service.Journal.GetAnalytics(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("Total Positions:", stats.TotalPositions)
	fmt.Println("Win Rate:", stats.WinRate)
	fmt.Println("Profit Factor:", stats.ProfitFactor)
	fmt.Println("Expectancy:", stats.Expectancy)
	fmt.Println("Total Pnl:", stats.TotalPnL)
}
