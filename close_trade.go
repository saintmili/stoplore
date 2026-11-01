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
		if !req.Exit.Quantity.Equal(trade.RemainingQuantity) {
			return fmt.Errorf("%w: complete exit quantity must equal remaining quantity", ErrInvalidRequest)
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
		exit := req.Exit
		fee, err := executionFee(exit, exchange.ExitFeeRate)
		if err != nil {
			return err
		}
		pnl := exit.Price.Sub(trade.AverageEntryPrice).Mul(exit.Quantity)
		if trade.Direction == Short {
			pnl = pnl.Neg()
		}
		trade.Executions = append(trade.Executions, Execution{ID: uuid.New(), Kind: Exit, Quantity: exit.Quantity, Price: exit.Price, ExecutedAt: exit.ExecutedAt, Margin: exit.Margin, Leverage: exit.Leverage, Fee: fee})
		trade.RealizedPriceProfitLoss = trade.RealizedPriceProfitLoss.Add(pnl)
		trade.TotalExecutionFees = trade.TotalExecutionFees.Add(fee.Amount)
		trade.NetProfitLoss = trade.RealizedPriceProfitLoss.Sub(trade.TotalExecutionFees).Add(trade.RecordedFunding)
		trade.CostBasis = RecordedCosts
		trade.RemainingQuantity = Decimal{}
		trade.CompletedAt = &exit.ExecutedAt
		trades[trade.ID.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return trade, nil
}
