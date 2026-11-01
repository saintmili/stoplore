package mtl

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	documents documentStore
	storage   StorageContract
	ctx       context.Context
	Journal   *Journal
	Exchange  ExchangeService
}

func NewWithPostgres(db *pgxpool.Pool, key []byte) *Service {
	storage := newPostgresStorage(db, key)
	j := newJournal(storage)
	ctx := context.Background()
	s := &Service{
		storage: storage,
		ctx:     ctx,
		Journal: j,
	}
	s.documents = &postgresDocuments{db: db}
	s.Exchange = ExchangeService{service: s}
	return s
}

func NewWithCSV(path string) *Service {
	storage := newCSVStorage(path)
	j := newJournal(storage)
	ctx := context.Background()
	s := &Service{
		storage: storage,
		ctx:     ctx,
		Journal: j,
	}
	s.documents = &csvDocuments{dir: path}
	s.Exchange = ExchangeService{service: s}
	return s
}

func (s *Service) Init() error {
	if err := s.documents.init(s.ctx); err != nil {
		return err
	}
	return s.storage.Init(s.ctx)
}

var _ ServiceContract = (*Service)(nil)
