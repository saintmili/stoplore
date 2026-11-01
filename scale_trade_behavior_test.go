package mtl_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"

	mtl "github.com/saintmili/mtl"
)

func TestScaleEntriesAverageAgainstRemainingPosition(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := mtl.MustDecimal("0.04")
		ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Scaling", EntryFeeRate: &rate, ExitFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Scale", Rules: "Add exposure"})
		if err != nil {
			t.Fatal(err)
		}
		for _, direction := range []mtl.TradeDirection{mtl.Long, mtl.Short} {
			pnl, partialNet, finalNet := "20", "19.86", "59.728"
			if direction == mtl.Short {
				pnl, partialNet, finalNet = "-20", "-20.14", "-60.272"
			}
			at := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC)
			fill := func(price string, hour int) mtl.ExecutionRequest {
				return mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal(price), ExecutedAt: at.Add(time.Duration(hour) * time.Hour)}
			}
			trade, err := s.RecordTrade(ctx, mtl.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: direction, SettlementCurrency: "USDT", FirstEntry: fill("100", 0)})
			if err != nil {
				t.Fatal(err)
			}
			trade, err = s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: trade.ID, Entry: fill("120", 1)})
			if err != nil {
				t.Fatal(err)
			}
			if trade.RemainingQuantity.String() != "2" || trade.AverageEntryPrice.String() != "110" || trade.TotalExecutionFees.String() != "0.088" || trade.NetProfitLoss.String() != "-0.088" || trade.CompletedAt != nil {
				t.Fatalf("scaled trade: %+v", trade)
			}
			trade, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: trade.ID, Exit: fill("130", 2)})
			if err != nil {
				t.Fatal(err)
			}
			if trade.RemainingQuantity.String() != "1" || trade.AverageEntryPrice.String() != "110" || trade.RealizedPriceProfitLoss.String() != pnl || trade.NetProfitLoss.String() != partialNet || trade.CompletedAt != nil {
				t.Fatalf("partial exit: %+v", trade)
			}
			trade, err = s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: trade.ID, Entry: fill("90", 3)})
			if err != nil {
				t.Fatal(err)
			}
			if trade.RemainingQuantity.String() != "2" || trade.AverageEntryPrice.String() != "100" || trade.RealizedPriceProfitLoss.String() != pnl || trade.TotalExecutionFees.String() != "0.176" {
				t.Fatalf("added after partial exit: %+v", trade)
			}

			s, err = b.Open()
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.GetTrade(ctx, trade.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.AverageEntryPrice.String() != "100" || got.RemainingQuantity.String() != "2" || len(got.Executions) != 4 || got.Executions[1].Kind != mtl.Entry || got.Executions[1].Fee.Amount.String() != "0.048" || got.Executions[1].Fee.Rate.String() != "0.04" {
				t.Fatalf("reopened: %+v", got)
			}
			finalFill := fill("120", 4)
			finalFill.Quantity = mtl.MustDecimal("2")
			closed, err := s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: trade.ID, Exit: finalFill})
			if err != nil {
				t.Fatal(err)
			}
			if closed.CompletedAt == nil || !closed.CompletedAt.Equal(finalFill.ExecutedAt) || !closed.RemainingQuantity.IsZero() || closed.NetProfitLoss.String() != finalNet || closed.TotalExecutionFees.String() != "0.272" {
				t.Fatalf("final exit: %+v", closed)
			}
			s, err = b.Open()
			if err != nil {
				t.Fatal(err)
			}
			got, err = s.GetTrade(ctx, trade.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.NetProfitLoss.String() != finalNet || len(got.Executions) != 5 || got.CompletedAt == nil || got.Executions[2].Kind != mtl.Exit || got.Executions[3].Kind != mtl.Entry {
				t.Fatalf("reopened completed scale: %+v", got)
			}
		}

	})
}

