package mtl

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FundingPayment records a signed settlement-currency payment: positive is
// received and negative is paid. Missing payments are not evidence of no funding.
type FundingPayment struct {
	ID     uuid.UUID
	Amount Decimal
	PaidAt time.Time
}

type RecordFundingPaymentRequest struct {
	TradeID uuid.UUID
	Amount  Decimal
	PaidAt  time.Time
}

func (s *Service) RecordFundingPayment(ctx context.Context, req RecordFundingPaymentRequest) (Trade, error) {
	if req.PaidAt.IsZero() {
		return Trade{}, fmt.Errorf("%w: funding payment timestamp is required", ErrInvalidRequest)
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
		trade.FundingPayments = append(trade.FundingPayments, FundingPayment{ID: uuid.New(), Amount: req.Amount, PaidAt: req.PaidAt})
		trade.RecordedFunding = trade.RecordedFunding.Add(req.Amount)
		trade.NetProfitLoss = trade.RealizedPriceProfitLoss.Sub(trade.TotalExecutionFees).Add(trade.RecordedFunding)
		trade.CostBasis = RecordedCosts
		trades[trade.ID.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return trade, nil
}
