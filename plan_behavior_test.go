package mtl_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	mtl "github.com/saintmili/mtl"
	"reflect"
	"testing"
	"time"
)

func planTradeRequest(t *testing.T, s *mtl.Service) mtl.RecordTradeRequest {
	t.Helper()
	ctx := context.Background()
	zero := mtl.MustDecimal("0")
	ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Plans", EntryFeeRate: &zero})
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
	return mtl.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: mtl.Long, SettlementCurrency: "USDT", FirstEntry: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
}

func TestOriginalTradePlanMayBeMissingOrPartial(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		req := planTradeRequest(t, s)
		historical, err := s.RecordTrade(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		target := mtl.MustDecimal("125.000000000000001")
		req.OriginalPlan = &mtl.TradePlan{EntryReason: "Breakout", Target: &target, Notes: "Wait for volume"}
		planned, err := s.RecordTrade(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		historical, err = s.GetTrade(ctx, historical.ID)
		if err != nil {
			t.Fatal(err)
		}
		planned, err = s.GetTrade(ctx, planned.ID)
		if err != nil {
			t.Fatal(err)
		}
		if historical.OriginalPlan != nil || len(historical.PlanRevisions) != 0 || len(historical.PlanCorrections) != 0 {
			t.Fatalf("invented historical plan: %+v", historical)
		}
		if !reflect.DeepEqual(planned.OriginalPlan, req.OriginalPlan) {
			t.Fatalf("original plan: %+v", planned.OriginalPlan)
		}
	})
}

func TestDatedPlanRevisionsRetainOriginalAndEarlierDecisions(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		req := planTradeRequest(t, s)
		stop := mtl.MustDecimal("90")
		target := mtl.MustDecimal("120")
		req.OriginalPlan = &mtl.TradePlan{EntryReason: "Breakout", Stop: &stop, Target: &target}
		trade, err := s.RecordTrade(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
		moved := mtl.MustDecimal("100")
		first, err := s.ReviseTradePlan(ctx, mtl.ReviseTradePlanRequest{TradeID: trade.ID, RevisedAt: at, Plan: mtl.TradePlan{Stop: &moved, Notes: "Protect capital"}})
		if err != nil {
			t.Fatal(err)
		}
		second, err := s.ReviseTradePlan(ctx, mtl.ReviseTradePlanRequest{TradeID: trade.ID, RevisedAt: at.Add(time.Hour), Plan: mtl.TradePlan{Notes: "Let it run"}})
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
		if !reflect.DeepEqual(got.OriginalPlan, req.OriginalPlan) || len(got.PlanRevisions) != 2 {
			t.Fatalf("lost plans: %+v", got)
		}
		if first.ID == second.ID || first.ID.String() == "00000000-0000-0000-0000-000000000000" || !got.PlanRevisions[0].RevisedAt.Equal(at) || got.PlanRevisions[0].Plan.Stop.String() != "100" || got.PlanRevisions[1].Plan.Notes != "Let it run" {
			t.Fatalf("revisions: %+v", got.PlanRevisions)
		}
	})
}

