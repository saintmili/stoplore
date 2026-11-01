package mtl_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"

	mtl "github.com/saintmili/mtl"
)

func reviewTrade(t *testing.T, s *mtl.Service) mtl.Trade {
	t.Helper()
	ctx := context.Background()
	rate := mtl.MustDecimal("0")
	ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Review venue", EntryFeeRate: &rate})
	if err != nil {
		t.Fatal(err)
	}
	account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	setup, err := s.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Breakout", Rules: "Enter above resistance"})
	if err != nil {
		t.Fatal(err)
	}
	trade, err := s.RecordTrade(ctx, mtl.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: mtl.Long, SettlementCurrency: "USDT", FirstEntry: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}})
	if err != nil {
		t.Fatal(err)
	}
	return trade
}

func TestTradeContextAndOptionalAdherenceSurviveReopening(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade := reviewTrade(t, s)
		if trade.Review.Adherence != mtl.Unassessed {
			t.Fatalf("new trade assessment: %+v", trade.Review)
		}
		_, err = s.SetTradeReview(ctx, mtl.SetTradeReviewRequest{TradeID: trade.ID, Tags: []string{" trend ", "breakout", "trend"}, Adherence: mtl.FollowedRules, Notes: "Waited for confirmation"})
		if err != nil {
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
		if len(got.Tags) != 2 || got.Tags[0] != "trend" || got.Tags[1] != "breakout" || got.Review.Adherence != mtl.FollowedRules || got.Review.Notes != "Waited for confirmation" {
			t.Fatalf("saved context: %+v", got)
		}
	})
}

func TestReviewCorrectionsRetainPreviousAssessmentsAndTags(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade := reviewTrade(t, s)
		before := time.Now()
		for _, req := range []mtl.SetTradeReviewRequest{
			{TradeID: trade.ID, Tags: []string{"trend"}, Adherence: mtl.FollowedRules, Notes: "Followed entry rule"},
			{TradeID: trade.ID, Tags: []string{"range"}, Adherence: mtl.Deviated, Notes: "Entered before confirmation"},
			{TradeID: trade.ID, Adherence: mtl.Unassessed, Notes: "Need to review chart"},
		} {
			if _, err = s.SetTradeReview(ctx, req); err != nil {
				t.Fatal(err)
			}
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Tags) != 0 || got.Review.Adherence != mtl.Unassessed || got.Review.Notes != "Need to review chart" {
			t.Fatalf("corrected review: %+v", got)
		}
		if len(got.ReviewHistory) != 3 {
			t.Fatalf("history: %+v", got.ReviewHistory)
		}
		first, followed, deviated := got.ReviewHistory[0], got.ReviewHistory[1], got.ReviewHistory[2]
		if first.PreviousReview.Adherence != mtl.Unassessed || len(first.PreviousTags) != 0 || first.CorrectedAt.Before(before) || first.CorrectedAt.After(time.Now()) {
			t.Fatalf("original review: %+v", first)
		}
		if len(followed.PreviousTags) != 1 || followed.PreviousTags[0] != "trend" || followed.PreviousReview.Adherence != mtl.FollowedRules || followed.PreviousReview.Notes != "Followed entry rule" || followed.CorrectedAt.Before(first.CorrectedAt) {
			t.Fatalf("followed review: %+v", followed)
		}
		if len(deviated.PreviousTags) != 1 || deviated.PreviousTags[0] != "range" || deviated.PreviousReview.Adherence != mtl.Deviated || deviated.PreviousReview.Notes != "Entered before confirmation" || deviated.CorrectedAt.Before(followed.CorrectedAt) {
			t.Fatalf("deviated review: %+v", deviated)
		}
		trades, err := s.ListTrades(ctx)
		if err != nil || len(trades) != 1 || len(trades[0].ReviewHistory) != 3 {
			t.Fatalf("list context: %+v %v", trades, err)
		}
	})
}

func TestInvalidTradeReviewsLeaveContextAndHistoryUnchanged(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade := reviewTrade(t, s)
		for _, req := range []mtl.SetTradeReviewRequest{
			{TradeID: uuid.Nil},
			{TradeID: trade.ID, Adherence: "MAYBE"},
			{TradeID: trade.ID, Tags: []string{"valid", " "}},
		} {
			if _, err = s.SetTradeReview(ctx, req); !errors.Is(err, mtl.ErrInvalidRequest) {
				t.Fatalf("invalid review %+v: %v", req, err)
			}
		}
		if _, err = s.SetTradeReview(ctx, mtl.SetTradeReviewRequest{TradeID: uuid.New()}); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing trade: %v", err)
		}
		got, err := s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Tags) != 0 || len(got.ReviewHistory) != 0 || got.Review.Adherence != mtl.Unassessed || got.Review.Notes != "" {
			t.Fatalf("invalid requests changed review: %+v", got)
		}
	})
}
