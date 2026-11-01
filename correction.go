package stoplore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ExecutionCorrection struct {
	Previous    Execution
	CorrectedAt time.Time
}

type CorrectExecutionRequest struct {
	TradeID     uuid.UUID
	ExecutionID uuid.UUID
	Execution   ExecutionRequest
}

// CorrectExecution replaces a fill while preserving its identity and kind.
// A nil ActualFee retains its existing override; CorrectExecutionFee can remove it.
func (s *Service) CorrectExecution(ctx context.Context, req CorrectExecutionRequest) (Trade, error) {
	if err := validateExecution(req.Execution); err != nil {
		return Trade{}, err
	}
	return s.correctTrade(ctx, req.TradeID, func(trade *Trade) error {
		for i, previous := range trade.Executions {
			if previous.ID != req.ExecutionID {
				continue
			}
			fill := req.Execution
			if fill.ActualFee == nil {
				fill.ActualFee = previous.Fee.ActualAmount
			}
			fee, err := executionFee(fill, previous.Fee.Rate)
			if err != nil {
				return err
			}
			trade.Executions[i] = Execution{ID: previous.ID, Kind: previous.Kind, Quantity: fill.Quantity, Price: fill.Price, ExecutedAt: fill.ExecutedAt, Margin: fill.Margin, Leverage: fill.Leverage, Fee: fee}
			trade.ExecutionCorrections = append(trade.ExecutionCorrections, ExecutionCorrection{Previous: previous, CorrectedAt: time.Now().UTC()})
			return replayTrade(trade)
		}
		return fmt.Errorf("%w: execution %s", ErrNotFound, req.ExecutionID)
	})
}

