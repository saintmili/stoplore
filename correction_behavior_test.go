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

func correctionTrade(t *testing.T, b backend) (*mtl.Service, mtl.Trade, time.Time) {
	t.Helper()
	ctx := context.Background()
	s, err := b.Open()
	if err != nil {
		t.Fatal(err)
	}
	rate := mtl.MustDecimal("1")
	ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Corrections", EntryFeeRate: &rate, ExitFeeRate: &rate})
	if err != nil {
		t.Fatal(err)
	}
	account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	setup, err := s.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Setup", Rules: "Rules"})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC)
	trade, err := s.RecordTrade(ctx, mtl.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: mtl.Long, SettlementCurrency: "USDT", FirstEntry: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("2"), Price: mtl.MustDecimal("100"), ExecutedAt: at}})
	if err != nil {
		t.Fatal(err)
	}
	trade, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: trade.ID, Exit: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("130"), ExecutedAt: at.Add(time.Hour)}})
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
		trade, err := s.CorrectExecution(ctx, mtl.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: previous.ID, Execution: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("110"), ExecutedAt: at}})
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
		rebate := mtl.MustDecimal("-0.5")
		var err error
		trade, err = s.CorrectExecutionFee(ctx, mtl.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, ActualFee: &rebate})
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.RecordFundingPayment(ctx, mtl.RecordFundingPaymentRequest{TradeID: trade.ID, Amount: mtl.MustDecimal("-5"), PaidAt: at})
		if err != nil {
			t.Fatal(err)
		}
		payment := trade.FundingPayments[0]
		trade, err = s.CorrectFundingPayment(ctx, mtl.CorrectFundingPaymentRequest{TradeID: trade.ID, FundingPaymentID: payment.ID, Amount: mtl.MustDecimal("1"), PaidAt: at.Add(time.Minute)})
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
		trade, err = s.CorrectExecutionFee(ctx, mtl.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID})
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
		fill := mtl.ExecutionRequest{Quantity: mtl.MustDecimal("2"), Price: mtl.MustDecimal("120"), ExecutedAt: at.Add(2 * time.Hour)}
		trade, err := s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: trade.ID, Entry: fill})
		if err != nil {
			t.Fatal(err)
		}
		addedID := trade.Executions[2].ID
		fill.ExecutedAt = at.Add(30 * time.Minute)
		trade, err = s.CorrectExecution(ctx, mtl.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: addedID, Execution: fill})
		if err != nil {
			t.Fatal(err)
		}
		if trade.Executions[1].ID != addedID || trade.RemainingQuantity.String() != "3" || trade.AverageEntryPrice.String() != "110" || trade.RealizedPriceProfitLoss.String() != "20" || trade.NetProfitLoss.String() != "14.3" {
			t.Fatalf("reordered: %+v", trade)
		}
		exit := mtl.ExecutionRequest{Quantity: mtl.MustDecimal("3"), Price: mtl.MustDecimal("140"), ExecutedAt: at.Add(3 * time.Hour)}
		trade, err = s.CloseTrade(ctx, mtl.CloseTradeRequest{TradeID: trade.ID, Exit: exit})
		if err != nil {
			t.Fatal(err)
		}
		exit.ExecutedAt = at.AddDate(0, 2, 0)
		trade, err = s.CorrectExecution(ctx, mtl.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[3].ID, Execution: exit})
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
			fill mtl.ExecutionRequest
			want error
		}{
			{"exit before entry", trade.Executions[1].ID, mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("130"), ExecutedAt: at.Add(-time.Hour)}, mtl.ErrInvalidRequest},
			{"overexit", trade.Executions[1].ID, mtl.ExecutionRequest{Quantity: mtl.MustDecimal("3"), Price: mtl.MustDecimal("130"), ExecutedAt: at.Add(time.Hour)}, mtl.ErrInvalidRequest},
			{"zero quantity", trade.Executions[0].ID, mtl.ExecutionRequest{Price: mtl.MustDecimal("100"), ExecutedAt: at}, mtl.ErrInvalidRequest},
			{"missing execution", uuid.New(), mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: at}, mtl.ErrNotFound},
		} {
			t.Run(test.name, func(t *testing.T) {
				if _, err := s.CorrectExecution(ctx, mtl.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: test.id, Execution: test.fill}); !errors.Is(err, test.want) {
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
		if _, err := s.CorrectExecutionFee(ctx, mtl.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: uuid.New()}); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing fee execution: %v", err)
		}
		if _, err := s.CorrectFundingPayment(ctx, mtl.CorrectFundingPaymentRequest{TradeID: trade.ID, FundingPaymentID: uuid.New(), PaidAt: at}); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing funding: %v", err)
		}
		if _, err := s.CorrectFundingPayment(ctx, mtl.CorrectFundingPaymentRequest{TradeID: trade.ID}); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("missing funding time: %v", err)
		}
		if _, err := s.CorrectExecution(ctx, mtl.CorrectExecutionRequest{TradeID: uuid.New(), Execution: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: at}}); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing trade: %v", err)
		}
	})
}

