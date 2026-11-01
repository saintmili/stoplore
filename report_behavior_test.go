package stoplore_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	stoplore "github.com/saintmili/stoplore"
)

func TestCompletedPerformanceUsesLifetimeRecordedCosts(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := stoplore.MustDecimal("0.04")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Reports", EntryFeeRate: &rate, ExitFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Breakout", Rules: "Rules"})
		if err != nil {
			t.Fatal(err)
		}
		jan := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC)
		march := time.Date(2026, 3, 1, 0, 0, 0, 123456789, time.UTC)
		trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("60000"), ExecutedAt: jan}})
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: trade.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("62000"), ExecutedAt: march}})
		if err != nil {
			t.Fatal(err)
		}
		for _, amount := range []string{"1", "-5"} {
			_, err = s.RecordFundingPayment(ctx, stoplore.RecordFundingPaymentRequest{TradeID: trade.ID, Amount: stoplore.MustDecimal(amount), PaidAt: jan.Add(time.Hour)})
			if err != nil {
				t.Fatal(err)
			}
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		report, err := s.ReportPerformance(ctx, stoplore.ReportFilter{Start: march, End: march.AddDate(0, 1, 0)})
		if err != nil {
			t.Fatal(err)
		}
		if report.CostBasis != stoplore.RecordedCosts || len(report.Groups) != 1 || len(report.Ongoing) != 0 {
			t.Fatalf("report: %+v", report)
		}
		g := report.Groups[0]
		if g.SetupID != setup.ID || g.SetupVersionID != setup.Versions[0].ID || g.SetupVersion != 1 || g.SetupName != "Breakout" || g.SettlementCurrency != "USDT" || g.EligibleCount != 1 || g.ExcludedCount != 0 || g.Wins != 1 || g.Losses != 0 || g.BreakEven != 0 || g.TotalNetProfit.String() != "191.12" {
			t.Fatalf("group: %+v", g)
		}
		assertOptionalDecimal(t, g.AverageNetProfit, "191.12")
		assertOptionalDecimal(t, g.AverageWin, "191.12")
		assertOptionalDecimal(t, g.WinRate, "1")
		if g.AverageLoss != nil || g.ProfitFactor != nil {
			t.Fatalf("undefined ratios: %+v", g)
		}
	})
}

func assertOptionalDecimal(t *testing.T, d *stoplore.Decimal, want string) {
	t.Helper()
	if d == nil || d.String() != want {
		t.Fatalf("decimal %v, want %s", d, want)
	}
}

