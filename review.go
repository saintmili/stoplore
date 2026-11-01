package stoplore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RuleAdherence string

const (
	Unassessed    RuleAdherence = ""
	FollowedRules RuleAdherence = "FOLLOWED"
	Deviated      RuleAdherence = "DEVIATED"
)

type TradeReview struct {
	Adherence RuleAdherence
	Notes     string
}

type ReviewCorrection struct {
	PreviousTags   []string
	PreviousReview TradeReview
	CorrectedAt    time.Time
}

type SetTradeReviewRequest struct {
	TradeID   uuid.UUID
	Tags      []string
	Adherence RuleAdherence
	Notes     string
}

// SetTradeReview replaces contextual tags and the optional rule assessment,
// preserving the previous context as a dated correction.
func (s *Service) SetTradeReview(ctx context.Context, req SetTradeReviewRequest) (Trade, error) {
	if req.TradeID == uuid.Nil {
		return Trade{}, fmt.Errorf("%w: trade ID is required", ErrInvalidRequest)
	}
	if req.Adherence != Unassessed && req.Adherence != FollowedRules && req.Adherence != Deviated {
		return Trade{}, fmt.Errorf("%w: rule adherence must be unassessed, followed, or deviated", ErrInvalidRequest)
	}
	tags := make([]string, 0, len(req.Tags))
	seen := make(map[string]bool)
	for _, tag := range req.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return Trade{}, fmt.Errorf("%w: contextual tag must not be blank", ErrInvalidRequest)
		}
		if !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	var trade Trade
	err := s.update(ctx, func(state *journalState) error {
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		var ok bool
		trade, ok = trades[req.TradeID.String()]
		if !ok {
			return fmt.Errorf("%w: trade %s", ErrNotFound, req.TradeID)
		}
		trade.ReviewHistory = append(trade.ReviewHistory, ReviewCorrection{PreviousTags: trade.Tags, PreviousReview: trade.Review, CorrectedAt: time.Now().UTC()})
		trade.Tags = tags
		trade.Review = TradeReview{Adherence: req.Adherence, Notes: req.Notes}
		trades[trade.ID.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return trade, nil
}
