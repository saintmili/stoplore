package mtl

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TradeDirection string

const (
	Long  TradeDirection = "LONG"
	Short TradeDirection = "SHORT"
)

type ExecutionKind string

const (
	Entry ExecutionKind = "ENTRY"
	Exit  ExecutionKind = "EXIT"
)

type CostBasis string

const RecordedCosts CostBasis = "RECORDED_COSTS"

type ExecutionFee struct {
	DefaultAmount *Decimal
	ActualAmount  *Decimal
	Rate          *Decimal
	Amount        Decimal
}

type Execution struct {
	ID         uuid.UUID
	Kind       ExecutionKind
	Quantity   Decimal
	Price      Decimal
	ExecutedAt time.Time
	Margin     *Decimal
	Leverage   *Decimal
	Fee        ExecutionFee
}

type ExecutionRequest struct {
	ActualFee  *Decimal
	Quantity   Decimal
	Price      Decimal
	ExecutedAt time.Time
	Margin     *Decimal
	Leverage   *Decimal
}

type Trade struct {
	ContextCorrections  []TradeContextCorrection
	Migration           *MigrationEvidence
	RiskCorrections     []RiskCorrection
	OriginalPlannedRisk *Decimal
	// PendingEssentials identifies migration facts awaiting explicit confirmation.
	PendingEssentials       []string
	FundingCorrections      []FundingCorrection
	ExecutionCorrections    []ExecutionCorrection
	FundingPayments         []FundingPayment
	ReviewHistory           []ReviewCorrection
	Tags                    []string
	Review                  TradeReview
	OriginalPlan            *TradePlan
	PlanRevisions           []PlanRevision
	PlanCorrections         []PlanCorrection
	ID                      uuid.UUID
	AccountID               uuid.UUID
	SetupVersionID          uuid.UUID
	Symbol                  string
	Direction               TradeDirection
	SettlementCurrency      string
	Executions              []Execution
	RemainingQuantity       Decimal
	AverageEntryPrice       Decimal
	RealizedPriceProfitLoss Decimal
	TotalExecutionFees      Decimal
	RecordedFunding         Decimal
	NetProfitLoss           Decimal
	CompletedAt             *time.Time
	CostBasis               CostBasis
	CreatedAt               time.Time
}

type RecordTradeRequest struct {
	OriginalPlannedRisk *Decimal
	OriginalPlan        *TradePlan
	AccountID           uuid.UUID
	SetupVersionID      uuid.UUID
	Symbol              string
	Direction           TradeDirection
	SettlementCurrency  string
	FirstEntry          ExecutionRequest
}

func loadTrades(state *journalState) (map[string]Trade, error) {
	trades := map[string]Trade{}
	if err := state.get("trades", &trades); err != nil {
		return nil, err
	}
	return trades, nil
}

func (s *Service) RecordTrade(ctx context.Context, req RecordTradeRequest) (Trade, error) {
	if err := validateTradePlan(req.OriginalPlan); err != nil {
		return Trade{}, err
	}
	req.Symbol = strings.TrimSpace(req.Symbol)
	if req.AccountID == uuid.Nil || req.SetupVersionID == uuid.Nil || req.Symbol == "" {
		return Trade{}, fmt.Errorf("%w: account, setup version and symbol are required", ErrInvalidRequest)
	}
	if req.Direction != Long && req.Direction != Short {
		return Trade{}, fmt.Errorf("%w: direction must be LONG or SHORT", ErrInvalidRequest)
	}
	if req.SettlementCurrency != "USDT" && req.SettlementCurrency != "USDC" {
		return Trade{}, fmt.Errorf("%w: settlement currency must be USDT or USDC", ErrInvalidRequest)
	}
	if err := validateExecution(req.FirstEntry); err != nil {
		return Trade{}, err
	}
	var trade Trade
	err := s.update(ctx, func(state *journalState) error {
		account, ok := state.Accounts[req.AccountID.String()]
		if !ok {
			return fmt.Errorf("%w: account %s", ErrNotFound, req.AccountID)
		}
		setups := map[string]Setup{}
		if err := state.get("setups", &setups); err != nil {
			return err
		}
		found := false
		for _, setup := range setups {
			for _, version := range setup.Versions {
				if version.ID == req.SetupVersionID {
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("%w: setup version %s", ErrNotFound, req.SetupVersionID)
		}
		exchange, ok := state.Exchanges[account.ExchangeID.String()]
		if !ok {
			return fmt.Errorf("%w: exchange %s", ErrNotFound, account.ExchangeID)
		}
		entry := req.FirstEntry
		fee, err := executionFee(entry, exchange.EntryFeeRate)
		if err != nil {
			return err
		}
		trade = Trade{OriginalPlannedRisk: req.OriginalPlannedRisk, OriginalPlan: req.OriginalPlan, ID: uuid.New(), AccountID: req.AccountID, SetupVersionID: req.SetupVersionID, Symbol: req.Symbol, Direction: req.Direction, SettlementCurrency: req.SettlementCurrency, CreatedAt: time.Now().UTC(), Executions: []Execution{{ID: uuid.New(), Kind: Entry, Quantity: entry.Quantity, Price: entry.Price, ExecutedAt: entry.ExecutedAt, Margin: entry.Margin, Leverage: entry.Leverage, Fee: fee}}}
		if err := replayTrade(&trade); err != nil {
			return err
		}
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		trades[trade.ID.String()] = trade
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return trade, nil
}

func validateExecution(req ExecutionRequest) error {
	if !req.Quantity.IsPositive() || !req.Price.IsPositive() || req.ExecutedAt.IsZero() {
		return fmt.Errorf("%w: execution quantity and price must be positive and timestamp is required", ErrInvalidRequest)
	}
	if req.Margin != nil && req.Margin.IsNegative() {
		return fmt.Errorf("%w: margin must be nonnegative", ErrInvalidRequest)
	}
	if req.Leverage != nil && !req.Leverage.IsPositive() {
		return fmt.Errorf("%w: leverage must be positive", ErrInvalidRequest)
	}
	return nil
}

func (s *Service) GetTrade(ctx context.Context, id uuid.UUID) (Trade, error) {
	state, err := s.read(ctx)
	if err != nil {
		return Trade{}, err
	}
	trades, err := loadTrades(state)
	if err != nil {
		return Trade{}, err
	}
	trade, ok := trades[id.String()]
	if !ok {
		return Trade{}, fmt.Errorf("%w: trade %s", ErrNotFound, id)
	}
	return trade, nil
}

func (s *Service) ListTrades(ctx context.Context) ([]Trade, error) {
	state, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	trades, err := loadTrades(state)
	if err != nil {
		return nil, err
	}
	out := make([]Trade, 0, len(trades))
	for _, trade := range trades {
		out = append(out, trade)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID.String() < out[j].ID.String()
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}
