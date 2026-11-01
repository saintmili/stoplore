package stoplore

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MigrationEvidence retains source values separately from confirmed journal facts.
// Fields uses current CSV column names; original fields retain the source order.
type MigrationEvidence struct {
	InferredEssentials []string
	SourceID           string
	Format             string
	Fields             map[string]string
	OriginalHeader     []string
	OriginalFields     []string
	ConfirmedAt        *time.Time
}

type CSVImportRequest struct{ Path, SourceID string }

type migrationRecord struct {
	Fields                         map[string]string
	Format                         string
	OriginalHeader, OriginalFields []string
}

// ImportCSV reads an existing positions file without initializing legacy storage.
// SourceID identifies the source across retries; by default it is the absolute path.
func (s *Service) ImportCSV(ctx context.Context, req CSVImportRequest) ([]Trade, error) {
	path, err := filepath.Abs(req.Path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open migration source: %w", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read migration header: %w", err)
	}
	legacy := []string{"ID", "SYMBOL", "SIDE", "EXCHANGE", "ENTRYTYPE", "ORDERTYPE", "STRATEGY", "LEVERAGE", "TIMEFRAME", "ENTRYPRICE", "POSITIONMARGIN", "ENTRYFEERATE", "ENTRYMENTALSTATE", "ENTRYIMAGEURL", "OPENEDAT", "EXITPRICE", "EXITFEERATE", "EXITMENTALSTATE", "EXITIMAGEURL", "EXITEDAT", "TAKEPROFITPRICE", "STOPLOSSPRICE", "EXITMARGINPERCENT", "NOTES", "CREATEDAT", "UPDATEDAT"}
	format := "csv-current"
	if reflect.DeepEqual(header, legacy) {
		format = "csv-legacy"
	} else if !reflect.DeepEqual(header, positionsHeader) {
		return nil, fmt.Errorf("%w: unsupported migration CSV header", ErrInvalidRequest)
	}
	var records []migrationRecord
	for line := 2; ; line++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("migration row %d: %w", line, err)
		}
		rowHeader, rowFormat := header, format
		if format == "csv-current" && !Side(row[4]).IsValid() && Side(row[2]).IsValid() {
			rowHeader, rowFormat = legacy, "csv-legacy"
		}
		fields := map[string]string{}
		for i, key := range rowHeader {
			fields[key] = row[i]
		}
		records = append(records, migrationRecord{Fields: fields, Format: rowFormat, OriginalHeader: header, OriginalFields: row})
	}
	source := strings.TrimSpace(req.SourceID)
	if source == "" {
		source = path
	}
	return s.importMigrationRecords(ctx, source, records)
}

