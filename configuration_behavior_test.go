package mtl_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	mtl "github.com/saintmili/mtl"
	"os"
	"testing"
)

type backend struct{ Open func() (*mtl.Service, error) }

func forEachBackend(t *testing.T, behavior func(*testing.T, backend)) {
	t.Helper()
	t.Run("CSV", func(t *testing.T) {
		dir := t.TempDir()
		behavior(t, backend{Open: func() (*mtl.Service, error) { s := mtl.NewWithCSV(dir); return s, s.Init() }})
	})
	t.Run("PostgreSQL", func(t *testing.T) {
		dsn := os.Getenv("MTL_TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("set MTL_TEST_POSTGRES_DSN to run isolated PostgreSQL behaviors")
		}
		ctx := context.Background()
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Close()
		schema := "mtl_test_" + uuid.New().String()[:8]
		if _, err = pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
			t.Fatal(err)
		}
		defer pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		config, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		config.ConnConfig.RuntimeParams["search_path"] = schema
		isolated, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			t.Fatal(err)
		}
		defer isolated.Close()
		behavior(t, backend{Open: func() (*mtl.Service, error) { s := mtl.NewWithPostgres(isolated, nil); return s, s.Init() }})
	})
}
func TestExchangeAccountsAndFeeDefaultsSurviveReopening(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		entry := mtl.MustDecimal("0.040000000000000000123")
		exit := mtl.MustDecimal("0")
		exchange, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Example", EntryFeeRate: &entry, ExitFeeRate: &exit})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: exchange.ID, Name: "Main"})
		if err != nil {
			t.Fatal(err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.Exchange.GetByID(ctx, exchange.ID.String())
		if err != nil {
			t.Fatal(err)
		}
		if got.EntryFeeRate == nil || got.EntryFeeRate.String() != "0.040000000000000000123" || got.ExitFeeRate == nil || got.ExitFeeRate.String() != "0" {
			t.Fatalf("fee defaults changed: %+v", got)
		}
		saved, err := s.GetAccount(ctx, account.ID)
		if err != nil || saved != account {
			t.Fatalf("account: %+v %v", saved, err)
		}
		unknown, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Unknown rates"})
		if err != nil {
			t.Fatal(err)
		}
		if unknown.EntryFeeRate != nil || unknown.ExitFeeRate != nil {
			t.Fatal("missing fee defaults became zero")
		}
		if _, err = s.Exchange.GetByID(ctx, uuid.NewString()); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing exchange: %v", err)
		}
	})
}