func (s *Service) correctTrade(ctx context.Context, id uuid.UUID, correct func(*Trade) error) (Trade, error) {
	var trade Trade
	err := s.update(ctx, func(state *journalState) error {
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		var ok bool
		trade, ok = trades[id.String()]
		if !ok {
			return fmt.Errorf("%w: trade %s", ErrNotFound, id)
		}
		if err := correct(&trade); err != nil {
			return err
		}
		trades[id.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return trade, nil
}

func replayTrade(trade *Trade) error {
	// Stable ordering keeps the recorded sequence for equal execution instants.
	sort.SliceStable(trade.Executions, func(i, j int) bool { return trade.Executions[i].ExecutedAt.Before(trade.Executions[j].ExecutedAt) })
	trade.RemainingQuantity = Decimal{}
	trade.AverageEntryPrice = Decimal{}
	trade.RealizedPriceProfitLoss = Decimal{}
	trade.TotalExecutionFees = Decimal{}
	trade.RecordedFunding = Decimal{}
	trade.CompletedAt = nil
	for i, fill := range trade.Executions {
		if trade.CompletedAt != nil {
			return fmt.Errorf("%w: execution %s follows a fully closed position", ErrInvalidRequest, fill.ID)
		}
		switch fill.Kind {
		case Entry:
			quantity := trade.RemainingQuantity.Add(fill.Quantity)
			if trade.RemainingQuantity.IsZero() {
				trade.AverageEntryPrice = fill.Price
			} else {
				trade.AverageEntryPrice = trade.AverageEntryPrice.Mul(trade.RemainingQuantity).Add(fill.Price.Mul(fill.Quantity)).DivRound(quantity, 18)
			}
			trade.RemainingQuantity = quantity
		case Exit:
			if i == 0 || fill.Quantity.GreaterThan(trade.RemainingQuantity) {
				return fmt.Errorf("%w: exit %s exceeds available quantity at its timestamp", ErrInvalidRequest, fill.ID)
			}
			pnl := fill.Price.Sub(trade.AverageEntryPrice).Mul(fill.Quantity)
			if trade.Direction == Short {
				pnl = pnl.Neg()
			}
			trade.RealizedPriceProfitLoss = trade.RealizedPriceProfitLoss.Add(pnl)
			trade.RemainingQuantity = trade.RemainingQuantity.Sub(fill.Quantity)
			if trade.RemainingQuantity.IsZero() {
				at := fill.ExecutedAt
				trade.CompletedAt = &at
			}
		default:
			return fmt.Errorf("%w: unknown execution kind %q", ErrInvalidRequest, fill.Kind)
		}
		trade.TotalExecutionFees = trade.TotalExecutionFees.Add(fill.Fee.Amount)
	}
	for _, payment := range trade.FundingPayments {
		trade.RecordedFunding = trade.RecordedFunding.Add(payment.Amount)
	}
	trade.NetProfitLoss = trade.RealizedPriceProfitLoss.Sub(trade.TotalExecutionFees).Add(trade.RecordedFunding)
	trade.CostBasis = RecordedCosts
	return nil
}

type CorrectExecutionFeeRequest struct {
	TradeID     uuid.UUID
	ExecutionID uuid.UUID
	ActualFee   *Decimal
}

// CorrectExecutionFee replaces the actual fee. Nil restores the recorded default.
func (s *Service) CorrectExecutionFee(ctx context.Context, req CorrectExecutionFeeRequest) (Trade, error) {
	return s.correctTrade(ctx, req.TradeID, func(trade *Trade) error {
		for i, previous := range trade.Executions {
			if previous.ID != req.ExecutionID {
				continue
			}
			fee := previous.Fee
			if req.ActualFee == nil && fee.DefaultAmount == nil {
				return fmt.Errorf("%w: execution has no recorded default fee", ErrInvalidRequest)
			}
			fee.ActualAmount = req.ActualFee
			if req.ActualFee != nil {
				fee.Amount = *req.ActualFee
			} else {
				fee.Amount = *fee.DefaultAmount
			}
			trade.Executions[i].Fee = fee
			trade.ExecutionCorrections = append(trade.ExecutionCorrections, ExecutionCorrection{Previous: previous, CorrectedAt: time.Now().UTC()})
			return replayTrade(trade)
		}
		return fmt.Errorf("%w: execution %s", ErrNotFound, req.ExecutionID)
	})
}

type FundingCorrection struct {
	Previous    FundingPayment
	CorrectedAt time.Time
}

type CorrectFundingPaymentRequest struct {
	TradeID          uuid.UUID
	FundingPaymentID uuid.UUID
	Amount           Decimal
	PaidAt           time.Time
}

func (s *Service) CorrectFundingPayment(ctx context.Context, req CorrectFundingPaymentRequest) (Trade, error) {
	if req.PaidAt.IsZero() {
		return Trade{}, fmt.Errorf("%w: funding payment timestamp is required", ErrInvalidRequest)
	}
	return s.correctTrade(ctx, req.TradeID, func(trade *Trade) error {
		for i, previous := range trade.FundingPayments {
			if previous.ID != req.FundingPaymentID {
				continue
			}
			trade.FundingPayments[i] = FundingPayment{ID: previous.ID, Amount: req.Amount, PaidAt: req.PaidAt}
			trade.FundingCorrections = append(trade.FundingCorrections, FundingCorrection{Previous: previous, CorrectedAt: time.Now().UTC()})
			return replayTrade(trade)
		}
		return fmt.Errorf("%w: funding payment %s", ErrNotFound, req.FundingPaymentID)
	})
}

// TradeContext is the journal context independent of executions and decisions.
type TradeContext struct {
	AccountID          uuid.UUID
	SetupVersionID     uuid.UUID
	Symbol             string
	Direction          TradeDirection
	SettlementCurrency string
}

type TradeContextCorrection struct {
	Previous    TradeContext
	CorrectedAt time.Time
}

type CorrectTradeContextRequest struct {
	TradeID            uuid.UUID
	AccountID          uuid.UUID
	SetupVersionID     uuid.UUID
	Symbol             string
	Direction          TradeDirection
	SettlementCurrency string
}

func (s *Service) CorrectTradeContext(ctx context.Context, req CorrectTradeContextRequest) (Trade, error) {
	req.Symbol = strings.TrimSpace(req.Symbol)
	if req.AccountID == uuid.Nil || req.SetupVersionID == uuid.Nil || req.Symbol == "" || (req.Direction != Long && req.Direction != Short) || (req.SettlementCurrency != "USDT" && req.SettlementCurrency != "USDC") {
		return Trade{}, fmt.Errorf("%w: valid account, named setup version, symbol, direction and settlement currency are required", ErrInvalidRequest)
	}
	var result Trade
	err := s.update(ctx, func(state *journalState) error {
		if _, ok := state.Accounts[req.AccountID.String()]; !ok {
			return fmt.Errorf("%w: account %s", ErrNotFound, req.AccountID)
		}
		if err := requireNamedSetupVersion(state, req.SetupVersionID); err != nil {
			return err
		}
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		trade, ok := trades[req.TradeID.String()]
		if !ok {
			return fmt.Errorf("%w: trade %s", ErrNotFound, req.TradeID)
		}
		previous := TradeContext{AccountID: trade.AccountID, SetupVersionID: trade.SetupVersionID, Symbol: trade.Symbol, Direction: trade.Direction, SettlementCurrency: trade.SettlementCurrency}
		trade.AccountID, trade.SetupVersionID, trade.Symbol, trade.Direction, trade.SettlementCurrency = req.AccountID, req.SetupVersionID, req.Symbol, req.Direction, req.SettlementCurrency
		if err := replayTrade(&trade); err != nil {
			return err
		}
		trade.ContextCorrections = append(trade.ContextCorrections, TradeContextCorrection{Previous: previous, CorrectedAt: time.Now().UTC()})
		trades[trade.ID.String()] = trade
		result = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return result, nil
}

func requireNamedSetupVersion(state *journalState, id uuid.UUID) error {
	setups := map[string]Setup{}
	if err := state.get("setups", &setups); err != nil {
		return err
	}
	for _, setup := range setups {
		for _, version := range setup.Versions {
			if version.ID == id && strings.TrimSpace(setup.Name) != "" {
				return nil
			}
		}
	}
	return fmt.Errorf("%w: named setup version %s", ErrNotFound, id)
}