func (s *Service) importMigrationRecords(ctx context.Context, sourceID string, records []migrationRecord) ([]Trade, error) {
	if strings.TrimSpace(sourceID) == "" {
		return nil, fmt.Errorf("%w: migration source identity is required", ErrInvalidRequest)
	}
	pending := make([]Trade, 0, len(records))
	seen := map[uuid.UUID]bool{}
	for i, record := range records {
		trade, err := prepareMigration(sourceID, record)
		if err != nil {
			return nil, fmt.Errorf("migration record %d: %w", i+1, err)
		}
		if seen[trade.ID] {
			return nil, fmt.Errorf("%w: duplicate source record %s", ErrInvalidRequest, record.Fields["ID"])
		}
		seen[trade.ID] = true
		pending = append(pending, trade)
	}
	result := make([]Trade, 0, len(pending))
	err := s.update(ctx, func(state *journalState) error {
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		setups := map[string]Setup{}
		if err := state.get("setups", &setups); err != nil {
			return err
		}
		for _, trade := range pending {
			if existing, ok := trades[trade.ID.String()]; ok {
				evidence := existing.Migration
				if evidence == nil || evidence.SourceID != sourceID || !reflect.DeepEqual(evidence.Fields, trade.Migration.Fields) || !reflect.DeepEqual(evidence.OriginalFields, trade.Migration.OriginalFields) || !reflect.DeepEqual(evidence.OriginalHeader, trade.Migration.OriginalHeader) || evidence.Format != trade.Migration.Format {
					return fmt.Errorf("%w: source record changed since import", ErrInvalidRequest)
				}
				result = append(result, existing)
				continue
			}
			if strings.TrimSpace(trade.Migration.Fields["STRATEGY"]) == "" {
				trade.PendingEssentials = append(trade.PendingEssentials, "setup")
				trades[trade.ID.String()] = trade
				result = append(result, trade)
				continue
			}
			// Historical strategy names are known, but their rules were never stored.
			setupID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(sourceID+"/setup/"+trade.Migration.Fields["STRATEGY"]))
			setup, ok := setups[setupID.String()]
			if !ok {
				setup = Setup{ID: setupID, Name: trade.Migration.Fields["STRATEGY"]}
				setup.Versions = []SetupVersion{{ID: uuid.NewSHA1(setupID, []byte("unknown-rules")), SetupID: setupID, Version: 1, CreatedAt: trade.CreatedAt}}
				setups[setupID.String()] = setup
			}
			trade.SetupVersionID = setup.Versions[0].ID
			trades[trade.ID.String()] = trade
			result = append(result, trade)
		}
		if err := state.put("setups", setups); err != nil {
			return err
		}
		return state.put("trades", trades)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func prepareMigration(source string, record migrationRecord) (Trade, error) {
	f := record.Fields
	id, err := uuid.Parse(f["ID"])
	if err != nil || id == uuid.Nil {
		return Trade{}, fmt.Errorf("%w: invalid source ID", ErrInvalidRequest)
	}
	if f["SIDE"] != "BUY" && f["SIDE"] != "SELL" {
		return Trade{}, fmt.Errorf("%w: invalid source side", ErrInvalidRequest)
	}
	if strings.TrimSpace(f["SYMBOL"]) == "" {
		return Trade{}, fmt.Errorf("%w: source symbol is required", ErrInvalidRequest)
	}
	numbers := map[string]*Decimal{}
	for _, key := range []string{"LEVERAGE", "ENTRYPRICE", "POSITIONMARGIN", "ENTRYFEERATE", "STOPLOSSPRICE", "TAKEPROFITPRICE", "EXITPRICE", "EXITMARGINPERCENT", "EXITFEERATE"} {
		if f[key] == "" {
			continue
		}
		value, err := ParseDecimal(f[key])
		if err != nil {
			return Trade{}, fmt.Errorf("%s: %w", key, err)
		}
		numbers[key] = &value
	}
	dates := map[string]time.Time{}
	for _, key := range []string{"OPENEDAT", "EXITEDAT", "CREATEDAT", "UPDATEDAT"} {
		if f[key] == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, f[key])
		if err != nil {
			return Trade{}, fmt.Errorf("%w: %s: %v", ErrInvalidRequest, key, err)
		}
		dates[key] = at
	}
	evidence := &MigrationEvidence{SourceID: source, Format: record.Format, Fields: f, OriginalHeader: record.OriginalHeader, OriginalFields: record.OriginalFields}
	tr := Trade{ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte(source+"/record/"+id.String())), Symbol: f["SYMBOL"], Direction: Long, CreatedAt: dates["CREATEDAT"], Migration: evidence, PendingEssentials: []string{"account", "settlement_currency", "entry_quantity", "entry_fee"}, CostBasis: RecordedCosts}
	if f["SIDE"] == "SELL" {
		tr.Direction = Short
	}
	if f["ENTRYTYPE"] != "TRADE" {
		tr.PendingEssentials = append(tr.PendingEssentials, "trade_classification")
	}
	quantity := Decimal{}
	price := Decimal{}
	if numbers["ENTRYPRICE"] != nil {
		price = *numbers["ENTRYPRICE"]
	}
	if price.IsPositive() && numbers["POSITIONMARGIN"] != nil && numbers["LEVERAGE"] != nil {
		quantity = numbers["POSITIONMARGIN"].Mul(*numbers["LEVERAGE"]).DivRound(price, 18)
		if quantity.IsPositive() {
			evidence.InferredEssentials = append(evidence.InferredEssentials, "entry_quantity")
		}
	}
	entry := Execution{ID: uuid.NewSHA1(tr.ID, []byte("entry")), Kind: Entry, Price: price, Quantity: quantity, ExecutedAt: dates["OPENEDAT"], Margin: numbers["POSITIONMARGIN"], Leverage: numbers["LEVERAGE"]}
	// Source fractional rates remain evidence; actual historical fees require review.
	tr.Executions = []Execution{entry}
	tr.RemainingQuantity = quantity
	tr.AverageEntryPrice = price
	if !price.IsPositive() {
		tr.PendingEssentials = append(tr.PendingEssentials, "entry_price")
	}
	if entry.ExecutedAt.IsZero() {
		tr.PendingEssentials = append(tr.PendingEssentials, "entry_time")
	}
	if f["EXITPRICE"] != "" || f["EXITEDAT"] != "" || f["EXITMARGINPERCENT"] != "" || f["EXITFEERATE"] != "" {
		tr.PendingEssentials = append(tr.PendingEssentials, "exit_quantity", "exit_fee", "exit_completion")
		exitQuantity := quantity
		if percent := numbers["EXITMARGINPERCENT"]; percent != nil {
			exitQuantity = quantity.Mul(*percent).DivRound(MustDecimal("100"), 18)
		}
		if exitQuantity.IsPositive() {
			evidence.InferredEssentials = append(evidence.InferredEssentials, "exit_quantity")
		}
		exitPrice := Decimal{}
		if numbers["EXITPRICE"] != nil {
			exitPrice = *numbers["EXITPRICE"]
		}
		tr.Executions = append(tr.Executions, Execution{ID: uuid.NewSHA1(tr.ID, []byte("exit")), Kind: Exit, Quantity: exitQuantity, Price: exitPrice, ExecutedAt: dates["EXITEDAT"]})
		if !exitPrice.IsPositive() {
			tr.PendingEssentials = append(tr.PendingEssentials, "exit_price")
		}
		if dates["EXITEDAT"].IsZero() {
			tr.PendingEssentials = append(tr.PendingEssentials, "exit_time")
		}
		// A historical exit is provisional: reports count it as excluded until reviewed.
		percent := numbers["EXITMARGINPERCENT"]
		partial := percent != nil && percent.IsPositive() && percent.LessThan(MustDecimal("100"))
		if partial {
			tr.RemainingQuantity = quantity.Sub(exitQuantity)
		}
		if at := dates["EXITEDAT"]; !at.IsZero() && !partial {
			tr.CompletedAt = &at
		}
	}
	return tr, nil
}

