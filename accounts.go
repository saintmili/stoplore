package mtl

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

type Account struct {
	ID         uuid.UUID
	ExchangeID uuid.UUID
	Name       string
}
type CreateAccountRequest struct {
	ExchangeID uuid.UUID
	Name       string
}

func (s *Service) CreateAccount(ctx context.Context, req CreateAccountRequest) (Account, error) {
	a := Account{ID: uuid.New(), ExchangeID: req.ExchangeID, Name: strings.TrimSpace(req.Name)}
	if a.Name == "" {
		return Account{}, fmt.Errorf("%w: account name is required", ErrInvalidRequest)
	}
	err := s.update(ctx, func(st *journalState) error {
		if _, ok := st.Exchanges[a.ExchangeID.String()]; !ok {
			return fmt.Errorf("%w: exchange %s", ErrNotFound, a.ExchangeID)
		}
		for _, existing := range st.Accounts {
			if existing.ExchangeID == a.ExchangeID && strings.EqualFold(existing.Name, a.Name) {
				return fmt.Errorf("%w: account name already exists at exchange", ErrConflict)
			}
		}
		st.Accounts[a.ID.String()] = a
		return nil
	})
	if err != nil {
		return Account{}, err
	}
	return a, nil
}
func (s *Service) GetAccount(ctx context.Context, id uuid.UUID) (Account, error) {
	st, err := s.read(ctx)
	if err != nil {
		return Account{}, err
	}
	a, ok := st.Accounts[id.String()]
	if !ok {
		return Account{}, fmt.Errorf("%w: account %s", ErrNotFound, id)
	}
	return a, nil
}
func (s *Service) ListAccounts(ctx context.Context, exchangeID uuid.UUID) ([]Account, error) {
	st, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := st.Exchanges[exchangeID.String()]; !ok {
		return nil, fmt.Errorf("%w: exchange %s", ErrNotFound, exchangeID)
	}
	out := []Account{}
	for _, a := range st.Accounts {
		if a.ExchangeID == exchangeID {
			out = append(out, a)
		}
	}
	return out, nil
}