func TestPerformanceMetricsFiltersCurrenciesAndCorrections(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		zero := stoplore.MustDecimal("0")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Metric exchange", EntryFeeRate: &zero, ExitFeeRate: &zero})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Setup", Rules: "First"})
		if err != nil {
			t.Fatal(err)
		}
		version, err := s.ReviseSetup(ctx, stoplore.ReviseSetupRequest{SetupID: setup.ID, Rules: "Second"})
		if err != nil {
			t.Fatal(err)
		}
		// Caller selected March 1 in UTC+03:30. The cohort includes its exact start.
		local := time.FixedZone("caller", 3*3600+30*60)
		start := time.Date(2026, 3, 1, 0, 0, 0, 0, local)
		end := start.AddDate(0, 1, 0)
		create := func(versionID uuid.UUID, currency, symbol string, direction stoplore.TradeDirection, exit string, closeAt time.Time) stoplore.Trade {
			t.Helper()
			tr, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: versionID, Symbol: symbol, Direction: direction, SettlementCurrency: currency, FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("100"), ExecutedAt: start.AddDate(0, -1, 0)}})
			if err != nil {
				t.Fatal(err)
			}
			if exit != "" {
				tr, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: tr.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal(exit), ExecutedAt: closeAt}})
				if err != nil {
					t.Fatal(err)
				}
			}
			return tr
		}
		win := create(setup.Versions[0].ID, "USDT", "BTCUSDT", stoplore.Long, "130", start)
		_, err = s.SetTradeReview(ctx, stoplore.SetTradeReviewRequest{TradeID: win.ID, Tags: []string{"trend", "morning"}, Adherence: stoplore.FollowedRules})
		if err != nil {
			t.Fatal(err)
		}
		create(setup.Versions[0].ID, "USDT", "BTCUSDT", stoplore.Long, "90", start.Add(time.Hour))
		create(setup.Versions[0].ID, "USDT", "BTCUSDT", stoplore.Long, "100", start.Add(2*time.Hour))
		create(setup.Versions[0].ID, "USDT", "BTCUSDT", stoplore.Long, "200", end) // excluded end
		create(version.ID, "USDT", "ETHUSDT", stoplore.Short, "90", start)
		create(setup.Versions[0].ID, "USDC", "BTCUSDC", stoplore.Long, "140", start)
		ongoing := create(setup.Versions[0].ID, "USDT", "BTCUSDT", stoplore.Long, "", time.Time{})
		ongoing, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: ongoing.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.5"), Price: stoplore.MustDecimal("110"), ExecutedAt: start}})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		report, err := s.ReportPerformance(ctx, stoplore.ReportFilter{Start: start, End: end})
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Groups) != 3 || len(report.Ongoing) != 1 || report.Ongoing[0].NetProfitLoss.String() != "5" {
			t.Fatalf("separation: %+v", report)
		}
		var g stoplore.SetupPerformance
		for _, group := range report.Groups {
			if group.SetupVersionID == setup.Versions[0].ID && group.SettlementCurrency == "USDT" {
				g = group
			}
		}
		if g.EligibleCount != 3 || g.Wins != 1 || g.Losses != 1 || g.BreakEven != 1 || g.TotalNetProfit.String() != "20" {
			t.Fatalf("metrics %+v", g)
		}
		assertOptionalDecimal(t, g.WinRate, "0.333333333333333333")
		assertOptionalDecimal(t, g.AverageNetProfit, "6.666666666666666667")
		assertOptionalDecimal(t, g.AverageWin, "30")
		assertOptionalDecimal(t, g.AverageLoss, "-10")
		assertOptionalDecimal(t, g.ProfitFactor, "3")
		followed := stoplore.FollowedRules
		unassessed := stoplore.Unassessed
		cases := []struct {
			name              string
			filter            stoplore.ReportFilter
			eligible, ongoing int
		}{
			{"symbol", stoplore.ReportFilter{Symbol: "ETHUSDT"}, 1, 0},
			{"direction", stoplore.ReportFilter{Direction: stoplore.Short}, 1, 0},
			{"tags", stoplore.ReportFilter{Tags: []string{"trend", "morning"}}, 1, 0},
			{"all tags required", stoplore.ReportFilter{Tags: []string{"trend", "absent"}}, 0, 0},
			{"adherence", stoplore.ReportFilter{Adherence: &followed}, 1, 0},
			{"unassessed", stoplore.ReportFilter{Adherence: &unassessed}, 4, 1},
			{"combined", stoplore.ReportFilter{Symbol: "BTCUSDT", Direction: stoplore.Long, Tags: []string{"trend"}, Adherence: &followed}, 1, 0},
			{"empty", stoplore.ReportFilter{Symbol: "missing"}, 0, 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				f := tc.filter
				f.Start = start
				f.End = end
				r, err := s.ReportPerformance(ctx, f)
				if err != nil {
					t.Fatal(err)
				}
				count := 0
				for _, group := range r.Groups {
					count += group.EligibleCount
				}
				if count != tc.eligible || len(r.Ongoing) != tc.ongoing {
					t.Fatalf("filter %+v", r)
				}
			})
		}
		// A corrected close leaves March and enters April, with its revised net result.
		corrected, err := s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: win.ID, ExecutionID: win.Executions[1].ID, Execution: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("150"), ExecutedAt: end}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.CorrectExecutionFee(ctx, stoplore.CorrectExecutionFeeRequest{TradeID: win.ID, ExecutionID: corrected.Executions[1].ID, ActualFee: decimalPtr("5")})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.ReportPerformance(ctx, stoplore.ReportFilter{Start: start, End: end, Adherence: &followed})
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Groups) != 0 {
			t.Fatalf("corrected trade stayed in cohort: %+v", r)
		}
		r, err = s.ReportPerformance(ctx, stoplore.ReportFilter{Start: end, End: end.AddDate(0, 1, 0), Adherence: &followed})
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Groups) != 1 || r.Groups[0].TotalNetProfit.String() != "45" {
			t.Fatalf("corrected next cohort: %+v", r)
		}
		for _, f := range []stoplore.ReportFilter{{Tags: []string{" "}}, {Start: end, End: start}, {Start: start, End: start}, {Direction: "invalid"}, {Adherence: adherencePtr("invalid")}} {
			if _, err = s.ReportPerformance(ctx, f); !errors.Is(err, stoplore.ErrInvalidRequest) {
				t.Fatalf("invalid filter: %v", err)
			}
		}
	})
}
func decimalPtr(s string) *stoplore.Decimal                         { d := stoplore.MustDecimal(s); return &d }
func adherencePtr(s stoplore.RuleAdherence) *stoplore.RuleAdherence { return &s }

func TestPerformanceClassifiesNetResultsAfterFees(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		zero := stoplore.MustDecimal("0")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Net outcomes", EntryFeeRate: &zero, ExitFeeRate: &zero})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Other setup", Rules: "Rules"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for _, fee := range []string{"10", "15"} {
			tr, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("100"), ExecutedAt: at}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: tr.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("110"), ExecutedAt: at.Add(time.Hour), ActualFee: decimalPtr(fee)}})
			if err != nil {
				t.Fatal(err)
			}
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.ReportPerformance(ctx, stoplore.ReportFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Groups) != 1 {
			t.Fatalf("groups: %+v", r)
		}
		g := r.Groups[0]
		if g.EligibleCount != 2 || g.ExcludedCount != 0 || g.Wins != 0 || g.Losses != 1 || g.BreakEven != 1 || g.TotalNetProfit.String() != "-5" || g.AverageWin != nil {
			t.Fatalf("net classification: %+v", g)
		}
		assertOptionalDecimal(t, g.WinRate, "0")
		assertOptionalDecimal(t, g.AverageLoss, "-5")
		assertOptionalDecimal(t, g.ProfitFactor, "0")
		assertOptionalDecimal(t, g.AverageNetProfit, "-2.5")
		empty, err := s.ReportPerformance(ctx, stoplore.ReportFilter{End: at})
		if err != nil {
			t.Fatal(err)
		}
		if empty.CostBasis != stoplore.RecordedCosts || len(empty.Groups) != 0 || len(empty.Ongoing) != 0 {
			t.Fatalf("empty: %+v", empty)
		}
	})
}