// ReviewMigration exposes durable source evidence and every unconfirmed essential.
func (s *Service) ReviewMigration(ctx context.Context, id uuid.UUID) (Trade, error) {
	tr, err := s.GetTrade(ctx, id)
	if err != nil {
		return Trade{}, err
	}
	if tr.Migration == nil {
		return Trade{}, fmt.Errorf("%w: trade was not migrated", ErrInvalidRequest)
	}
	return tr, nil
}

type MigrationExecution struct {
	Kind ExecutionKind
	ExecutionRequest
}

type ConfirmMigrationRequest struct {
	TradeID, AccountID uuid.UUID
	SetupVersionID     uuid.UUID
	SettlementCurrency string
	Executions         []MigrationExecution
	ConfirmTrade       bool
}

// ConfirmMigration replaces provisional executions with explicitly confirmed facts.
// Later changes use the ordinary correction workflow, retaining this source evidence.
func (s *Service) ConfirmMigration(ctx context.Context, req ConfirmMigrationRequest) (Trade, error) {
	if req.AccountID == uuid.Nil || (req.SettlementCurrency != "USDT" && req.SettlementCurrency != "USDC") || len(req.Executions) == 0 {
		return Trade{}, fmt.Errorf("%w: account, settlement currency and executions are required", ErrInvalidRequest)
	}
	for _, fill := range req.Executions {
		if fill.Kind != Entry && fill.Kind != Exit {
			return Trade{}, fmt.Errorf("%w: invalid execution kind", ErrInvalidRequest)
		}
		if err := validateExecution(fill.ExecutionRequest); err != nil {
			return Trade{}, err
		}
		if fill.ActualFee == nil {
			return Trade{}, fmt.Errorf("%w: confirmed actual fee required for every execution", ErrInvalidRequest)
		}
	}
	var confirmed Trade
	err := s.update(ctx, func(state *journalState) error {
		if _, ok := state.Accounts[req.AccountID.String()]; !ok {
			return fmt.Errorf("%w: account %s", ErrNotFound, req.AccountID)
		}
		trades, err := loadTrades(state)
		if err != nil {
			return err
		}
		tr, ok := trades[req.TradeID.String()]
		if !ok {
			return fmt.Errorf("%w: trade %s", ErrNotFound, req.TradeID)
		}
		if tr.Migration == nil || tr.Migration.ConfirmedAt != nil {
			return fmt.Errorf("%w: trade is not awaiting migration confirmation", ErrInvalidRequest)
		}
		if tr.Migration.Fields["ENTRYTYPE"] != "TRADE" && !req.ConfirmTrade {
			return fmt.Errorf("%w: confirm source record represents a trade", ErrInvalidRequest)
		}
		if req.SetupVersionID != uuid.Nil {
			tr.SetupVersionID = req.SetupVersionID
		}
		if err := requireNamedSetupVersion(state, tr.SetupVersionID); err != nil {
			return err
		}
		tr.AccountID = req.AccountID
		tr.SettlementCurrency = req.SettlementCurrency
		tr.Executions = nil
		for _, fill := range req.Executions {
			fee, err := executionFee(fill.ExecutionRequest, nil)
			if err != nil {
				return err
			}
			tr.Executions = append(tr.Executions, Execution{ID: uuid.New(), Kind: fill.Kind, Quantity: fill.Quantity, Price: fill.Price, ExecutedAt: fill.ExecutedAt, Margin: fill.Margin, Leverage: fill.Leverage, Fee: fee})
		}
		if err := replayTrade(&tr); err != nil {
			return err
		}
		tr.PendingEssentials = nil
		at := time.Now().UTC()
		tr.Migration.ConfirmedAt = &at
		trades[tr.ID.String()] = tr
		confirmed = tr
		return state.put("trades", trades)
	})
	if err != nil {
		return Trade{}, err
	}
	return confirmed, nil
}
