package stoplore_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"

	stoplore "github.com/saintmili/stoplore"
)

func TestScaleEntriesAverageAgainstRemainingPosition(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := stoplore.MustDecimal("0.04")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Scaling", EntryFeeRate: &rate, ExitFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Scale", Rules: "Add exposure"})
		if err != nil {
			t.Fatal(err)
		}
		for _, direction := range []stoplore.TradeDirection{stoplore.Long, stoplore.Short} {
			pnl, partialNet, finalNet := "20", "19.86", "59.728"
			if direction == stoplore.Short {
				pnl, partialNet, finalNet = "-20", "-20.14", "-60.272"
			}
			at := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC)
			fill := func(price string, hour int) stoplore.ExecutionRequest {
				return stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal(price), ExecutedAt: at.Add(time.Duration(hour) * time.Hour)}
			}
			trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: direction, SettlementCurrency: "USDT", FirstEntry: fill("100", 0)})
			if err != nil {
				t.Fatal(err)
			}
			trade, err = s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: trade.ID, Entry: fill("120", 1)})
			if err != nil {
				t.Fatal(err)
			}
			if trade.RemainingQuantity.String() != "2" || trade.AverageEntryPrice.String() != "110" || trade.TotalExecutionFees.String() != "0.088" || trade.NetProfitLoss.String() != "-0.088" || trade.CompletedAt != nil {
				t.Fatalf("scaled trade: %+v", trade)
			}
			trade, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: trade.ID, Exit: fill("130", 2)})
			if err != nil {
				t.Fatal(err)
			}
			if trade.RemainingQuantity.String() != "1" || trade.AverageEntryPrice.String() != "110" || trade.RealizedPriceProfitLoss.String() != pnl || trade.NetProfitLoss.String() != partialNet || trade.CompletedAt != nil {
				t.Fatalf("partial exit: %+v", trade)
			}
			trade, err = s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: trade.ID, Entry: fill("90", 3)})
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
			if got.AverageEntryPrice.String() != "100" || got.RemainingQuantity.String() != "2" || len(got.Executions) != 4 || got.Executions[1].Kind != stoplore.Entry || got.Executions[1].Fee.Amount.String() != "0.048" || got.Executions[1].Fee.Rate.String() != "0.04" {
				t.Fatalf("reopened: %+v", got)
			}
			finalFill := fill("120", 4)
			finalFill.Quantity = stoplore.MustDecimal("2")
			closed, err := s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: trade.ID, Exit: finalFill})
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
			if got.NetProfitLoss.String() != finalNet || len(got.Executions) != 5 || got.CompletedAt == nil || got.Executions[2].Kind != stoplore.Exit || got.Executions[3].Kind != stoplore.Entry {
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
		rate := stoplore.MustDecimal("0")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Sequence", EntryFeeRate: &rate, ExitFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Sequence", Rules: "No overselling"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		fill := stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("100"), ExecutedAt: at}
		trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "ETHUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: fill})
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name   string
			change func(*stoplore.ExecutionRequest)
		}{
			{"over exit", func(r *stoplore.ExecutionRequest) { r.Quantity = stoplore.MustDecimal("2") }},
			{"zero quantity", func(r *stoplore.ExecutionRequest) { r.Quantity = stoplore.MustDecimal("0") }},
			{"negative price", func(r *stoplore.ExecutionRequest) { r.Price = stoplore.MustDecimal("-1") }},
			{"missing time", func(r *stoplore.ExecutionRequest) { r.ExecutedAt = time.Time{} }},
			{"earlier fill", func(r *stoplore.ExecutionRequest) { r.ExecutedAt = at.Add(-time.Second) }},
		} {
			bad := fill
			tc.change(&bad)
			if _, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: trade.ID, Exit: bad}); !errors.Is(err, stoplore.ErrInvalidRequest) {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if tc.name != "over exit" {
				if _, err = s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: trade.ID, Entry: bad}); !errors.Is(err, stoplore.ErrInvalidRequest) {
					t.Fatalf("entry %s: %v", tc.name, err)
				}
			}
		}
		if _, err = s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: uuid.New(), Entry: fill}); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing entry trade: %v", err)
		}
		if _, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: uuid.New(), Exit: fill}); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing exit trade: %v", err)
		}
		if _, err = s.Exchange.Edit(ctx, stoplore.EditExchangeRequest{ID: ex.ID, Name: ex.Name}); err != nil {
			t.Fatal(err)
		}
		if _, err = s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: trade.ID, Entry: fill}); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("unknown fee: %v", err)
		}
		if _, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: trade.ID, Exit: fill}); !errors.Is(err, stoplore.ErrInvalidRequest) {
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
		if _, err = s.Exchange.Edit(ctx, stoplore.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate, ExitFeeRate: &rate}); err != nil {
			t.Fatal(err)
		}
		if _, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: trade.ID, Exit: fill}); err != nil {
			t.Fatal(err)
		}
		if _, err = s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: trade.ID, Entry: fill}); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("entry after closure: %v", err)
		}
		if _, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: trade.ID, Exit: fill}); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("exit after closure: %v", err)
		}
		first := fill
		first.Price = stoplore.MustDecimal("1")
		rounded, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "ETHUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: first})
		if err != nil {
			t.Fatal(err)
		}
		addition := fill
		addition.Quantity = stoplore.MustDecimal("2")
		addition.Price = stoplore.MustDecimal("2")
		rebate := stoplore.MustDecimal("-0.123")
		addition.ActualFee = &rebate
		rounded, err = s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: rounded.ID, Entry: addition})
		if err != nil {
			t.Fatal(err)
		}
		if rounded.AverageEntryPrice.String() != "1.666666666666666667" {
			t.Fatalf("rounded weighted average: %s", rounded.AverageEntryPrice)
		}
		partial := addition
		partial.Quantity = stoplore.MustDecimal("1")
		charge := stoplore.MustDecimal("0.023")
		partial.ActualFee = &charge
		rounded, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: rounded.ID, Exit: partial})
		if err != nil {
			t.Fatal(err)
		}
		if rounded.TotalExecutionFees.String() != "-0.1" || rounded.NetProfitLoss.String() != "0.433333333333333333" || rounded.Executions[1].Fee.ActualAmount.String() != "-0.123" || rounded.Executions[2].Fee.ActualAmount.String() != "0.023" {
			t.Fatalf("scaling actual fees: %+v", rounded)
		}
		complete := addition
		complete.ActualFee = nil
		complete.Quantity = stoplore.MustDecimal("2")
		rounded, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: rounded.ID, Exit: complete})
		if err != nil {
			t.Fatal(err)
		}
		if rounded.NetProfitLoss.String() != "1.099999999999999999" {
			t.Fatalf("profit against recorded average: %s", rounded.NetProfitLoss)
		}

	})
}