func TestScalingRejectsInvalidSequencesWithoutChangingTrade(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := mtl.MustDecimal("0")
		ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Sequence", EntryFeeRate: &rate, ExitFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Sequence", Rules: "No overselling"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		fill := mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: at}
		trade, err := s.RecordTrade(ctx, mtl.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "ETHUSDT", Direction: mtl.Long, SettlementCurrency: "USDT", FirstEntry: fill})
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name   string
			change func(*mtl.ExecutionRequest)
		}{
			{"over exit", func(r *mtl.ExecutionRequest) { r.Quantity = mtl.MustDecimal("2") }},
			{"zero quantity", func(r *mtl.ExecutionRequest) { r.Quantity = mtl.MustDecimal("0") }},
			{"negative price", func(r *mtl.ExecutionRequest) { r.Price = mtl.MustDecimal("-1") }},
			{"missing time", func(r *mtl.ExecutionRequest) { r.ExecutedAt = time.Time{} }},
			{"earlier fill", func(r *mtl.ExecutionRequest) { r.ExecutedAt = at.Add(-time.Second) }},
		} {
			bad := fill
			tc.change(&bad)
			if _, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: trade.ID, Exit: bad}); !errors.Is(err, mtl.ErrInvalidRequest) {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if tc.name != "over exit" {
				if _, err = s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: trade.ID, Entry: bad}); !errors.Is(err, mtl.ErrInvalidRequest) {
					t.Fatalf("entry %s: %v", tc.name, err)
				}
			}
		}
		if _, err = s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: uuid.New(), Entry: fill}); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing entry trade: %v", err)
		}
		if _, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: uuid.New(), Exit: fill}); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing exit trade: %v", err)
		}
		if _, err = s.Exchange.Edit(ctx, mtl.EditExchangeRequest{ID: ex.ID, Name: ex.Name}); err != nil {
			t.Fatal(err)
		}
		if _, err = s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: trade.ID, Entry: fill}); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("unknown fee: %v", err)
		}
		if _, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: trade.ID, Exit: fill}); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("unknown exit fee: %v", err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Executions) != 1 || got.RemainingQuantity.String() != "1" || got.CompletedAt != nil {
			t.Fatalf("failed changes persisted: %+v", got)
		}
		if _, err = s.Exchange.Edit(ctx, mtl.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate, ExitFeeRate: &rate}); err != nil {
			t.Fatal(err)
		}
		if _, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: trade.ID, Exit: fill}); err != nil {
			t.Fatal(err)
		}
		if _, err = s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: trade.ID, Entry: fill}); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("entry after closure: %v", err)
		}
		if _, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: trade.ID, Exit: fill}); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("exit after closure: %v", err)
		}
		first := fill
		first.Price = mtl.MustDecimal("1")
		rounded, err := s.RecordTrade(ctx, mtl.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "ETHUSDT", Direction: mtl.Long, SettlementCurrency: "USDT", FirstEntry: first})
		if err != nil {
			t.Fatal(err)
		}
		addition := fill
		addition.Quantity = mtl.MustDecimal("2")
		addition.Price = mtl.MustDecimal("2")
		rebate := mtl.MustDecimal("-0.123")
		addition.ActualFee = &rebate
		rounded, err = s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: rounded.ID, Entry: addition})
		if err != nil {
			t.Fatal(err)
		}
		if rounded.AverageEntryPrice.String() != "1.666666666666666667" {
			t.Fatalf("rounded weighted average: %s", rounded.AverageEntryPrice)
		}
		partial := addition
		partial.Quantity = mtl.MustDecimal("1")
		charge := mtl.MustDecimal("0.023")
		partial.ActualFee = &charge
		rounded, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: rounded.ID, Exit: partial})
		if err != nil {
			t.Fatal(err)
		}
		if rounded.TotalExecutionFees.String() != "-0.1" || rounded.NetProfitLoss.String() != "0.433333333333333333" || rounded.Executions[1].Fee.ActualAmount.String() != "-0.123" || rounded.Executions[2].Fee.ActualAmount.String() != "0.023" {
			t.Fatalf("scaling actual fees: %+v", rounded)
		}
		complete := addition
		complete.ActualFee = nil
		complete.Quantity = mtl.MustDecimal("2")
		rounded, err = s.CloseTrade(ctx, mtl.CloseTradeRequest{TradeID: rounded.ID, Exit: complete})
		if err != nil {
			t.Fatal(err)
		}
		if rounded.NetProfitLoss.String() != "1.099999999999999999" {
			t.Fatalf("profit against recorded average: %s", rounded.NetProfitLoss)
		}

	})
}
