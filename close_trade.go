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
		exit := req.Exit
		fee, err := executionFee(exit, exchange.ExitFeeRate)
		if err != nil {
			return err
		}
		trade.Executions = append(trade.Executions, Execution{ID: uuid.New(), Kind: Exit, Quantity: exit.Quantity, Price: exit.Price, ExecutedAt: exit.ExecutedAt, Margin: exit.Margin, Leverage: exit.Leverage, Fee: fee})
		if err := replayTrade(&trade); err != nil {
			return err
		}
		trades[trade.ID.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return trade, nil
}
