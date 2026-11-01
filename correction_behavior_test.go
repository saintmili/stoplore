package stoplore_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	stoplore "github.com/saintmili/stoplore"
	"reflect"
	"testing"
	"time"
)

func correctionTrade(t *testing.T, b backend) (*stoplore.Service, stoplore.Trade, time.Time) {
	t.Helper()
	ctx := context.Background()
	s, err := b.Open()
	if err != nil {
		t.Fatal(err)
	}
	rate := stoplore.MustDecimal("1")
	ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Corrections", EntryFeeRate: &rate, ExitFeeRate: &rate})
	if err != nil {
		t.Fatal(err)
	}
	account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Setup", Rules: "Rules"})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC)
	trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("2"), Price: stoplore.MustDecimal("100"), ExecutedAt: at}})
	if err != nil {
		t.Fatal(err)
	}
	trade, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: trade.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("130"), ExecutedAt: at.Add(time.Hour)}})
	if err != nil {
		t.Fatal(err)
	}
	return s, trade, at
}

func TestExecutionCorrectionRecalculatesAndRetainsHistory(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		ctx := context.Background()
		previous := trade.Executions[0]
		trade, err := s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: previous.ID, Execution: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("110"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if trade.RemainingQuantity.String() != "0" || trade.AverageEntryPrice.String() != "110" || trade.RealizedPriceProfitLoss.String() != "20" || trade.TotalExecutionFees.String() != "2.4" || trade.NetProfitLoss.String() != "17.6" || trade.CompletedAt == nil || !trade.CompletedAt.Equal(at.Add(time.Hour)) {
			t.Fatalf("corrected trade: %+v", trade)
		}
		if len(trade.ExecutionCorrections) != 1 || trade.ExecutionCorrections[0].Previous.Quantity.String() != "2" || trade.ExecutionCorrections[0].Previous.Price.String() != "100" || trade.ExecutionCorrections[0].Previous.Fee.Amount.String() != "2" || trade.ExecutionCorrections[0].CorrectedAt.IsZero() || len(trade.Executions) != 2 {
			t.Fatalf("history: %+v", trade.ExecutionCorrections)
		}
	})
}

func TestFeeAndFundingCorrectionsRetainPreviousRecordedCosts(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		ctx := context.Background()
		rebate := stoplore.MustDecimal("-0.5")
		var err error
		trade, err = s.CorrectExecutionFee(ctx, stoplore.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, ActualFee: &rebate})
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.RecordFundingPayment(ctx, stoplore.RecordFundingPaymentRequest{TradeID: trade.ID, Amount: stoplore.MustDecimal("-5"), PaidAt: at})
		if err != nil {
			t.Fatal(err)
		}
		payment := trade.FundingPayments[0]
		trade, err = s.CorrectFundingPayment(ctx, stoplore.CorrectFundingPaymentRequest{TradeID: trade.ID, FundingPaymentID: payment.ID, Amount: stoplore.MustDecimal("1"), PaidAt: at.Add(time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if trade.NetProfitLoss.String() != "30.2" || trade.TotalExecutionFees.String() != "0.8" || trade.RecordedFunding.String() != "1" {
			t.Fatalf("corrected costs: %+v", trade)
		}
		fee := trade.Executions[0].Fee
		if fee.DefaultAmount.String() != "2" || fee.Rate.String() != "1" || fee.ActualAmount.String() != "-0.5" || trade.Executions[1].Fee.Amount.String() != "1.3" {
			t.Fatalf("historical fees: %+v", trade.Executions)
		}
		if len(trade.ExecutionCorrections) != 1 || trade.ExecutionCorrections[0].Previous.Fee.Amount.String() != "2" || len(trade.FundingCorrections) != 1 || trade.FundingCorrections[0].Previous.Amount.String() != "-5" || !trade.FundingCorrections[0].Previous.PaidAt.Equal(at) || trade.FundingCorrections[0].CorrectedAt.IsZero() || !trade.FundingPayments[0].PaidAt.Equal(at.Add(time.Minute)) {
			t.Fatalf("cost history: %+v", trade)
		}
		trade, err = s.CorrectExecutionFee(ctx, stoplore.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID})
		if err != nil {
			t.Fatal(err)
		}
		if trade.NetProfitLoss.String() != "27.7" || trade.Executions[0].Fee.ActualAmount != nil || len(trade.ExecutionCorrections) != 2 {
			t.Fatalf("restored default: %+v", trade)
		}
	})
}

