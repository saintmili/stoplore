package mtl

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

type MockStorage struct {
	positions map[uuid.UUID]*Position
}

func newMockStorage() *MockStorage {
	return &MockStorage{
		positions: map[uuid.UUID]*Position{},
	}
}

func (m *MockStorage) Init(ctx context.Context) error {
	return nil
}

func (m *MockStorage) CreatePosition(ctx context.Context, position Position) (*Position, error) {
	m.positions[position.ID] = &position
	return &position, nil
}

func (m *MockStorage) GetPosition(ctx context.Context, tradeID uuid.UUID) (*Position, error) {
	position, ok := m.positions[tradeID]
	if !ok {
		return nil, ErrPositionNotFound
	}
	return position, nil
}

func (m *MockStorage) ListPositions(ctx context.Context) ([]*Position, error) {
	positions := make([]*Position, 0, len(m.positions))
	for _, position := range m.positions {
		positions = append(positions, position)
	}
	return positions, nil
}

func (m *MockStorage) EditPosition(ctx context.Context, position Position) (*Position, error) {
	if _, ok := m.positions[position.ID]; !ok {
		return nil, ErrPositionNotFound
	}
	m.positions[position.ID] = &position
	return &position, nil
}

func TestOpenPositionCreatesTrade(t *testing.T) {
	storage := newMockStorage()
	journal := newJournal(storage)
	openedAt := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

	created, err := journal.OpenPosition(context.Background(), &OpenPositionRequest{
		Side:           SideSell,
		Exchange:       "BITUNIX",
		Symbol:         "BTC",
		EntryPrice:     57342.34,
		PositionMargin: 45.04,
		OrderType:      "MARKET",
		EntryType:      "TRADE",
		Leverage:       10,
		TimeFrame:      "4H",
		OpenedAt:       openedAt,
		Notes:          "planned short",
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.ID == uuid.Nil {
		t.Fatal("expected generated position ID")
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected created timestamp")
	}
	if created.OpenedAt != openedAt {
		t.Fatalf("expected opened_at %v, got %v", openedAt, created.OpenedAt)
	}
	if created.Notes != "planned short" {
		t.Fatalf("expected notes to round trip, got %q", created.Notes)
	}
}

func TestClosePositionUpdatesExistingTrade(t *testing.T) {
	storage := newMockStorage()
	journal := newJournal(storage)
	openedAt := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	exitedAt := openedAt.Add(2 * time.Hour)

	created, err := journal.OpenPosition(context.Background(), &OpenPositionRequest{
		Side:           SideBuy,
		Exchange:       "BITUNIX",
		Symbol:         "BTC",
		EntryPrice:     100,
		PositionMargin: 50,
		OrderType:      "MARKET",
		EntryType:      "TRADE",
		Leverage:       2,
		TimeFrame:      "1H",
		OpenedAt:       openedAt,
	})
	if err != nil {
		t.Fatal(err)
	}

	closed, err := journal.ClosePosition(context.Background(), &ClosePositionRequest{
		ID:                created.ID,
		ExitPrice:         110,
		ExitMarginPercent: 20,
		ExitFeeRate:       0.001,
		ExitedAt:          exitedAt,
		Notes:             "closed at target",
	})
	if err != nil {
		t.Fatal(err)
	}

	if closed.ID != created.ID {
		t.Fatalf("expected same position ID, got %s", closed.ID)
	}
	if closed.ExitPrice == nil || *closed.ExitPrice != 110 {
		t.Fatalf("expected exit price to be set, got %+v", closed.ExitPrice)
	}
	if closed.ExitedAt == nil || !closed.ExitedAt.Equal(exitedAt) {
		t.Fatalf("expected exited_at %v, got %+v", exitedAt, closed.ExitedAt)
	}
	if closed.UpdatedAt == nil {
		t.Fatal("expected updated timestamp")
	}
	if closed.Notes != "closed at target" {
		t.Fatalf("expected close notes, got %q", closed.Notes)
	}
}

func TestClosePositionMissingTrade(t *testing.T) {
	journal := newJournal(newMockStorage())

	_, err := journal.ClosePosition(context.Background(), &ClosePositionRequest{
		ID:          uuid.New(),
		ExitPrice:   110,
		ExitFeeRate: 0.001,
		ExitedAt:    time.Now(),
	})
	if !errors.Is(err, ErrPositionNotFound) {
		t.Fatalf("expected ErrPositionNotFound, got %v", err)
	}
}

func TestCSVStorageRoundTripAndEdit(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "positions.csv")
	storage := newCSVStorage(path)
	if err := storage.Init(ctx); err != nil {
		t.Fatal(err)
	}

	stopLoss := 90.0
	takeProfit := 120.0
	openedAt := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	position := Position{
		ID:              uuid.New(),
		Exchange:        "BITUNIX",
		Strategy:        "breakout",
		Symbol:          "BTCUSDT",
		Side:            SideBuy,
		OrderType:       "LIMIT",
		EntryType:       "TRADE",
		Leverage:        3,
		TimeFrame:       "1H",
		EntryPrice:      100,
		PositionMargin:  50,
		EntryFeeRate:    0.001,
		StopLossPrice:   &stopLoss,
		TakeProfitPrice: &takeProfit,
		OpenedAt:        openedAt,
		Notes:           "open",
		CreatedAt:       openedAt,
	}

	created, err := storage.CreatePosition(ctx, position)
	if err != nil {
		t.Fatal(err)
	}
	fetched, err := storage.GetPosition(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Exchange != position.Exchange || fetched.Strategy != position.Strategy || fetched.Symbol != position.Symbol {
		t.Fatalf("unexpected fetched position: %+v", fetched)
	}
	if fetched.StopLossPrice == nil || *fetched.StopLossPrice != stopLoss {
		t.Fatalf("expected stop loss %v, got %+v", stopLoss, fetched.StopLossPrice)
	}

	exitPrice := 110.0
	exitedAt := openedAt.Add(time.Hour)
	fetched.ExitPrice = &exitPrice
	fetched.ExitedAt = &exitedAt
	fetched.Notes = "closed"
	updated, err := storage.EditPosition(ctx, *fetched)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ExitPrice == nil || *updated.ExitPrice != exitPrice {
		t.Fatalf("expected updated exit price, got %+v", updated.ExitPrice)
	}

	listed, err := storage.ListPositions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 position, got %d", len(listed))
	}
	if listed[0].Notes != "closed" {
		t.Fatalf("expected edited notes, got %q", listed[0].Notes)
	}
}

func TestCSVStorageReadsLegacyRecordOrder(t *testing.T) {
	openedAt := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	createdAt := openedAt.Add(time.Minute)
	id := uuid.New()
	record := []string{
		id.String(),
		"BTCUSDT",
		"BUY",
		"BITUNIX",
		"TRADE",
		"LIMIT",
		"breakout",
		"2",
		"1H",
		"100",
		"12",
		"0.001",
		"RELAXED",
		"https://entry.example",
		openedAt.Format(time.RFC3339Nano),
		"",
		"",
		"",
		"",
		"",
		"120",
		"90",
		"",
		"legacy row",
		createdAt.Format(time.RFC3339Nano),
		"",
	}

	position, err := mapRecordToPosition(record)
	if err != nil {
		t.Fatal(err)
	}

	if position.ID != id || position.Exchange != "BITUNIX" || position.Symbol != "BTCUSDT" || position.Side != SideBuy {
		t.Fatalf("unexpected legacy position: %+v", position)
	}
	if position.TakeProfitPrice == nil || *position.TakeProfitPrice != 120 {
		t.Fatalf("expected legacy take profit, got %+v", position.TakeProfitPrice)
	}
	if position.StopLossPrice == nil || *position.StopLossPrice != 90 {
		t.Fatalf("expected legacy stop loss, got %+v", position.StopLossPrice)
	}
}

func TestAnalytics(t *testing.T) {
	storage := newMockStorage()
	journal := newJournal(storage)
	openedAt := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

	winExit := 110.0
	lossExit := 95.0
	storage.positions[uuid.New()] = &Position{
		ID:             uuid.New(),
		Exchange:       "BITUNIX",
		Symbol:         "BTC",
		Side:           SideBuy,
		OrderType:      "MARKET",
		EntryType:      "TRADE",
		Leverage:       1,
		TimeFrame:      "1H",
		EntryPrice:     100,
		PositionMargin: 100,
		OpenedAt:       openedAt,
		CreatedAt:      openedAt,
		ExitPrice:      &winExit,
	}
	storage.positions[uuid.New()] = &Position{
		ID:             uuid.New(),
		Exchange:       "BITUNIX",
		Symbol:         "ETH",
		Side:           SideBuy,
		OrderType:      "MARKET",
		EntryType:      "TRADE",
		Leverage:       1,
		TimeFrame:      "1H",
		EntryPrice:     100,
		PositionMargin: 100,
		OpenedAt:       openedAt,
		CreatedAt:      openedAt,
		ExitPrice:      &lossExit,
	}

	analytics, err := journal.GetAnalytics(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if analytics.TotalPositions != 2 || analytics.Wins != 1 || analytics.Losses != 1 {
		t.Fatalf("unexpected counts: %+v", analytics)
	}
	assertFloat(t, analytics.TotalPnL, 5)
	assertFloat(t, analytics.ProfitFactor, 2)
	assertFloat(t, analytics.Expectancy, 2.5)
}

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	encrypted, err := Encrypt("private note", key)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "private note" {
		t.Fatal("expected ciphertext to differ from plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "private note" {
		t.Fatalf("expected decrypted text, got %q", decrypted)
	}
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

var _ StorageContract = (*MockStorage)(nil)
