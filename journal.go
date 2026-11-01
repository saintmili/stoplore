package mtl

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Journal struct {
	storage StorageContract
}

type PositionFilter struct {
	Symbol string
	Side   string
	From   time.Time
	To     time.Time
}

func newJournal(storage StorageContract) *Journal {
	return &Journal{
		storage: storage,
	}
}

func (j *Journal) OpenPosition(ctx context.Context, req *OpenPositionRequest) (*Position, error) {
	if req == nil {
		return nil, ErrNilOpenPositionRequest
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	position := req.ToPosition(time.Now())
	createdPosition, err := j.storage.CreatePosition(ctx, position)
	if err != nil {
		return nil, err
	}

	return createdPosition, nil
}

func (j *Journal) ClosePosition(ctx context.Context, req *ClosePositionRequest) (*Position, error) {
	if req == nil {
		return nil, ErrNilClosePositionRequest
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	position, err := j.storage.GetPosition(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if position.ExitedAt != nil || position.ExitPrice != nil {
		return nil, ErrPositionAlreadyClosed
	}

	now := time.Now()
	position.ExitPrice = &req.ExitPrice
	position.ExitMarginPercent = &req.ExitMarginPercent
	position.ExitFeeRate = &req.ExitFeeRate
	position.ExitMentalState = req.ExitMentalState
	position.ExitImageUrl = req.ExitImageUrl
	position.ExitedAt = &req.ExitedAt
	position.UpdatedAt = &now
	if req.Notes != "" {
		position.Notes = req.Notes
	}

	return j.storage.EditPosition(ctx, *position)
}

func (j *Journal) GetPosition(ctx context.Context, id uuid.UUID) (*Position, error) {
	position, err := j.storage.GetPosition(ctx, id)
	if err != nil {
		return nil, err
	}

	return position, nil
}

func (j *Journal) ListPositions(ctx context.Context) ([]*Position, error) {
	positions, err := j.storage.ListPositions(ctx)
	if err != nil {
		return nil, err
	}
	return positions, nil
}

func (j *Journal) GetAnalytics(ctx context.Context) (*Analytics, error) {
	positions, err := j.storage.ListPositions(ctx)
	if err != nil {
		return nil, err
	}

	a := &Analytics{}

	var totalWin float64
	var totalLoss float64

	for _, t := range positions {

		// position still open
		if t.ExitPrice == nil {
			continue
		}

		a.TotalPositions++
		profitLoss := t.GetProfitLoss()
		a.TotalPnL += profitLoss

		if profitLoss > 0 {
			a.Wins++
			totalWin += profitLoss
		} else if profitLoss < 0 {
			a.Losses++
			totalLoss += profitLoss
		}
	}

	if a.TotalPositions == 0 {
		return a, nil
	}

	a.WinRate = (float64(a.Wins) / float64(a.TotalPositions))
	a.LossRate = float64(a.Losses) / float64(a.TotalPositions)

	if a.Wins > 0 {
		a.AvgWin = totalWin / float64(a.Wins)
	}

	if a.Losses > 0 {
		a.AvgLoss = totalLoss / float64(a.Losses)
	}

	if totalLoss != 0 {
		a.ProfitFactor = totalWin / (-totalLoss)
	}

	a.Expectancy = a.WinRate*a.AvgWin + a.LossRate*a.AvgLoss

	return a, nil
}

var _ JournalContract = (*Journal)(nil)