func TestPlanningCorrectionsRetainPreviousValuesWithoutNewDecisions(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		req := planTradeRequest(t, s)
		req.OriginalPlan = &mtl.TradePlan{EntryReason: "Brekout"}
		trade, err := s.RecordTrade(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		revision, err := s.ReviseTradePlan(ctx, mtl.ReviseTradePlanRequest{TradeID: trade.ID, RevisedAt: at, Plan: mtl.TradePlan{Notes: "Tigthten stop"}})
		if err != nil {
			t.Fatal(err)
		}
		corrected := mtl.TradePlan{EntryReason: "Breakout"}
		_, err = s.CorrectTradePlan(ctx, mtl.CorrectTradePlanRequest{TradeID: trade.ID, CorrectedAt: at.Add(time.Hour), Reason: "Typo", Plan: &corrected})
		if err != nil {
			t.Fatal(err)
		}
		fixedRevision := mtl.TradePlan{Notes: "Tighten stop"}
		_, err = s.CorrectTradePlan(ctx, mtl.CorrectTradePlanRequest{TradeID: trade.ID, RevisionID: revision.ID, CorrectedAt: at.Add(2 * time.Hour), Reason: "Typo", Plan: &fixedRevision})
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.CorrectTradePlan(ctx, mtl.CorrectTradePlanRequest{TradeID: trade.ID, CorrectedAt: at.Add(3 * time.Hour), Reason: "Reason belongs to another trade", Plan: nil})
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
		if got.OriginalPlan != nil || len(got.PlanRevisions) != 1 || got.PlanRevisions[0].Plan.Notes != "Tighten stop" || !got.PlanRevisions[0].RevisedAt.Equal(at) || len(got.PlanCorrections) != 3 {
			t.Fatalf("correction changed decisions: %+v", got)
		}
		first, second, third := got.PlanCorrections[0], got.PlanCorrections[1], got.PlanCorrections[2]
		if first.Before.EntryReason != "Brekout" || first.After.EntryReason != "Breakout" || first.Reason != "Typo" || !first.CorrectedAt.Equal(at.Add(time.Hour)) || second.RevisionID != revision.ID || second.Before.Notes != "Tigthten stop" || third.Before.EntryReason != "Breakout" || third.After != nil {
			t.Fatalf("lost correction trace: %+v", got.PlanCorrections)
		}
	})
}

func TestPlanningRequestsRejectInvalidPricesAndUnknownRecords(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		req := planTradeRequest(t, s)
		invalid := mtl.MustDecimal("-1")
		req.OriginalPlan = &mtl.TradePlan{Stop: &invalid}
		if _, err = s.RecordTrade(ctx, req); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("negative stop accepted: %v", err)
		}
		req.OriginalPlan = nil
		trade, err := s.RecordTrade(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		for _, bad := range []mtl.ReviseTradePlanRequest{{TradeID: trade.ID}, {TradeID: trade.ID, RevisedAt: at, Plan: mtl.TradePlan{Target: &invalid}}} {
			if _, err = s.ReviseTradePlan(ctx, bad); !errors.Is(err, mtl.ErrInvalidRequest) {
				t.Fatalf("invalid revision: %v", err)
			}
		}
		if _, err = s.ReviseTradePlan(ctx, mtl.ReviseTradePlanRequest{TradeID: uuid.New(), RevisedAt: at}); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("unknown trade: %v", err)
		}
		for _, bad := range []mtl.CorrectTradePlanRequest{
			{TradeID: trade.ID, Reason: "Missing date"},
			{TradeID: trade.ID, CorrectedAt: at, Reason: " "},
			{TradeID: trade.ID, CorrectedAt: at, Reason: "Bad price", Plan: &mtl.TradePlan{Stop: &invalid}},
			{TradeID: trade.ID, RevisionID: uuid.New(), CorrectedAt: at, Reason: "Missing plan"},
		} {
			if _, err = s.CorrectTradePlan(ctx, bad); !errors.Is(err, mtl.ErrInvalidRequest) {
				t.Fatalf("invalid correction: %v", err)
			}
		}
		for _, bad := range []mtl.CorrectTradePlanRequest{
			{TradeID: uuid.New(), CorrectedAt: at, Reason: "Unknown trade"},
			{TradeID: trade.ID, RevisionID: uuid.New(), CorrectedAt: at, Reason: "Unknown revision", Plan: &mtl.TradePlan{}},
		} {
			if _, err = s.CorrectTradePlan(ctx, bad); !errors.Is(err, mtl.ErrNotFound) {
				t.Fatalf("unknown correction target: %v", err)
			}
		}
		got, err := s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.OriginalPlan != nil || len(got.PlanRevisions) != 0 || len(got.PlanCorrections) != 0 {
			t.Fatalf("failed requests changed plans: %+v", got)
		}
	})
}