func TestCorrectionRetainsExactFirstEntryPrice(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		s, trade, at := correctionTrade(t, b)
		trade, err := s.CorrectExecution(context.Background(), mtl.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, Execution: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("2"), Price: mtl.MustDecimal("100.123456789012345678901"), ExecutedAt: at}})
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
		rebate := mtl.MustDecimal("-0.5")
		trade, err := s.CorrectExecutionFee(ctx, mtl.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, ActualFee: &rebate})
		if err != nil {
			t.Fatal(err)
		}
		ex, err := s.Exchange.GetByName(ctx, "Corrections")
		if err != nil {
			t.Fatal(err)
		}
		rate := mtl.MustDecimal("9")
		if _, err = s.Exchange.Edit(ctx, mtl.EditExchangeRequest{ID: ex.ID, Name: ex.Name, EntryFeeRate: &rate, ExitFeeRate: &rate}); err != nil {
			t.Fatal(err)
		}
		trade, err = s.CorrectExecution(ctx, mtl.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, Execution: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("110"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		if trade.NetProfitLoss.String() != "19.2" || trade.CompletedAt == nil || trade.Executions[0].Fee.DefaultAmount.String() != "1.1" || trade.Executions[0].Fee.ActualAmount.String() != "-0.5" || trade.Executions[0].Fee.Rate.String() != "1" || trade.Executions[1].Fee.DefaultAmount.String() != "1.3" {
			t.Fatalf("historical defaults and override: %+v", trade)
		}
		trade, err = s.CorrectExecution(ctx, mtl.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, Execution: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("2"), Price: mtl.MustDecimal("110"), ExecutedAt: at}})
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
		if _, err = s.Exchange.Edit(ctx, mtl.EditExchangeRequest{ID: ex.ID, Name: ex.Name}); err != nil {
			t.Fatal(err)
		}
		zero := mtl.MustDecimal("0")
		trade, err = s.RecordEntry(ctx, mtl.RecordEntryRequest{TradeID: trade.ID, Entry: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("90"), ExecutedAt: at.Add(2 * time.Hour), ActualFee: &zero}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.CorrectExecutionFee(ctx, mtl.CorrectExecutionFeeRequest{TradeID: trade.ID, ExecutionID: trade.Executions[2].ID}); !errors.Is(err, mtl.ErrInvalidRequest) {
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
		trade, err := s.RecordTrade(ctx, mtl.RecordTradeRequest{AccountID: reference.AccountID, SetupVersionID: reference.SetupVersionID, Symbol: "BTCUSDT", Direction: mtl.Short, SettlementCurrency: "USDT", FirstEntry: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("2"), Price: mtl.MustDecimal("100"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.RecordExit(ctx, mtl.RecordExitRequest{TradeID: trade.ID, Exit: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("130"), ExecutedAt: at.Add(time.Hour)}})
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.CorrectExecution(ctx, mtl.CorrectExecutionRequest{TradeID: trade.ID, ExecutionID: trade.Executions[0].ID, Execution: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("110"), ExecutedAt: at}})
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
