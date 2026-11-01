package stoplore_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"

	stoplore "github.com/saintmili/stoplore"
)

func TestActualFeesReplaceDefaultsAndRetainSnapshots(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := stoplore.MustDecimal("0.04")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Costs", EntryFeeRate: &rate, ExitFeeRate: &rate})
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
		discount := stoplore.MustDecimal("1.2")
		trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("60000"), ExecutedAt: at, ActualFee: &discount}})
		if err != nil {
			t.Fatal(err)
		}
		rebate := stoplore.MustDecimal("-0.5")
		trade, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: trade.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("62000"), ExecutedAt: at.Add(time.Hour), ActualFee: &rebate}})
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
		if trade.TotalExecutionFees.String() != "0.7" || trade.NetProfitLoss.String() != "199.3" {
			t.Fatalf("effective fees: %+v", trade)
		}
		for i, expected := range []struct{ defaultFee, actual string }{{"2.4", "1.2"}, {"2.48", "-0.5"}} {
			fee := trade.Executions[i].Fee
			if fee.Rate == nil || fee.Rate.String() != "0.04" || fee.DefaultAmount == nil || fee.DefaultAmount.String() != expected.defaultFee || fee.ActualAmount == nil || fee.ActualAmount.String() != expected.actual || fee.Amount.String() != expected.actual {
				t.Fatalf("fee snapshot: %+v", fee)
			}
		}
	})
}

func TestFundingPaymentsAdjustRecordedCostResultAndSurviveReopening(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		rate := stoplore.MustDecimal("0.04")
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Funding", EntryFeeRate: &rate, ExitFeeRate: &rate})
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
		trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "BTCUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("60000"), ExecutedAt: at}})
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: trade.ID, Exit: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("0.1"), Price: stoplore.MustDecimal("62000"), ExecutedAt: at.Add(time.Hour)}})
		if err != nil {
			t.Fatal(err)
		}
		if trade.NetProfitLoss.String() != "195.12" || len(trade.FundingPayments) != 0 || trade.CostBasis != stoplore.RecordedCosts {
			t.Fatalf("absent funding: %+v", trade)
		}
		for i, amount := range []string{"1", "-5"} {
			trade, err = s.RecordFundingPayment(ctx, stoplore.RecordFundingPaymentRequest{TradeID: trade.ID, Amount: stoplore.MustDecimal(amount), PaidAt: at.Add(time.Duration(i+1) * time.Minute)})
			if err != nil {
				t.Fatal(err)
			}
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if trade.RecordedFunding.String() != "-4" || trade.NetProfitLoss.String() != "191.12" || trade.TotalExecutionFees.String() != "4.88" || trade.CostBasis != stoplore.RecordedCosts || len(trade.FundingPayments) != 2 {
			t.Fatalf("funded result: %+v", trade)
		}
		for i, amount := range []string{"1", "-5"} {
			payment := trade.FundingPayments[i]
			if payment.Amount.String() != amount || !payment.PaidAt.Equal(at.Add(time.Duration(i+1)*time.Minute)) {
				t.Fatalf("payment: %+v", payment)
			}
		}
	})
}

func TestActualFeeSuppliesMissingDefaultsAndFundingErrorsLeaveTradeUnchanged(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		ex, err := s.Exchange.Add(ctx, stoplore.AddExchangeRequest{Name: "Unknown fees"})
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
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		zero := stoplore.MustDecimal("0")
		trade, err := s.RecordTrade(ctx, stoplore.RecordTradeRequest{AccountID: account.ID, SetupVersionID: setup.Versions[0].ID, Symbol: "ETHUSDT", Direction: stoplore.Long, SettlementCurrency: "USDT", FirstEntry: stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("100"), ExecutedAt: at, ActualFee: &zero}})
		if err != nil {
			t.Fatal(err)
		}
		exit := stoplore.ExecutionRequest{Quantity: stoplore.MustDecimal("1"), Price: stoplore.MustDecimal("110"), ExecutedAt: at.Add(time.Hour)}
		if _, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: trade.ID, Exit: exit}); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("missing fee: %v", err)
		}
		rebate := stoplore.MustDecimal("-0.123456789012345678901")
		exit.ActualFee = &rebate
		trade, err = s.CloseTrade(ctx, stoplore.CloseTradeRequest{TradeID: trade.ID, Exit: exit})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.RecordFundingPayment(ctx, stoplore.RecordFundingPaymentRequest{TradeID: trade.ID, Amount: stoplore.MustDecimal("1")}); !errors.Is(err, stoplore.ErrInvalidRequest) {
			t.Fatalf("missing time: %v", err)
		}
		if _, err = s.RecordFundingPayment(ctx, stoplore.RecordFundingPaymentRequest{TradeID: uuid.New(), PaidAt: at}); !errors.Is(err, stoplore.ErrNotFound) {
			t.Fatalf("missing trade: %v", err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		trade, err = s.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Fatal(err)
		}
		if trade.NetProfitLoss.String() != "10.123456789012345678901" || len(trade.FundingPayments) != 0 {
			t.Fatalf("result: %+v", trade)
		}
		for _, execution := range trade.Executions {
			if execution.Fee.Rate != nil || execution.Fee.DefaultAmount != nil || execution.Fee.ActualAmount == nil {
				t.Fatalf("unknown default lost: %+v", execution.Fee)
			}
		}
	})
}
