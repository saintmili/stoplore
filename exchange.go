package mtl

import (
	"context"

	"github.com/google/uuid"
)

type Exchange struct {
	ID           uuid.UUID
	Name         string
	MakerFeeRate *float64
	TakerFeeRate *float64
}

type AddExchangeRequest struct {
	Name         string
	MakerFeeRate *float64
	TakerFeeRate *float64
}
type EditExchangeRequest struct {
	ID           uuid.UUID
	Name         string
	MakerFeeRate *float64
	TakerFeeRate *float64
}

type ExchangeService struct {
	storage StorageContract
}

func (e ExchangeService) Add(ctx context.Context, req AddExchangeRequest) (Exchange, error) {
	return Exchange{}, nil
}

func (e ExchangeService) Edit(ctx context.Context, req EditExchangeRequest) (Exchange, error) {
	return Exchange{}, nil
}

func (e ExchangeService) DeleteByName(ctx context.Context, name string) (Exchange, error) {
	return Exchange{}, nil
}

func (e ExchangeService) DeleteByID(ctx context.Context, id string) (Exchange, error) {
	return Exchange{}, nil
}

func (e ExchangeService) GetByName(ctx context.Context, name string) (Exchange, error) {
	return Exchange{}, nil
}

func (e ExchangeService) GetByID(ctx context.Context, id string) (Exchange, error) {
	return Exchange{}, nil
}

func (e ExchangeService) List(ctx context.Context) ([]Exchange, error) {
	return []Exchange{}, nil
}

var _ ExchangeContract = (*ExchangeService)(nil)
