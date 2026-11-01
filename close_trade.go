package mtl

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type CloseTradeRequest struct {
	TradeID uuid.UUID
	Exit    ExecutionRequest
}

func (s *Service) CloseTrade(ctx context.Context, req CloseTradeRequest) (Trade, error) {
	return s.recordExit(ctx, req, true)
}

type RecordExitRequest struct {
	TradeID uuid.UUID
	Exit    ExecutionRequest
}

func (s *Service) RecordExit(ctx context.Context, req RecordExitRequest) (Trade, error) {
	return s.recordExit(ctx, CloseTradeRequest{TradeID: req.TradeID, Exit: req.Exit}, false)
}

func (s *Service) recordExit(ctx context.Context, req CloseTradeRequest, complete bool) (Trade, error) {
	if err := validateExecution(req.Exit); err != nil {
		return Trade{}, err
	}
	var trade Trade
	err := s.update(ctx, func(state *journalState) error {
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		var ok bool
		trade, ok = trades[req.TradeID.String()]
		if !ok {
			return fmt.Errorf("%w: trade %s", ErrNotFound, req.TradeID)
		}
		if trade.CompletedAt != nil || trade.RemainingQuantity.IsZero() {
			return fmt.Errorf("%w: trade is already completed", ErrInvalidRequest)
		}
		if complete && !req.Exit.Quantity.Equal(trade.RemainingQuantity) {
			return fmt.Errorf("%w: complete exit quantity must equal remaining quantity", ErrInvalidRequest)
		}
		if req.Exit.Quantity.GreaterThan(trade.RemainingQuantity) {
			return fmt.Errorf("%w: exit quantity exceeds remaining quantity", ErrInvalidRequest)
		}
		for _, execution := range trade.Executions {
			if req.Exit.ExecutedAt.Before(execution.ExecutedAt) {
				return fmt.Errorf("%w: exit precedes an existing execution", ErrInvalidRequest)
			}
		}
		account, ok := state.Accounts[trade.AccountID.String()]
		if !ok {
			return fmt.Errorf("%w: account %s", ErrNotFound, trade.AccountID)
		}
		exchange, ok := state.Exchanges[account.ExchangeID.String()]
		if !ok {
			return fmt.Errorf("%w: exchange %s", ErrNotFound, account.ExchangeID)
		}
		if exchange.ExitFeeRate == nil {
			return fmt.Errorf("%w: exit fee rate is unknown for exchange %s", ErrInvalidRequest, exchange.ID)
		}
		exit := req.Exit
		fee := exit.Quantity.Mul(exit.Price).Mul(*exchange.ExitFeeRate).DivRound(MustDecimal("100"), 18)
		pnl := exit.Price.Sub(trade.AverageEntryPrice).Mul(exit.Quantity)
		if trade.Direction == Short {
			pnl = pnl.Neg()
		}
		trade.Executions = append(trade.Executions, Execution{ID: uuid.New(), Kind: Exit, Quantity: exit.Quantity, Price: exit.Price, ExecutedAt: exit.ExecutedAt, Margin: exit.Margin, Leverage: exit.Leverage, Fee: ExecutionFee{Rate: exchange.ExitFeeRate, Amount: fee}})
		trade.RealizedPriceProfitLoss = trade.RealizedPriceProfitLoss.Add(pnl)
		trade.TotalExecutionFees = trade.TotalExecutionFees.Add(fee)
		trade.NetProfitLoss = trade.RealizedPriceProfitLoss.Sub(trade.TotalExecutionFees).Add(trade.RecordedFunding)
		trade.CostBasis = RecordedCosts
		trade.RemainingQuantity = trade.RemainingQuantity.Sub(exit.Quantity)
		if trade.RemainingQuantity.IsZero() {
			trade.CompletedAt = &exit.ExecutedAt
		}
		trades[trade.ID.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return trade, nil
}
