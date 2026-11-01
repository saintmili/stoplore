package stoplore_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	stoplore "github.com/saintmili/stoplore"
	"testing"
	"time"
)

func TestRecordTradePreservesFirstEntryAndContext(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := stoplore.MustDecimal("0.04")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Venue", EntryFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Breakout", Rules: "Price breaks resistance"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.FixedZone("local", 12600))
		margin, leverage := stoplore.MustDecimal("50.001"), stoplore.MustDecimal("3")
		for _, direction := range []stoplore.TradeDirection{stoplore.Long, stoplore.Short} {
			trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: direction, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("60000.000000000000001"), ExecutedAt: at, Margin: &margin, Leverage: &leverage}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.ReviseSetup(ctx, stoplore.ReviseSetupRequest{SetupID: setup.ID, Rules: "Changed rules"}); err != nil {
				t.Fatal(err)
			}
			changed := stoplore.MustDecimal("1")
			if _, err = s.Exchange.Edit(ctx, stoplore.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &changed}); err != nil {
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
			if got.AccountID != account.ID || got.SetupVersionID != setup.Versions[0].ID || got.Direction != direction || got.Symbol != "BTCUSDT" || got.SettlementCurrency != "USDT" {
				t.Fatalf("trade context: %+v", got)
			}
			if got.RemainingQuantity.String() != "0.1" || got.AverageEntryPrice.String() != "60000.000000000000001" || len(got.Executions) != 1 {
				t.Fatalf("position: %+v", got)
			}
			fill := got.Executions[0]
			if fill.Kind != stoplore.Entry || !fill.ExecutedAt.Equal(at) || fill.Quantity.String() != "0.1" || fill.Price.String() != "60000.000000000000001" || fill.Margin.String() != "50.001" || fill.Leverage.String() != "3" {
				t.Fatalf("entry: %+v", fill)
			}
			if fill.Fee.Rate == nil || fill.Fee.Rate.String() != "0.04" || fill.Fee.Amount.String() != "2.4" {
				t.Fatalf("fee snapshot: %+v", fill.Fee)
			}
			if _, err = s.Exchange.Edit(ctx, stoplore.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate}); err != nil {
				t.Fatal(err)
			}
		}
		trades, err := s.ListTrades(ctx)
		if err != nil || len(trades) != 2 {
			t.Fatalf("trades: %+v, %v", trades, err)
		}
	})
}

func TestTradeFeesDistinguishUnknownZeroAndRoundedAmounts(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Fees"})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Rules", Rules: "Enter on breakout"})
		if err != nil {
			t.Fatal(err)
		}
		req := stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "ETHUSDC", Direction: stoplore.Short, SettlementCurrency: "USDC", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("1"), ExecutedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
		if _, err = s.RecordTrade(ctx, req); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("unknown fee: %v", err)
		}
		for _, tc := range []struct{ rate, fee string }{{"0", "0"}, {"0.00000000000000005", "0.000000000000000001"}} {
			rate := stoplore.MustDecimal(tc.rate)
			if _, err = s.Exchange.Edit(ctx, stoplore.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate}); err != nil {
				t.Fatal(err)
			}
			trade, err := s.RecordTrade(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			s, err = b.Open()
			if err != nil {
				t.Fatal(err)
			}
			saved, err := s.GetTrade(ctx, trade.ID)
			if err != nil {
				t.Fatal(err)
			}
			if saved.Executions[0].Fee.Amount.String() != tc.fee || saved.Executions[0].Fee.Rate == nil || saved.Executions[0].Fee.Rate.String() != tc.rate {
				t.Fatalf("fee: %+v", saved.Executions[0].Fee)
			}
		}
		invalid := []func(*stoplore.RecordTradeRequest){
			func(r *stoplore.RecordTradeRequest) { r.AccountID = uuid.Nil },
			func(r *stoplore.RecordTradeRequest) { r.SetupVersionID = uuid.Nil },
			func(r *stoplore.RecordTradeRequest) { r.Symbol = " " },
			func(r *stoplore.RecordTradeRequest) { r.Direction = "BUY" },
			func(r *stoplore.RecordTradeRequest) { r.SettlementCurrency = "BTC" },
			func(r *stoplore.RecordTradeRequest) { r.FirstEntry.Quantity = stoplore.MustDecimal("0") },
			func(r *stoplore.RecordTradeRequest) { r.FirstEntry.Price = stoplore.MustDecimal("-1") },
			func(r *stoplore.RecordTradeRequest) { r.FirstEntry.ExecutedAt = time.Time{} },
			func(r *stoplore.RecordTradeRequest) { v := stoplore.MustDecimal("-1"); r.FirstEntry.Margin = &v },
			func(r *stoplore.RecordTradeRequest) { v := stoplore.MustDecimal("0"); r.FirstEntry.Leverage = &v },
		}
		for i, change := range invalid {
			bad := req
			change(&bad)
			if _, err = s.RecordTrade(ctx, bad); !errors.Is(err, stoplore.ErrInvalidRequest) {
				t.Fatalf("invalid request %d: %v", i, err)
			}
		}
		bad := req
		bad.AccountID = uuid.New()
		if _, err = s.RecordTrade(ctx, bad); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing account: %v", err)
		}
		bad = req
		bad.SetupVersionID = uuid.New()
		if _, err = s.RecordTrade(ctx, bad); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing setup version: %v", err)
		}
		if _, err = s.GetTrade(ctx, uuid.New()); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing trade: %v", err)
		}
		trades, err := s.ListTrades(ctx)
		if err != nil || len(trades) != 2 {
			t.Fatalf("failed requests changed journal: %+v %v", trades, err)
		}
	})
}
