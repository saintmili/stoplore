package mtl

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	storage StorageContract
	ctx     context.Context
	Journal *Journal
	Exchange ExchangeService
}

func NewWithPostgres(db *pgxpool.Pool, key []byte) *Service {
	storage := newPostgresStorage(db, key)
	j := newJournal(storage)
	ctx := context.Background()
	return &Service{
		storage: storage,
		ctx:     ctx,
		Journal: j,
	}
}

func NewWithCSV(path string) *Service {
	storage := newCSVStorage(path)
	j := newJournal(storage)
	ctx := context.Background()
	return &Service{
		storage: storage,
		ctx:     ctx,
		Journal: j,
	}
}

func (s *Service) Init() error {
	return s.storage.Init(s.ctx)
}

var _ ServiceContract = (*Service)(nil)
