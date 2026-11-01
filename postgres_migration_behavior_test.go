package mtl_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	mtl "github.com/saintmili/mtl"
)

// This schema is the old application's source format, not the journal's target storage.
const legacyTradeSchema = `CREATE TABLE trades (
 id UUID PRIMARY KEY, exchange TEXT NOT NULL, strategy TEXT NOT NULL,
 symbol TEXT NOT NULL, side TEXT NOT NULL, order_type TEXT NOT NULL,
 entry_type TEXT NOT NULL, leverage INTEGER NOT NULL, time_frame TEXT NOT NULL,
 entry_price DOUBLE PRECISION NOT NULL, entry_mental_state TEXT NOT NULL,
 position_margin DOUBLE PRECISION NOT NULL, entry_fee_rate DOUBLE PRECISION NOT NULL,
 stop_loss_price DOUBLE PRECISION, take_profit_price DOUBLE PRECISION,
 entry_image_url TEXT NOT NULL, opened_at TIMESTAMPTZ NOT NULL,
 exit_price DOUBLE PRECISION, exit_margin_percent DOUBLE PRECISION,
 exit_fee_rate DOUBLE PRECISION, exit_mental_state TEXT NOT NULL,
 exit_image_url TEXT NOT NULL, exited_at TIMESTAMPTZ, notes TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ)`

func postgresMigrationFixture(t *testing.T, key []byte, fixtureSQL ...string) (*pgxpool.Pool, func() string) {
	t.Helper()
	dsn := os.Getenv("MTL_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set MTL_TEST_POSTGRES_DSN for isolated PostgreSQL source fixtures")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "mtl_source_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	writer, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(writer.Close)
	if _, err = writer.Exec(ctx, legacyTradeSchema); err != nil {
		t.Fatal(err)
	}
	notes := syntheticImportRow()[23]
	if len(key) > 0 {
		notes, err = mtl.Encrypt(notes, key)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = writer.Exec(ctx, `INSERT INTO trades VALUES (
 'c1c0c94b-41dc-4450-9be4-2e67c6078f11','Old venue','Breakout','BTCUSDT','BUY','LIMIT','TRADE',2,'1h',
 100,'calm',50,0.001,90,120,'entry.png','2026-01-01T10:00:00.123456+03:30',
 120,100,NULL,'happy','exit.png','2026-01-02T10:00:00.123456+03:30',$1,'2026-01-01T09:00:00Z',NULL)`, notes)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range fixtureSQL {
		if _, err := writer.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	cfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	source, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(source.Close)
	snapshot := func() string {
		t.Helper()
		var data string
		if err := writer.QueryRow(ctx, `SELECT json_agg(trades ORDER BY id)::text FROM trades`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	return source, snapshot
}

func TestPostgresMigrationPreservesSourceThroughReviewConfirmationAndReopen(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		name := "plaintext"
		var key []byte
		if encrypted {
			name = "encrypted"
			key = []byte("12345678901234567890123456789012")
		}
		t.Run(name, func(t *testing.T) {
			forEachBackend(t, func(t *testing.T, b backend) {
				ctx := context.Background()
				source, snapshot := postgresMigrationFixture(t, key)
				original := snapshot()
				s, err := b.Open()
				if err != nil {
					t.Fatal(err)
				}
				req := mtl.PostgresImportRequest{Source: source, SourceID: "synthetic-postgres", SourceKey: key}
				trades, err := s.ImportPostgres(ctx, req)
				if err != nil {
					t.Fatal(err)
				}
				if len(trades) != 1 {
					t.Fatal(trades)
				}
				tr := trades[0]
				listed, err := s.ListTrades(ctx)
				if err != nil || len(listed) != 1 {
					t.Fatalf("visible journal: %+v %v", listed, err)
				}
				s, err = b.Open()
				if err != nil {
					t.Fatal(err)
				}
				review, err := s.ReviewMigration(ctx, tr.ID)
				if err != nil {
					t.Fatal(err)
				}
				if review.Migration.Format != "postgres" || review.Migration.Fields["NOTES"] != syntheticImportRow()[23] || review.Migration.Fields["EXITFEERATE"] != "" || review.Migration.Fields["UPDATEDAT"] != "" || review.Migration.Fields["ENTRYMENTALSTATE"] != "calm" || review.Migration.Fields["EXITIMAGEURL"] != "exit.png" || len(review.Migration.OriginalFields) != 26 || len(review.Migration.InferredEssentials) != 2 || len(review.PendingEssentials) < 4 || review.OriginalPlan != nil {
					t.Fatalf("review evidence: %+v", review)
				}
				if encrypted && review.Migration.OriginalFields[23] == review.Migration.Fields["NOTES"] {
					t.Fatal("original encrypted notes lost")
				}
				version, err := s.GetSetupVersion(ctx, tr.SetupVersionID)
				if err != nil || version.Rules != "" {
					t.Fatalf("unknown rules: %+v %v", version, err)
				}
				report, err := s.ReportPerformance(ctx, mtl.ReportFilter{})
				if err != nil || len(report.Groups) != 1 || report.Groups[0].ExcludedCount != 1 || report.Groups[0].EligibleCount != 0 {
					t.Fatalf("pending: %+v %v", report, err)
				}
				exchange, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Confirmed venue"})
				if err != nil {
					t.Fatal(err)
				}
				account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: exchange.ID, Name: "Historical"})
				if err != nil {
					t.Fatal(err)
				}
				at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				confirmed, err := s.ConfirmMigration(ctx, mtl.ConfirmMigrationRequest{TradeID: tr.ID, AccountID: account.ID, SettlementCurrency: "USDC", Executions: []mtl.MigrationExecution{
					{Kind: mtl.Entry, ExecutionRequest: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: at, ActualFee: decimalPtr("0.1")}},
					{Kind: mtl.Exit, ExecutionRequest: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("120"), ExecutedAt: at.Add(time.Hour), ActualFee: decimalPtr("0.2")}},
				}})
				if err != nil {
					t.Fatal(err)
				}
				if len(confirmed.PendingEssentials) != 0 || confirmed.Migration.ConfirmedAt == nil || len(confirmed.FundingPayments) != 0 {
					t.Fatalf("confirmation: %+v", confirmed)
				}
				s, err = b.Open()
				if err != nil {
					t.Fatal(err)
				}
				got, err := s.GetTrade(ctx, tr.ID)
				if err != nil || got.NetProfitLoss.String() != "19.7" || got.Migration.Fields["STOPLOSSPRICE"] != "90" || got.Migration.Fields["NOTES"] != syntheticImportRow()[23] {
					t.Fatalf("reopen: %+v %v", got, err)
				}
				report, err = s.ReportPerformance(ctx, mtl.ReportFilter{})
				if err != nil || len(report.Groups) != 1 || report.Groups[0].EligibleCount != 1 || report.Groups[0].ExcludedCount != 0 || report.Groups[0].SettlementCurrency != "USDC" || report.Groups[0].TotalNetProfit.String() != "19.7" {
					t.Fatalf("confirmed report: %+v %v", report, err)
				}
				again, err := s.ImportPostgres(ctx, req)
				if err != nil || len(again) != 1 || again[0].ID != tr.ID || again[0].Migration.ConfirmedAt == nil {
					t.Fatalf("retry: %+v %v", again, err)
				}
				if snapshot() != original {
					t.Fatal("source record changed")
				}
			})
		})
	}
}

