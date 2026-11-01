package stoplore

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresImportRequest identifies legacy trades in the supplied pool's search path.
// SourceID must be a stable, nonsecret identity reused for subsequent imports.
// SourceKey decrypts legacy notes independently of the target's encryption key.
// The caller owns the pool; migration never initializes or writes source storage.
type PostgresImportRequest struct {
	Source    *pgxpool.Pool
	SourceID  string
	SourceKey []byte
}

func (s *Service) ImportPostgres(ctx context.Context, req PostgresImportRequest) ([]Trade, error) {
	sourceID := strings.TrimSpace(req.SourceID)
	if req.Source == nil || sourceID == "" {
		return nil, fmt.Errorf("%w: PostgreSQL source pool and source identity are required", ErrInvalidRequest)
	}
	records, err := readPostgresMigration(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.importMigrationRecords(ctx, sourceID, records)
}

func readPostgresMigration(ctx context.Context, req PostgresImportRequest) ([]migrationRecord, error) {
	tx, err := req.Source.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("open read-only migration source: %w", err)
	}
	defer tx.Rollback(ctx)
	columns := []string{"id", "exchange", "strategy", "symbol", "side", "order_type", "entry_type", "leverage", "time_frame", "entry_price", "entry_mental_state", "position_margin", "entry_fee_rate", "stop_loss_price", "take_profit_price", "entry_image_url", "opened_at", "exit_price", "exit_margin_percent", "exit_fee_rate", "exit_mental_state", "exit_image_url", "exited_at", "notes", "created_at", "updated_at"}
	rows, err := tx.Query(ctx, "SELECT id::text,"+strings.Join(columns[1:], ",")+" FROM trades ORDER BY opened_at,created_at,id")
	if err != nil {
		return nil, fmt.Errorf("read legacy PostgreSQL trades: %w", err)
	}
	defer rows.Close()
	var records []migrationRecord
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("read migration source row: %w", err)
		}
		raw := make([]string, len(columns))
		fields := make(map[string]string, len(columns))
		for i, value := range values {
			// PostgreSQL has already rounded source doubles and timestamps. Preserve
			// their stored values, without asserting the old user's input precision.
			switch v := value.(type) {
			case nil:
			case string:
				raw[i] = v
			case int32:
				raw[i] = strconv.FormatInt(int64(v), 10)
			case float64:
				raw[i] = strconv.FormatFloat(v, 'g', -1, 64)
			case time.Time:
				raw[i] = v.Format(time.RFC3339Nano)
			default:
				return nil, fmt.Errorf("unsupported migration source type for %s: %T", columns[i], value)
			}
			fields[positionsHeader[i]] = raw[i]
		}
		if len(req.SourceKey) > 0 && fields["NOTES"] != "" {
			notes, err := Decrypt(fields["NOTES"], req.SourceKey)
			if err != nil {
				return nil, fmt.Errorf("decrypt source notes for %s: %w", fields["ID"], err)
			}
			fields["NOTES"] = notes
		}
		records = append(records, migrationRecord{Fields: fields, Format: "postgres", OriginalHeader: columns, OriginalFields: raw})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read migration source rows: %w", err)
	}
	return records, nil
}
