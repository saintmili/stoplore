package stoplore

import (
	"fmt"
)

type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

func (s Side) IsValid() bool {
	switch s {
	case SideBuy, SideSell:
		return true
	default:
		return false
	}
}
func (s Side) Validate() error {
	if s == "" {
		return fmt.Errorf("side is required")
	}
	if !s.IsValid() {
		return fmt.Errorf("invalid side: %s", s)
	}
	return nil
}

func (r *Position) GetOrderValue() float64 {
	return r.PositionMargin * float64(r.Leverage)
}

func (r *Position) GetEntryFeeAmount() float64 {
	return r.GetOrderValue() * r.EntryFeeRate
}

func (r *Position) GetExitFeeAmount() float64 {
	if r.ExitFeeRate == nil {
		return 0
	}
	return r.GetOrderValue() * *r.ExitFeeRate
}
func (r *Position) GetTotalFeesAmount() float64 {
	return r.GetEntryFeeAmount() + r.GetExitFeeAmount()
}

func (r *Position) GetQuantity() float64 {
	if r.EntryPrice <= 0 {
		return 0
	}
	return r.GetOrderValue() / r.EntryPrice
}

func (r *Position) GetExitType() string {
	if r.ExitPrice == nil {
		return ""
	}

	switch r.Side {
	case SideBuy:
		if *r.ExitPrice > r.EntryPrice {
			return "TP"
		}
		return "SL"
	case SideSell:
		if *r.ExitPrice > r.EntryPrice {
			return "SL"
		}
		return "TP"
	default:
		return ""
	}
}

func (r *Position) GetProfitLoss() float64 {
	if r.ExitPrice == nil {
		return 0
	}

	switch r.Side {
	case SideBuy:
		return (*r.ExitPrice - r.EntryPrice) * r.GetQuantity()
	case SideSell:
		return (r.EntryPrice - *r.ExitPrice) * r.GetQuantity()
	default:
		return 0
	}
}
