package stoplore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Tags requires every listed tag; nil Adherence accepts any assessment.
// A pointer to Unassessed explicitly selects trades without an assessment.
type ReportFilter struct {
	Start, End time.Time
	Symbol     string
	Direction  TradeDirection
	Tags       []string
	Adherence  *RuleAdherence
}

// SetupPerformance keeps each setup version and settlement currency separate.
// Ratios are nil when their denominator is zero; AverageLoss retains its sign.
type SetupPerformance struct {
	SetupID, SetupVersionID                                          uuid.UUID
	SetupName, SettlementCurrency                                    string
	SetupVersion                                                     int
	EligibleCount, ExcludedCount, Wins, Losses, BreakEven            int
	TotalNetProfit                                                   Decimal
	AverageNetProfit, WinRate, AverageWin, AverageLoss, ProfitFactor *Decimal
}

type PerformanceReport struct {
	// Completed contains the eligible completed trades selected by the filter.
	Completed []CompletedTradeResult
	CostBasis CostBasis
	Groups    []SetupPerformance
	Ongoing   []Trade
}

// ReportPerformance selects completed trades by final close in [Start, End).
// Zero bounds are unbounded. Each selected result includes all recorded costs.
// Ongoing trades obey context filters but have no closing-date restriction.
func (s *Service) ReportPerformance(ctx context.Context, filter ReportFilter) (PerformanceReport, error) {
	filter.Tags = append([]string(nil), filter.Tags...)
	for i, tag := range filter.Tags {
		if strings.TrimSpace(tag) == "" {
			return PerformanceReport{}, fmt.Errorf("%w: report tag must not be blank", ErrInvalidRequest)
		}
		filter.Tags[i] = strings.TrimSpace(tag)
	}
	if !filter.Start.IsZero() && !filter.End.IsZero() && !filter.Start.Before(filter.End) {
		return PerformanceReport{}, fmt.Errorf("%w: report start must precede end", ErrInvalidRequest)
	}
	if filter.Direction != "" && filter.Direction != Long && filter.Direction != Short {
		return PerformanceReport{}, fmt.Errorf("%w: invalid report direction", ErrInvalidRequest)
	}
	if filter.Adherence != nil && *filter.Adherence != Unassessed && *filter.Adherence != FollowedRules && *filter.Adherence != Deviated {
		return PerformanceReport{}, fmt.Errorf("%w: invalid report adherence", ErrInvalidRequest)
	}
	state, err := s.read(ctx)
	if err != nil {
		return PerformanceReport{}, err
	}
	trades, err := loadTrades(state)
	if err != nil {
		return PerformanceReport{}, err
	}
	setups := map[string]Setup{}
	if err = state.get("setups", &setups); err != nil {
		return PerformanceReport{}, err
	}
	type identity struct {
		setup   Setup
		version SetupVersion
	}
	identities := map[uuid.UUID]identity{}
	for _, setup := range setups {
		for _, version := range setup.Versions {
			identities[version.ID] = identity{setup, version}
		}
	}
	type key struct {
		version  uuid.UUID
		currency string
	}
	groups := map[key]*SetupPerformance{}
	positive, negative := map[key]Decimal{}, map[key]Decimal{}
	report := PerformanceReport{CostBasis: RecordedCosts, Completed: []CompletedTradeResult{}, Groups: []SetupPerformance{}, Ongoing: []Trade{}}
	for _, trade := range trades {
		if !matchesReport(trade, filter) {
			continue
		}
		if trade.CompletedAt == nil {
			report.Ongoing = append(report.Ongoing, trade)
			continue
		}
		if (!filter.Start.IsZero() && trade.CompletedAt.Before(filter.Start)) || (!filter.End.IsZero() && !trade.CompletedAt.Before(filter.End)) {
			continue
		}
		k := key{trade.SetupVersionID, trade.SettlementCurrency}
		g := groups[k]
		if g == nil {
			id := identities[trade.SetupVersionID]
			g = &SetupPerformance{SetupID: id.setup.ID, SetupName: id.setup.Name, SetupVersionID: trade.SetupVersionID, SetupVersion: id.version.Version, SettlementCurrency: trade.SettlementCurrency}
			groups[k] = g
		}
		if !reportEligible(trade, identities[trade.SetupVersionID].version.ID != uuid.Nil && strings.TrimSpace(identities[trade.SetupVersionID].setup.Name) != "") {
			g.ExcludedCount++
			continue
		}
		report.Completed = append(report.Completed, completedTradeResult(trade))
		g.EligibleCount++
		g.TotalNetProfit = g.TotalNetProfit.Add(trade.NetProfitLoss)
		switch {
		case trade.NetProfitLoss.IsPositive():
			g.Wins++
			positive[k] = positive[k].Add(trade.NetProfitLoss)
		case trade.NetProfitLoss.IsNegative():
			g.Losses++
			negative[k] = negative[k].Add(trade.NetProfitLoss)
		default:
			g.BreakEven++
		}
	}
	for k, g := range groups {
		g.AverageNetProfit = reportRatio(g.TotalNetProfit, MustDecimal(fmt.Sprint(g.EligibleCount)))
		g.WinRate = reportRatio(MustDecimal(fmt.Sprint(g.Wins)), MustDecimal(fmt.Sprint(g.EligibleCount)))
		g.AverageWin = reportRatio(positive[k], MustDecimal(fmt.Sprint(g.Wins)))
		g.AverageLoss = reportRatio(negative[k], MustDecimal(fmt.Sprint(g.Losses)))
		g.ProfitFactor = reportRatio(positive[k], negative[k].Neg())
		report.Groups = append(report.Groups, *g)
	}
	sort.Slice(report.Groups, func(i, j int) bool {
		a, b := report.Groups[i], report.Groups[j]
		if a.SetupName != b.SetupName {
			return a.SetupName < b.SetupName
		}
		if a.SetupID != b.SetupID {
			return a.SetupID.String() < b.SetupID.String()
		}
		if a.SetupVersionID != b.SetupVersionID {
			return a.SetupVersionID.String() < b.SetupVersionID.String()
		}
		return a.SettlementCurrency < b.SettlementCurrency
	})
	sort.Slice(report.Completed, func(i, j int) bool {
		return report.Completed[i].TradeID.String() < report.Completed[j].TradeID.String()
	})
	sort.Slice(report.Ongoing, func(i, j int) bool { return report.Ongoing[i].ID.String() < report.Ongoing[j].ID.String() })
	return report, nil
}

