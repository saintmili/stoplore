package mtl

import (
	"context"

	"github.com/google/uuid"
)

type ServiceContract interface {
	Init() error
}

type StorageContract interface {
	Init(ctx context.Context) error
	CreatePosition(ctx context.Context, position Position) (*Position, error)
	GetPosition(ctx context.Context, positionID uuid.UUID) (*Position, error)
	ListPositions(ctx context.Context) ([]*Position, error)
	EditPosition(ctx context.Context, position Position) (*Position, error)
}

type JournalContract interface {
	OpenPosition(ctx context.Context, request *OpenPositionRequest) (*Position, error)
	ClosePosition(ctx context.Context, request *ClosePositionRequest) (*Position, error)
	GetPosition(ctx context.Context, positionID uuid.UUID) (*Position, error)
	ListPositions(ctx context.Context) ([]*Position, error)
}

type ExchangeContract interface {
	Add(context.Context, AddExchangeRequest) (Exchange, error)
	Edit(context.Context, EditExchangeRequest) (Exchange, error)
	DeleteByName(context.Context, string) (Exchange, error)
	DeleteByID(context.Context, string) (Exchange, error)
	List(context.Context) ([]Exchange, error)
	GetByName(context.Context, string) (Exchange, error)
	GetByID(context.Context, string) (Exchange, error)
}
