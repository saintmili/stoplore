package mtl

import "errors"

var (
	ErrNilOpenPositionRequest  = errors.New("open position request is required")
	ErrNilClosePositionRequest = errors.New("close position request is required")
	ErrPositionNotFound        = errors.New("position not found")
	ErrExchangeNotFound        = errors.New("exchange not found")
	ErrPositionAlreadyClosed   = errors.New("position already closed")
)
