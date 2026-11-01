package mtl

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

type CompletedTradeResult struct {
	TradeID             uuid.UUID
	SetupVersionID      uuid.UUID
	SettlementCurrency  string
	NetProfitLoss       Decimal
	OriginalPlannedRisk *Decimal
	RMultiple           *Decimal
}

func completedTradeResult(trade Trade) CompletedTradeResult {
	result := CompletedTradeResult{TradeID: trade.ID, SetupVersionID: trade.SetupVersionID, SettlementCurrency: trade.SettlementCurrency, NetProfitLoss: trade.NetProfitLoss, OriginalPlannedRisk: trade.OriginalPlannedRisk}
	if trade.OriginalPlannedRisk != nil && trade.OriginalPlannedRisk.IsPositive() {
		result.RMultiple = reportRatio(trade.NetProfitLoss, *trade.OriginalPlannedRisk)
	}
	return result
}

type RiskCorrection struct {
	Before, After *Decimal
	CorrectedAt   time.Time
	Reason        string
}

type CorrectOriginalPlannedRiskRequest struct {
	TradeID             uuid.UUID
	OriginalPlannedRisk *Decimal
	CorrectedAt         time.Time
	Reason              string
}

// CorrectOriginalPlannedRisk fixes a journal error, preserving the prior baseline.
// Changes to the trader's decisions belong in plan revisions instead.
func (s *Service) CorrectOriginalPlannedRisk(ctx context.Context, req CorrectOriginalPlannedRiskRequest) (Trade, error) {
	if req.CorrectedAt.IsZero() || strings.TrimSpace(req.Reason) == "" {
		return Trade{}, fmt.Errorf("%w: risk correction date and reason are required", ErrInvalidRequest)
	}
	return s.correctTrade(ctx, req.TradeID, func(trade *Trade) error {
		trade.RiskCorrections = append(trade.RiskCorrections, RiskCorrection{Before: trade.OriginalPlannedRisk, After: req.OriginalPlannedRisk, CorrectedAt: req.CorrectedAt, Reason: req.Reason})
		trade.OriginalPlannedRisk = req.OriginalPlannedRisk
		return nil
	})
}
