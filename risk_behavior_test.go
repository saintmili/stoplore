package stoplore_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"

	stoplore "github.com/saintmili/stoplore"
)

func TestCompletedRiskComparisonPreservesOriginalBudget(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := stoplore.MustDecimal("0")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Risk", EntryFeeRate: &rate, ExitFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Risk setup", Rules: "Rules"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		risk := stoplore.MustDecimal("50")
		tr, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{OriginalPlannedRisk: &risk, AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("100"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.ReviseTradePlan(ctx, stoplore.ReviseTradePlanRequest{TradeID: tr.ID, RevisedAt: at.Add(time.Hour), Plan: stoplore.TradePlan{Stop: decimalPtr("80")}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: tr.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("210"), ActualFee: decimalPtr("10"), ExecutedAt: at.Add(2 * time.Hour)}})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.GetTrade(ctx, tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertOptionalDecimal(t, got.OriginalPlannedRisk, "50")
		report, err := s.ReportPerformance(ctx, stoplore.ReportFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Completed) != 1 {
			t.Fatalf("completed results: %+v", report)
		}
		result := report.Completed[0]
		if result.TradeID != tr.ID || result.SetupVersionID != setup.Versions[0].ID || result.SettlementCurrency != "USDT" || result.NetProfitLoss.String() != "100" {
			t.Fatalf("result: %+v", result)
		}
		assertOptionalDecimal(t, result.OriginalPlannedRisk, "50")
		assertOptionalDecimal(t, result.RMultiple, "2")
		correction := stoplore.CorrectOriginalPlannedRiskRequest{TradeID: tr.ID, OriginalPlannedRisk: decimalPtr("25"), CorrectedAt: at.Add(3 * time.Hour), Reason: "Original budget mistyped"}
		corrected, err := s.CorrectOriginalPlannedRisk(ctx, correction)
		if err != nil {
			t.Fatal(err)
		}
		assertOptionalDecimal(t, corrected.OriginalPlannedRisk, "25")
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		corrected, err = s.GetTrade(ctx, tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(corrected.RiskCorrections) != 1 || corrected.RiskCorrections[0].Reason != correction.Reason || !corrected.RiskCorrections[0].CorrectedAt.Equal(correction.CorrectedAt) {
			t.Fatalf("risk history: %+v", corrected.RiskCorrections)
		}
		assertOptionalDecimal(t, corrected.RiskCorrections[0].Before, "50")
		assertOptionalDecimal(t, corrected.RiskCorrections[0].After, "25")
		report, err = s.ReportPerformance(ctx, stoplore.ReportFilter{})
		if err != nil {
			t.Fatal(err)
		}
		assertOptionalDecimal(t, report.Completed[0].RMultiple, "4")
		correction.TradeID = uuid.New()
		if _, err = s.CorrectOriginalPlannedRisk(ctx, correction); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing trade: %v", err)
		}
		correction.TradeID = tr.ID
		correction.Reason = " "
		if _, err = s.CorrectOriginalPlannedRisk(ctx, correction); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("missing reason: %v", err)
		}
		correction.Reason = "Correct unknown budget"
		correction.CorrectedAt = time.Time{}
		if _, err = s.CorrectOriginalPlannedRisk(ctx, correction); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("missing timestamp: %v", err)
		}

	})
}

func TestMissingAndNonpositiveRiskRetainsMonetaryReports(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		zero := stoplore.MustDecimal("0")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Unavailable risk", EntryFeeRate: &zero, ExitFeeRate: &zero})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Risk setup", Rules: "Rules"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for _, risk := range []*stoplore.Decimal{nil, decimalPtr("0"), decimalPtr("-50")} {
			tr, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{OriginalPlannedRisk: risk, AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("100"), ExecutedAt: at}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: tr.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("200"), ExecutedAt: at.Add(time.Hour)}})
			if err != nil {
				t.Fatal(err)
			}
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		report, err := s.ReportPerformance(ctx, stoplore.ReportFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Completed) != 3 || len(report.Groups) != 1 || report.Groups[0].EligibleCount != 3 || report.Groups[0].ExcludedCount != 0 || report.Groups[0].TotalNetProfit.String() != "300" {
			t.Fatalf("report: %+v", report)
		}
		for _, result := range report.Completed {
			if result.RMultiple != nil {
				t.Fatalf("unavailable R: %+v", result)
			}
		}
	})
}