func TestExecutionTimeCorrectionReordersPositionAndFinalClosingInstant(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		ctx := context.Background()
		fill := stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("2"), Price: stoplore.MustDecimal("120"), ExecutedAt: at.Add(2 * time.Hour)}
		trade, err := s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: trade.ID, Entry: fill})
		if err != nil {
			t.Fatal(err)
		}
		addedID := trade.Executions[2].ID
		fill.ExecutedAt = at.Add(30 * time.Minute)
		trade, err = s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: addedID, Execution: fill})
		if err != nil {
			t.Fatal(err)
		}
		if trade.Executions[1].ID != addedID || trade.RemainingQuantity.String() != "3" || trade.AverageEntryPrice.String() != "110" || trade.RealizedPriceProfitLoss.String() != "20" || trade.NetProfitLoss.String() != "14.3" {
			t.Fatalf("reordered: %+v", trade)
		}
		exit := stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("3"), Price: stoplore.MustDecimal("140"), ExecutedAt: at.Add(3 * time.Hour)}
		trade, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: trade.ID, Exit: exit})
		if err != nil {
			t.Fatal(err)
		}
		exit.ExecutedAt = at.AddDate(0, 2, 0)
		trade, err = s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[3].ID, Execution: exit})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if trade.CompletedAt == nil || !trade.CompletedAt.Equal(exit.ExecutedAt) || trade.NetProfitLoss.String() != "100.1" || len(trade.ExecutionCorrections) != 2 || !trade.ExecutionCorrections[0].Previous.ExecutedAt.Equal(at.Add(2*time.Hour)) {
			t.Fatalf("corrected close: %+v", trade)
		}
	})
}

func TestInvalidCorrectionsAreAtomic(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		ctx := context.Background()
		original, err := s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range []struct {
			name string
			id   uuid.UUID
			fill stoplore.ExecutionRequest
			want error
		}{
			{"exit before entry", trade.Executions[1].ID, stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("130"), ExecutedAt: at.Add(-time.Hour)}, stoplore.ErrInvalidRequest},
			{"overexit", trade.Executions[1].ID, stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("3"), Price: stoplore.MustDecimal("130"), ExecutedAt: at.Add(time.Hour)}, stoplore.ErrInvalidRequest},
			{"zero quantity", trade.Executions[0].ID, stoplore.ExecutionRequest{Price: stoplore.MustDecimal("100"), ExecutedAt: at}, stoplore.ErrInvalidRequest},
			{"missing execution", uuid.New(), stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("100"), ExecutedAt: at}, stoplore.ErrNotFound},
		} {
			t.Run(test.name, func(t *testing.T) {
				if _, err := s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: test.id, Execution: test.fill}); !errors.Is(err, test.want) {
					t.Fatalf("correction error: %v", err)
				}
				reopened, err := b.Open()
				if err != nil {
					t.Fatal(err)
				}
				got, err := reopened.GetTrade(ctx, trade.ID)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, original) {
					t.Fatalf("rejected correction changed trade: %+v", got)
				}
			})
		}
		if _, err := s.CorrectExecutionFee(ctx, stoplore.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: uuid.New()}); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing fee execution: %v", err)
		}
		if _, err := s.CorrectFundingPayment(ctx, stoplore.CorrectFundingPaymentRequest{TradeID: trade.ID, FundingPaymentID: uuid.New(), PaidAt: at}); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing funding: %v", err)
		}
		if _, err := s.CorrectFundingPayment(ctx, stoplore.CorrectFundingPaymentRequest{TradeID: trade.ID}); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("missing funding time: %v", err)
		}
		if _, err := s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: uuid.New(), Execution: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("100"), ExecutedAt: at}}); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing trade: %v", err)
		}
	})
}