func matchesReport(trade Trade, filter ReportFilter) bool {
	if filter.Symbol != "" && filter.Symbol != trade.Symbol || filter.Direction != "" && filter.Direction != trade.Direction || filter.Adherence != nil && *filter.Adherence != trade.Review.Adherence {
		return false
	}
	for _, wanted := range filter.Tags {
		found := false
		for _, tag := range trade.Tags {
			if tag == wanted {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func reportEligible(trade Trade, knownSetup bool) bool {
	if len(trade.PendingEssentials) > 0 || !knownSetup || trade.AccountID == uuid.Nil || trade.Symbol == "" || (trade.Direction != Long && trade.Direction != Short) || (trade.SettlementCurrency != "USDT" && trade.SettlementCurrency != "USDC") || len(trade.Executions) == 0 {
		return false
	}
	for _, fill := range trade.Executions {
		if !fill.Quantity.IsPositive() || !fill.Price.IsPositive() || fill.ExecutedAt.IsZero() || (fill.Fee.DefaultAmount == nil && fill.Fee.ActualAmount == nil) {
			return false
		}
	}
	return true
}
func reportRatio(numerator, denominator Decimal) *Decimal {
	if denominator.IsZero() {
		return nil
	}
	result := numerator.DivRound(denominator, 18)
	return &result
}
