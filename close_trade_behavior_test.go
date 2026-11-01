package mtl_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"

	mtl "github.com/saintmili/mtl"
)

func TestFullyCloseTradeRetainsDirectionalRecordedCostResult(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := mtl.MustDecimal("0.04")
		ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Close venue", EntryFeeRate: &rate, ExitFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Breakout", Rules: "Break resistance"})
		if err != nil {
			t.Fatal(err)
		}
		entryAt := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC)
		closeAt := time.Date(2026, 3, 1, 0, 0, 0, 987654321, time.UTC)
		for _, tc := range []struct {
			direction                       mtl.TradeDirection
			price, pnl, fees, net, exitRate string
		}{
			{mtl.Long, "62000", "200", "4.88", "195.12", "0.04"},
			{mtl.Short, "58000", "200", "4.72", "195.28", "0.04"},
			{mtl.Short, "62000", "-200", "4.88", "-204.88", "0.04"},
			{mtl.Long, "62000", "200", "5.5", "194.5", "0.05"},
		} {
			exitRate := mtl.MustDecimal(tc.exitRate)
			if _, err = s.Exchange.Edit(ctx, mtl.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate, ExitFeeRate: &exitRate}); err != nil {
				t.Fatal(err)
			}
			trade, err := s.RecordTrade(ctx, mtl.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: tc.direction, SettlementCurrency: "USDT", FirstEntry: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("0.1"), Price: mtl.MustDecimal("60000"), ExecutedAt: entryAt}})
			if err != nil {
				t.Fatal(err)
			}
			closed, err := s.CloseTrade(ctx, mtl.CloseTradeRequest{TradeID: trade.ID, Exit: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("0.1"), Price: mtl.MustDecimal(tc.price), ExecutedAt: closeAt}})
			if err != nil {
				t.Fatal(err)
			}
			if closed.CompletedAt == nil || !closed.CompletedAt.Equal(closeAt) || !closed.RemainingQuantity.IsZero() {
				t.Fatalf("completion: %+v", closed)
			}
			if closed.RealizedPriceProfitLoss.String() != tc.pnl || closed.TotalExecutionFees.String() != tc.fees || closed.NetProfitLoss.String() != tc.net || !closed.RecordedFunding.IsZero() || closed.CostBasis != mtl.RecordedCosts {
				t.Fatalf("result: %+v", closed)
			}
			if len(closed.Executions) != 2 || closed.Executions[1].Kind != mtl.Exit || closed.Executions[1].Fee.Rate == nil || closed.Executions[1].Fee.Rate.String() != tc.exitRate || !closed.Executions[1].ExecutedAt.Equal(closeAt) {
				t.Fatalf("exit: %+v", closed.Executions)
			}
			changed := mtl.MustDecimal("1")
			if _, err = s.Exchange.Edit(ctx, mtl.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate, ExitFeeRate: &changed}); err != nil {
				t.Fatal(err)
			}
			s, err = b.Open()
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.GetTrade(ctx, trade.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.NetProfitLoss.String() != tc.net || got.RealizedPriceProfitLoss.String() != tc.pnl || got.TotalExecutionFees.String() != tc.fees || got.CompletedAt == nil || !got.CompletedAt.Equal(closeAt) || got.Executions[1].Fee.Rate.String() != tc.exitRate {
				t.Fatalf("reopened: %+v", got)
			}
			if _, err = s.Exchange.Edit(ctx, mtl.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate, ExitFeeRate: &rate}); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func TestCompleteExitRequiresKnownFeeAndValidFullFill(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		zero := mtl.MustDecimal("0")
		ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Zero fees", EntryFeeRate: &zero})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Rules", Rules: "Enter"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		trade, err := s.RecordTrade(ctx, mtl.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "ETHUSDC", Direction: mtl.Long, SettlementCurrency: "USDC", FirstEntry: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		req := mtl.CloseTradeRequest{TradeID: trade.ID, Exit: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("110"), ExecutedAt: at.Add(time.Hour)}}
		if _, err = s.CloseTrade(ctx, req); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("unknown exit fee: %v", err)
		}
		if _, err = s.Exchange.Edit(ctx, mtl.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &zero, ExitFeeRate: &zero}); err != nil {
			t.Fatal(err)
		}
		for _, change := range []func(*mtl.CloseTradeRequest){
			func(r *mtl.CloseTradeRequest) { r.Exit.Quantity = mtl.MustDecimal("0.5") },
			func(r *mtl.CloseTradeRequest) { r.Exit.Quantity = mtl.MustDecimal("2") },
			func(r *mtl.CloseTradeRequest) { r.Exit.Price = mtl.MustDecimal("0") },
			func(r *mtl.CloseTradeRequest) { r.Exit.ExecutedAt = time.Time{} },
			func(r *mtl.CloseTradeRequest) { r.Exit.ExecutedAt = at.Add(-time.Second) },
		} {
			bad := req
			change(&bad)
			if _, err = s.CloseTrade(ctx, bad); !errors.Is(err, mtl.ErrInvalidRequest) {
				t.Fatalf("invalid exit: %v", err)
			}
		}
		bad := req
		bad.TradeID = uuid.New()
		if _, err = s.CloseTrade(ctx, bad); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing trade: %v", err)
		}
		got, err := s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Executions) != 1 || got.RemainingQuantity.String() != "1" || got.CompletedAt != nil {
			t.Fatalf("failed exits changed trade: %+v", got)
		}
		closed, err := s.CloseTrade(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if closed.Executions[1].Fee.Rate == nil || !closed.Executions[1].Fee.Rate.IsZero() || !closed.Executions[1].Fee.Amount.IsZero() || closed.NetProfitLoss.String() != "10" {
			t.Fatalf("known zero fee: %+v", closed)
		}
		if _, err = s.CloseTrade(ctx, req); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("already completed: %v", err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Executions) != 2 || got.Executions[1].Fee.Rate == nil || !got.Executions[1].Fee.Rate.IsZero() || got.NetProfitLoss.String() != "10" || got.SettlementCurrency != "USDC" {
			t.Fatalf("reopened zero fee: %+v", got)
		}
	})
}
