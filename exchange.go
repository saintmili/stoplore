package mtl

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"sort"
	"strings"
)

type Exchange struct {
	ID           uuid.UUID
	Name         string
	EntryFeeRate *Decimal
	ExitFeeRate  *Decimal
	// Legacy maker/taker values are retained for source compatibility; defaults
	// for new executions only use the explicit entry/exit rates.
	MakerFeeRate *float64
	TakerFeeRate *float64
}
type AddExchangeRequest struct {
	Name                       string
	EntryFeeRate, ExitFeeRate  *Decimal
	MakerFeeRate, TakerFeeRate *float64
}
type EditExchangeRequest struct {
	ID                         uuid.UUID
	Name                       string
	EntryFeeRate, ExitFeeRate  *Decimal
	MakerFeeRate, TakerFeeRate *float64
}
type ExchangeService struct{ service *Service }

func validateExchange(e Exchange) error {
	if e.Name == "" {
		return fmt.Errorf("%w: exchange name is required", ErrInvalidRequest)
	}
	for _, rate := range []*Decimal{e.EntryFeeRate, e.ExitFeeRate} {
		if rate != nil && rate.IsNegative() {
			return fmt.Errorf("%w: fee percentage cannot be negative", ErrInvalidRequest)
		}
	}
	return nil
}
func (e ExchangeService) Add(ctx context.Context, req AddExchangeRequest) (Exchange, error) {
	out := Exchange{ID: uuid.New(), Name: strings.TrimSpace(req.Name), EntryFeeRate: req.EntryFeeRate, ExitFeeRate: req.ExitFeeRate, MakerFeeRate: req.MakerFeeRate, TakerFeeRate: req.TakerFeeRate}
	if err := validateExchange(out); err != nil {
		return Exchange{}, err
	}
	err := e.service.update(ctx, func(st *journalState) error {
		for _, other := range st.Exchanges {
			if strings.EqualFold(other.Name, out.Name) {
				return fmt.Errorf("%w: exchange name already exists", ErrConflict)
			}
		}
		st.Exchanges[out.ID.String()] = out
		return nil
	})
	if err != nil {
		return Exchange{}, err
	}
	return out, nil
}
func (e ExchangeService) Edit(ctx context.Context, req EditExchangeRequest) (Exchange, error) {
	out := Exchange{ID: req.ID, Name: strings.TrimSpace(req.Name), EntryFeeRate: req.EntryFeeRate, ExitFeeRate: req.ExitFeeRate, MakerFeeRate: req.MakerFeeRate, TakerFeeRate: req.TakerFeeRate}
	if err := validateExchange(out); err != nil {
		return Exchange{}, err
	}
	err := e.service.update(ctx, func(st *journalState) error {
		if _, ok := st.Exchanges[out.ID.String()]; !ok {
			return fmt.Errorf("%w: exchange %s", ErrNotFound, out.ID)
		}
		for _, other := range st.Exchanges {
			if other.ID != out.ID && strings.EqualFold(other.Name, out.Name) {
				return fmt.Errorf("%w: exchange name already exists", ErrConflict)
			}
		}
		st.Exchanges[out.ID.String()] = out
		return nil
	})
	if err != nil {
		return Exchange{}, err
	}
	return out, nil
}
func (e ExchangeService) GetByID(ctx context.Context, id string) (Exchange, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return Exchange{}, fmt.Errorf("%w: invalid exchange ID", ErrInvalidRequest)
	}
	st, err := e.service.read(ctx)
	if err != nil {
		return Exchange{}, err
	}
	out, ok := st.Exchanges[parsed.String()]
	if !ok {
		return Exchange{}, fmt.Errorf("%w: exchange %s", ErrNotFound, id)
	}
	return out, nil
}
func (e ExchangeService) GetByName(ctx context.Context, name string) (Exchange, error) {
	st, err := e.service.read(ctx)
	if err != nil {
		return Exchange{}, err
	}
	for _, out := range st.Exchanges {
		if strings.EqualFold(out.Name, strings.TrimSpace(name)) {
			return out, nil
		}
	}
	return Exchange{}, fmt.Errorf("%w: exchange %q", ErrNotFound, name)
}
func (e ExchangeService) List(ctx context.Context) ([]Exchange, error) {
	st, err := e.service.read(ctx)
	if err != nil {
		return nil, err
	}
	out := []Exchange{}
	for _, x := range st.Exchanges {
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func (e ExchangeService) DeleteByName(ctx context.Context, name string) (Exchange, error) {
	out, err := e.GetByName(ctx, name)
	if err != nil {
		return Exchange{}, err
	}
	return e.DeleteByID(ctx, out.ID.String())
}
func (e ExchangeService) DeleteByID(ctx context.Context, id string) (Exchange, error) {
	out, err := e.GetByID(ctx, id)
	if err != nil {
		return Exchange{}, err
	}
	err = e.service.update(ctx, func(st *journalState) error {
		if _, ok := st.Exchanges[id]; !ok {
			return fmt.Errorf("%w: exchange %s", ErrNotFound, id)
		}
		for _, a := range st.Accounts {
			if a.ExchangeID == out.ID {
				return fmt.Errorf("%w: exchange has accounts", ErrConflict)
			}
		}
		delete(st.Exchanges, id)
		return nil
	})
	return out, err
}

var _ ExchangeContract = (*ExchangeService)(nil)
