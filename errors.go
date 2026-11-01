package stoplore

import "errors"

var (
	ErrNotFound                = errors.New("record not found")
	ErrInvalidRequest          = errors.New("invalid request")
	ErrConflict                = errors.New("conflicting record")
	ErrUnavailable             = errors.New("operation unavailable")
	ErrNilOpenPositionRequest  = errors.New("open position request is required")
	ErrNilClosePositionRequest = errors.New("close position request is required")
	ErrPositionNotFound        = errors.New("position not found")
	ErrExchangeNotFound        = errors.New("exchange not found")
	ErrPositionAlreadyClosed   = errors.New("position already closed")
)
