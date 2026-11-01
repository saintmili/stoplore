package mtl

import "fmt"

func executionFee(req ExecutionRequest, rate *Decimal) (ExecutionFee, error) {
	fee := ExecutionFee{Rate: rate, ActualAmount: req.ActualFee}
	if rate != nil {
		amount := req.Quantity.Mul(req.Price).Mul(*rate).DivRound(MustDecimal("100"), 18)
		fee.DefaultAmount = &amount
		fee.Amount = amount
	}
	if req.ActualFee != nil {
		fee.Amount = *req.ActualFee
	} else if rate == nil {
		return ExecutionFee{}, fmt.Errorf("%w: execution requires a configured fee rate or actual fee", ErrInvalidRequest)
	}
	return fee, nil
}