func TestPostgresMigrationRejectsUnreadableSourcesWithoutPartialJournalWrites(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.ImportPostgres(ctx, mtl.PostgresImportRequest{SourceID: "missing-pool"}); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("missing pool: %v", err)
		}
		key := []byte("12345678901234567890123456789012")
		source, snapshot := postgresMigrationFixture(t, key)
		original := snapshot()
		if _, err = s.ImportPostgres(ctx, mtl.PostgresImportRequest{Source: source, SourceID: "wrong-key", SourceKey: []byte("abcdefghijklmnopabcdefghijklmnop")}); err == nil {
			t.Fatal("wrong source key accepted")
		}
		if _, err = s.ImportPostgres(ctx, mtl.PostgresImportRequest{Source: source}); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("missing source identity: %v", err)
		}
		if snapshot() != original {
			t.Fatal("decryption failure changed source")
		}
		bad, badSnapshot := postgresMigrationFixture(t, key, `INSERT INTO trades
 SELECT (jsonb_populate_record(NULL::trades, to_jsonb(trades) ||
 '{"id":"e85322f7-2c4d-4999-bbf0-25aa510bf552","notes":"invalid ciphertext"}'::jsonb)).* FROM trades`)
		originalBad := badSnapshot()
		if _, err = s.ImportPostgres(ctx, mtl.PostgresImportRequest{Source: bad, SourceID: "bad-second-row", SourceKey: key}); err == nil {
			t.Fatal("partially decrypted batch accepted")
		}
		if badSnapshot() != originalBad {
			t.Fatal("batch failure changed source")
		}
		empty, _ := postgresMigrationFixture(t, nil, "DROP TABLE trades")
		if _, err = s.ImportPostgres(ctx, mtl.PostgresImportRequest{Source: empty, SourceID: "missing-table"}); err == nil {
			t.Fatal("missing source table accepted")
		}
		var count int
		if err = empty.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema()`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("source storage initialized: %d %v", count, err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		listed, err := s.ListTrades(ctx)
		if err != nil || len(listed) != 0 {
			t.Fatalf("failed imports changed target: %+v %v", listed, err)
		}
	})
}