func TestCorrectionRetainsExactFirstEntryPrice(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		trade, err := s.CorrectExecution(context.Background(), stoplore.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, Execution: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("2"), Price: stoplore.MustDecimal("100.123456789012345678901"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		if trade.AverageEntryPrice.String() != "100.123456789012345678901" {
			t.Fatalf("exact average: %s", trade.AverageEntryPrice)
		}
	})
}

func TestFillCorrectionUsesHistoricalRateAndRetainsActualOverride(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		ctx := context.Background()
		rebate := stoplore.MustDecimal("-0.5")
		trade, err := s.CorrectExecutionFee(ctx, stoplore.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, ActualFee: &rebate})
		if err != nil {
			t.Fatal(err)
		}
		ex, err := s.Exchange.GetByName(ctx, "Corrections")
		if err != nil {
			t.Fatal(err)
		}
		rate := stoplore.MustDecimal("9")
		if _, err = s.Exchange.Edit(ctx, stoplore.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate, ExitFeeRate: &rate}); err != nil {
			t.Fatal(err)
		}
		trade, err = s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, Execution: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("110"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		if trade.NetProfitLoss.String() != "19.2" || trade.CompletedAt == nil || trade.Executions[0].Fee.DefaultAmount.String() != "1.1" || trade.Executions[0].Fee.ActualAmount.String() != "-0.5" || trade.Executions[0].Fee.Rate.String() != "1" || trade.Executions[1].Fee.DefaultAmount.String() != "1.3" {
			t.Fatalf("historical defaults and override: %+v", trade)
		}
		trade, err = s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, Execution: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("2"), Price: stoplore.MustDecimal("110"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if trade.CompletedAt != nil || trade.RemainingQuantity.String() != "1" || len(trade.ExecutionCorrections) != 3 || trade.ExecutionCorrections[2].Previous.Quantity.String() != "1" || trade.ExecutionCorrections[1].Previous.Fee.DefaultAmount.String() != "2" {
			t.Fatalf("reopened position and histories: %+v", trade)
		}
	})
}

func TestCannotRemoveActualFeeWhenHistoricalDefaultIsUnknown(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		ctx := context.Background()
		ex, err := s.Exchange.GetByName(ctx, "Corrections")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Exchange.Edit(ctx, stoplore.EditExchangeRequest{ID: ex.ID, Name: ex.Name}); err != nil {
			t.Fatal(err)
		}
		zero := stoplore.MustDecimal("0")
		trade, err = s.RecordEntry(ctx, stoplore.RecordEntryRequest{TradeID: trade.ID, Entry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("90"), ExecutedAt: at.Add(2 * time.Hour), ActualFee: &zero}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.CorrectExecutionFee(ctx, stoplore.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: trade.Executions[2].ID}); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("missing default: %v", err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(trade.ExecutionCorrections) != 0 || trade.Executions[2].Fee.ActualAmount == nil || trade.Executions[2].Fee.ActualAmount.String() != "0" || trade.Executions[2].Fee.DefaultAmount != nil {
			t.Fatalf("failed reset changed unknown default: %+v", trade)
		}
	})
}

