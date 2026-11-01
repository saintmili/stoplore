package stoplore

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	orderTypeMarket = "MARKET"
	orderTypeLimit  = "LIMIT"

	entryTypeTrade       = "TRADE"
	entryTypeMarginInOut = "MARGININOUT"
	entryTypeDeposit     = "DEPOSIT"
	entryTypeWithdrawal  = "WITHDRAWAL"
)

type Position struct {
	ID        uuid.UUID
	Exchange  string // BINANCE, BITUNIX
	Strategy  string
	Symbol    string // BTC, ETH
	Side      Side   // BUY, SELL
	OrderType string // LIMIT, MARKET
	EntryType string // TRADE, MARGININOUT, DEPOSIT, WITHDRAWAL
	Leverage  int
	TimeFrame string // 15M,1H,4H,1D

	// Entry
	EntryPrice       float64
	EntryMentalState string // HAPPY, RELAXED, WORRIED
	PositionMargin   float64
	EntryFeeRate     float64
	StopLossPrice    *float64
	TakeProfitPrice  *float64
	EntryImageUrl    string
	OpenedAt         time.Time

	// Exit
	ExitPrice         *float64
	ExitMarginPercent *float64
	ExitFeeRate       *float64
	ExitMentalState   string // HAPPY, RELAXED, WORRIED
	ExitImageUrl      string
	ExitedAt          *time.Time

	Notes     string
	CreatedAt time.Time
	UpdatedAt *time.Time
}

func (r *Position) Validate() error {
	return validatePositionFields(
		r.Side,
		r.Exchange,
		r.Symbol,
		r.EntryPrice,
		r.PositionMargin,
		r.OpenedAt,
		r.OrderType,
		r.EntryType,
		r.Leverage,
		r.TimeFrame,
	)
}

type OpenPositionRequest struct {
	Exchange         string
	Strategy         string
	Symbol           string
	Side             Side
	OrderType        string
	EntryType        string
	Leverage         int
	TimeFrame        string
	EntryPrice       float64
	EntryMentalState string
	PositionMargin   float64
	EntryFeeRate     float64
	StopLossPrice    *float64
	TakeProfitPrice  *float64
	EntryImageUrl    string
	OpenedAt         time.Time
	Notes            string
}

func (r *OpenPositionRequest) Validate() error {
	return validatePositionFields(
		r.Side,
		r.Exchange,
		r.Symbol,
		r.EntryPrice,
		r.PositionMargin,
		r.OpenedAt,
		r.OrderType,
		r.EntryType,
		r.Leverage,
		r.TimeFrame,
	)
}

func (r *OpenPositionRequest) ToPosition(now time.Time) Position {
	if now.IsZero() {
		now = time.Now()
	}

	return Position{
		ID:               uuid.New(),
		Exchange:         r.Exchange,
		Strategy:         r.Strategy,
		Symbol:           r.Symbol,
		Side:             r.Side,
		OrderType:        r.OrderType,
		EntryType:        r.EntryType,
		Leverage:         r.Leverage,
		TimeFrame:        r.TimeFrame,
		EntryPrice:       r.EntryPrice,
		EntryMentalState: r.EntryMentalState,
		PositionMargin:   r.PositionMargin,
		EntryFeeRate:     r.EntryFeeRate,
		StopLossPrice:    r.StopLossPrice,
		TakeProfitPrice:  r.TakeProfitPrice,
		EntryImageUrl:    r.EntryImageUrl,
		OpenedAt:         r.OpenedAt,
		Notes:            r.Notes,
		CreatedAt:        now,
	}
}

type ClosePositionRequest struct {
	ID                uuid.UUID
	ExitPrice         float64
	ExitMarginPercent float64
	ExitFeeRate       float64
	ExitMentalState   string
	ExitImageUrl      string
	ExitedAt          time.Time
	Notes             string
}

func (r *ClosePositionRequest) Validate() error {
	if r.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if r.ExitPrice <= 0 {
		return errors.New("exit price must be > 0")
	}
	if r.ExitedAt.IsZero() {
		return errors.New("exit date is required")
	}
	if r.ExitFeeRate < 0 {
		return errors.New("exit fee rate must be >= 0")
	}

	return nil
}

func validatePositionFields(
	side Side,
	exchange string,
	symbol string,
	entryPrice float64,
	positionMargin float64,
	openedAt time.Time,
	orderType string,
	entryType string,
	leverage int,
	timeFrame string,
) error {
	if err := side.Validate(); err != nil {
		return err
	}
	if exchange == "" {
		return errors.New("exchange is required")
	}
	if symbol == "" {
		return errors.New("symbol is required")
	}
	if entryPrice <= 0 {
		return errors.New("entry price must be > 0")
	}
	if positionMargin <= 0 {
		return errors.New("position margin must be > 0")
	}
	if openedAt.IsZero() {
		return errors.New("opened date is required")
	}
	if !isValidOrderType(orderType) {
		return errors.New("order type must be MARKET or LIMIT")
	}
	if !isValidEntryType(entryType) {
		return errors.New("entry type must be TRADE or MARGININOUT or DEPOSIT or WITHDRAWAL")
	}
	if leverage <= 0 {
		return errors.New("leverage must be > 0")
	}
	if timeFrame == "" {
		return errors.New("timeframe is required")
	}

	return nil
}

func isValidOrderType(orderType string) bool {
	switch orderType {
	case orderTypeMarket, orderTypeLimit:
		return true
	default:
		return false
	}
}

func isValidEntryType(entryType string) bool {
	switch entryType {
	case entryTypeTrade, entryTypeMarginInOut, entryTypeDeposit, entryTypeWithdrawal:
		return true
	default:
		return false
	}
}
