package mtl

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type RecordEntryRequest struct {
	TradeID uuid.UUID
	Entry   ExecutionRequest
}

func (s *Service) RecordEntry(ctx context.Context, req RecordEntryRequest) (Trade, error) {
	if err := validateExecution(req.Entry); err != nil {
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
		for _, execution := range trade.Executions {
			if req.Entry.ExecutedAt.Before(execution.ExecutedAt) {
				return fmt.Errorf("%w: entry precedes an existing execution", ErrInvalidRequest)
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
		entry := req.Entry
		fee, err := executionFee(entry, exchange.EntryFeeRate)
		if err != nil {
			return err
		}
		trade.Executions = append(trade.Executions, Execution{ID: uuid.New(), Kind: Entry, Quantity: entry.Quantity, Price: entry.Price, ExecutedAt: entry.ExecutedAt, Margin: entry.Margin, Leverage: entry.Leverage, Fee: fee})
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