func TestShortExecutionCorrectionRecalculatesDirectionalResult(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, reference, at := correctionTrade(t, b)
		ctx := context.Background()
		trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: reference.AccountID, SetupVersionID: reference.SetupVersionID, Symbol: "BTCUSDT", Direction: stoplore.Short, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("2"), Price: stoplore.MustDecimal("100"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.RecordExit(ctx, stoplore.RecordExitRequest{TradeID: trade.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("130"), ExecutedAt: at.Add(time.Hour)}})
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.CorrectExecution(ctx, stoplore.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, Execution: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("110"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if trade.RealizedPriceProfitLoss.String() != "-20" || trade.NetProfitLoss.String() != "-22.4" || trade.CompletedAt == nil {
			t.Fatalf("short correction: %+v", trade)
		}
	})
}

func TestTradeContextCorrectionPreservesHistoryFeesAndReports(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		ctx := context.Background()
		trade, err := s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: trade.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("130"), ExecutedAt: at.Add(2 * time.Hour)}})
		if err != nil {
			t.Fatal(err)
		}
		previous, err := s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		rate := stoplore.MustDecimal("9")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Correct venue", EntryFeeRate: &rate, ExitFeeRate: &rate})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, stoplore.CreateAccountRequest{ExchangeID: ex.ID, Name: "Correct account"})
		if err != nil {
			t.Fatal(err)
		}
		setup, err := s.CreateSetup(ctx, stoplore.CreateSetupRequest{Name: "Correct setup", Rules: "Correct rules"})
		if err != nil {
			t.Fatal(err)
		}
		req := stoplore.CorrectTradeContextRequest{TradeID: trade.ID, AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: " ETHUSDC ", Direction: stoplore.Short, SettlementCurrency: "USDC"}
		trade, err = s.CorrectTradeContext(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if trade.AccountID != account.ID || trade.SetupVersionID != setup.Versions[0].ID || trade.Symbol != "ETHUSDC" || trade.Direction != stoplore.Short || trade.SettlementCurrency != "USDC" || trade.RealizedPriceProfitLoss.String() != "-60" || trade.NetProfitLoss.String() != "-64.6" {
			t.Fatalf("corrected context: %+v", trade)
		}
		if !reflect.DeepEqual(trade.Executions, previous.Executions) || len(trade.ContextCorrections) != 1 || trade.ContextCorrections[0].Previous.AccountID != previous.AccountID || trade.ContextCorrections[0].Previous.SetupVersionID != previous.SetupVersionID || trade.ContextCorrections[0].Previous.Symbol != previous.Symbol || trade.ContextCorrections[0].Previous.Direction != previous.Direction || trade.ContextCorrections[0].Previous.SettlementCurrency != previous.SettlementCurrency || trade.ContextCorrections[0].CorrectedAt.IsZero() {
			t.Fatalf("history and snapshots: %+v", trade)
		}
		report, err := s.ReportPerformance(ctx, stoplore.ReportFilter{Symbol: "ETHUSDC", Direction: stoplore.Short})
		if err != nil || len(report.Groups) != 1 || report.Groups[0].SetupName != "Correct setup" || report.Groups[0].SettlementCurrency != "USDC" || report.Groups[0].TotalNetProfit.String() != "-64.6" {
			t.Fatalf("report: %+v %v", report, err)
		}
		for _, mutate := range []func(*stoplore.CorrectTradeContextRequest){func(r *stoplore.CorrectTradeContextRequest) { r.AccountID = uuid.New() }, func(r *stoplore.CorrectTradeContextRequest) { r.SetupVersionID = uuid.New() }, func(r *stoplore.CorrectTradeContextRequest) { r.Symbol = " " }, func(r *stoplore.CorrectTradeContextRequest) { r.Direction = "SIDEWAYS" }, func(r *stoplore.CorrectTradeContextRequest) { r.SettlementCurrency = "BTC" }} {
			invalid := req
			mutate(&invalid)
			if _, err := s.CorrectTradeContext(ctx, invalid); err == nil {
				t.Fatal("invalid context accepted")
			}
			got, err := s.GetTrade(ctx, trade.ID)
			if err != nil || !reflect.DeepEqual(got, trade) {
				t.Fatalf("invalid correction changed trade: %+v %v", got, err)
			}
		}
	})
}
