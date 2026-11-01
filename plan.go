package mtl

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

type TradePlan struct {
	EntryReason string
	Stop        *Decimal
	Target      *Decimal
	Notes       string
}

type PlanRevision struct {
	ID        uuid.UUID
	RevisedAt time.Time
	Plan      TradePlan
}

type PlanCorrection struct {
	ID          uuid.UUID
	RevisionID  uuid.UUID
	CorrectedAt time.Time
	Reason      string
	Before      *TradePlan
	After       *TradePlan
}

type ReviseTradePlanRequest struct {
	TradeID   uuid.UUID
	RevisedAt time.Time
	Plan      TradePlan
}

// Each revision is a full plan snapshot; nil stop or target remains unknown.
func (s *Service) ReviseTradePlan(ctx context.Context, req ReviseTradePlanRequest) (PlanRevision, error) {
	if err := validateTradePlan(&req.Plan); err != nil {
		return PlanRevision{}, err
	}
	if req.RevisedAt.IsZero() {
		return PlanRevision{}, fmt.Errorf("%w: revision date is required", ErrInvalidRequest)
	}
	var revision PlanRevision
	err := s.update(ctx, func(state *journalState) error {
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		trade, ok := trades[req.TradeID.String()]
		if !ok {
			return fmt.Errorf("%w: trade %s", ErrNotFound, req.TradeID)
		}
		revision = PlanRevision{ID: uuid.New(), RevisedAt: req.RevisedAt, Plan: req.Plan}
		trade.PlanRevisions = append(trade.PlanRevisions, revision)
		trades[trade.ID.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return PlanRevision{}, err
	}
	return revision, nil
}

type CorrectTradePlanRequest struct {
	TradeID uuid.UUID
	// A nil UUID selects the original plan; otherwise this selects a revision.
	RevisionID  uuid.UUID
	Plan        *TradePlan
	CorrectedAt time.Time
	Reason      string
}

func (s *Service) CorrectTradePlan(ctx context.Context, req CorrectTradePlanRequest) (PlanCorrection, error) {
	if err := validateTradePlan(req.Plan); err != nil {
		return PlanCorrection{}, err
	}
	if req.CorrectedAt.IsZero() || strings.TrimSpace(req.Reason) == "" {
		return PlanCorrection{}, fmt.Errorf("%w: correction date and reason are required", ErrInvalidRequest)
	}
	if req.RevisionID != uuid.Nil && req.Plan == nil {
		return PlanCorrection{}, fmt.Errorf("%w: a revision must contain a plan", ErrInvalidRequest)
	}
	var correction PlanCorrection
	err := s.update(ctx, func(state *journalState) error {
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		trade, ok := trades[req.TradeID.String()]
		if !ok {
			return fmt.Errorf("%w: trade %s", ErrNotFound, req.TradeID)
		}
		correction = PlanCorrection{ID: uuid.New(), RevisionID: req.RevisionID, CorrectedAt: req.CorrectedAt, Reason: req.Reason, After: req.Plan}
		if req.RevisionID == uuid.Nil {
			correction.Before = trade.OriginalPlan
			trade.OriginalPlan = req.Plan
		} else {
			found := false
			for i := range trade.PlanRevisions {
				if trade.PlanRevisions[i].ID == req.RevisionID {
					before := trade.PlanRevisions[i].Plan
					correction.Before = &before
					trade.PlanRevisions[i].Plan = *req.Plan
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("%w: plan revision %s", ErrNotFound, req.RevisionID)
			}
		}
		trade.PlanCorrections = append(trade.PlanCorrections, correction)
		trades[trade.ID.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return PlanCorrection{}, err
	}
	return correction, nil
}

func validateTradePlan(plan *TradePlan) error {
	if plan == nil {
		return nil
	}
	if (plan.Stop != nil && !plan.Stop.IsPositive()) || (plan.Target != nil && !plan.Target.IsPositive()) {
		return fmt.Errorf("%w: recorded stop and target prices must be positive", ErrInvalidRequest)
	}
	return nil
}
