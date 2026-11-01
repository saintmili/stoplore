package stoplore

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	db  *pgxpool.Pool
	key []byte
}

func newPostgresStorage(db *pgxpool.Pool, key []byte) *PostgresStorage {
	return &PostgresStorage{
		db:  db,
		key: key,
	}
}

func (s *PostgresStorage) Init(ctx context.Context) error {
	_, err := s.db.Exec(
		ctx,
		`
			CREATE TABLE IF NOT EXISTS trades (
				id UUID PRIMARY KEY,
				exchange TEXT NOT NULL,
				strategy TEXT NOT NULL,
				symbol TEXT NOT NULL,
				side TEXT NOT NULL,
				order_type TEXT NOT NULL,
				entry_type TEXT NOT NULL,
				leverage INTEGER NOT NULL,
				time_frame TEXT NOT NULL,
				entry_price DOUBLE PRECISION NOT NULL,
				entry_mental_state TEXT NOT NULL,
				position_margin DOUBLE PRECISION NOT NULL,
				entry_fee_rate DOUBLE PRECISION NOT NULL,
				stop_loss_price DOUBLE PRECISION,
				take_profit_price DOUBLE PRECISION,
				entry_image_url TEXT NOT NULL,
				opened_at TIMESTAMPTZ NOT NULL,
				exit_price DOUBLE PRECISION,
				exit_margin_percent DOUBLE PRECISION,
				exit_fee_rate DOUBLE PRECISION,
				exit_mental_state TEXT NOT NULL,
				exit_image_url TEXT NOT NULL,
				exited_at TIMESTAMPTZ,
				notes TEXT NOT NULL,
				created_at TIMESTAMPTZ NOT NULL,
				updated_at TIMESTAMPTZ
			);
		`,
	)

	return err
}

func (s *PostgresStorage) CreatePosition(ctx context.Context, position Position) (*Position, error) {
	if err := position.Validate(); err != nil {
		return nil, err
	}
	if position.ID == uuid.Nil {
		position.ID = uuid.New()
	}
	if position.CreatedAt.IsZero() {
		position.CreatedAt = time.Now()
	}

	notes, err := s.encryptString(position.Notes)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(
		ctx,
		`
			INSERT INTO trades (
				id,
				exchange,
				strategy,
				symbol,
				side,
				order_type,
				entry_type,
				leverage,
				time_frame,
				entry_price,
				entry_mental_state,
				position_margin,
				entry_fee_rate,
				stop_loss_price,
				take_profit_price,
				entry_image_url,
				opened_at,
				exit_price,
				exit_margin_percent,
				exit_fee_rate,
				exit_mental_state,
				exit_image_url,
				exited_at,
				notes,
				created_at,
				updated_at
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)
		`,
		position.ID,
		position.Exchange,
		position.Strategy,
		position.Symbol,
		position.Side,
		position.OrderType,
		position.EntryType,
		position.Leverage,
		position.TimeFrame,
		position.EntryPrice,
		position.EntryMentalState,
		position.PositionMargin,
		position.EntryFeeRate,
		position.StopLossPrice,
		position.TakeProfitPrice,
		position.EntryImageUrl,
		position.OpenedAt,
		position.ExitPrice,
		position.ExitMarginPercent,
		position.ExitFeeRate,
		position.ExitMentalState,
		position.ExitImageUrl,
		position.ExitedAt,
		notes,
		position.CreatedAt,
		position.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &position, nil
}

func (s *PostgresStorage) GetPosition(ctx context.Context, id uuid.UUID) (*Position, error) {
	row := s.db.QueryRow(
		ctx,
		`
			SELECT
				id,
				exchange,
				strategy,
				symbol,
				side,
				order_type,
				entry_type,
				leverage,
				time_frame,
				entry_price,
				entry_mental_state,
				position_margin,
				entry_fee_rate,
				stop_loss_price,
				take_profit_price,
				entry_image_url,
				opened_at,
				exit_price,
				exit_margin_percent,
				exit_fee_rate,
				exit_mental_state,
				exit_image_url,
				exited_at,
				notes,
				created_at,
				updated_at
			FROM trades
			WHERE id = $1
		`,
		id,
	)

	position, err := s.scanPosition(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionNotFound
	}
	if err != nil {
		return nil, err
	}

	return position, nil
}

func (s *PostgresStorage) ListPositions(ctx context.Context) ([]*Position, error) {
	rows, err := s.db.Query(
		ctx,
		`
			SELECT
				id,
				exchange,
				strategy,
				symbol,
				side,
				order_type,
				entry_type,
				leverage,
				time_frame,
				entry_price,
				entry_mental_state,
				position_margin,
				entry_fee_rate,
				stop_loss_price,
				take_profit_price,
				entry_image_url,
				opened_at,
				exit_price,
				exit_margin_percent,
				exit_fee_rate,
				exit_mental_state,
				exit_image_url,
				exited_at,
				notes,
				created_at,
				updated_at
			FROM trades
			ORDER BY opened_at, created_at
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := []*Position{}
	for rows.Next() {
		position, err := s.scanPosition(rows)
		if err != nil {
			return nil, err
		}
		positions = append(positions, position)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return positions, nil
}

func (s *PostgresStorage) EditPosition(ctx context.Context, position Position) (*Position, error) {
	notes, err := s.encryptString(position.Notes)
	if err != nil {
		return nil, err
	}

	tag, err := s.db.Exec(
		ctx,
		`
			UPDATE trades
			SET
				exchange = $2,
				strategy = $3,
				symbol = $4,
				side = $5,
				order_type = $6,
				entry_type = $7,
				leverage = $8,
				time_frame = $9,
				entry_price = $10,
				entry_mental_state = $11,
				position_margin = $12,
				entry_fee_rate = $13,
				stop_loss_price = $14,
				take_profit_price = $15,
				entry_image_url = $16,
				opened_at = $17,
				exit_price = $18,
				exit_margin_percent = $19,
				exit_fee_rate = $20,
				exit_mental_state = $21,
				exit_image_url = $22,
				exited_at = $23,
				notes = $24,
				created_at = $25,
				updated_at = $26
			WHERE id = $1
		`,
		position.ID,
		position.Exchange,
		position.Strategy,
		position.Symbol,
		position.Side,
		position.OrderType,
		position.EntryType,
		position.Leverage,
		position.TimeFrame,
		position.EntryPrice,
		position.EntryMentalState,
		position.PositionMargin,
		position.EntryFeeRate,
		position.StopLossPrice,
		position.TakeProfitPrice,
		position.EntryImageUrl,
		position.OpenedAt,
		position.ExitPrice,
		position.ExitMarginPercent,
		position.ExitFeeRate,
		position.ExitMentalState,
		position.ExitImageUrl,
		position.ExitedAt,
		notes,
		position.CreatedAt,
		position.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrPositionNotFound
	}

	return &position, nil
}

type positionScanner interface {
	Scan(dest ...any) error
}

func (s *PostgresStorage) scanPosition(scanner positionScanner) (*Position, error) {
	var position Position
	var side Side
	var notes string

	err := scanner.Scan(
		&position.ID,
		&position.Exchange,
		&position.Strategy,
		&position.Symbol,
		&side,
		&position.OrderType,
		&position.EntryType,
		&position.Leverage,
		&position.TimeFrame,
		&position.EntryPrice,
		&position.EntryMentalState,
		&position.PositionMargin,
		&position.EntryFeeRate,
		&position.StopLossPrice,
		&position.TakeProfitPrice,
		&position.EntryImageUrl,
		&position.OpenedAt,
		&position.ExitPrice,
		&position.ExitMarginPercent,
		&position.ExitFeeRate,
		&position.ExitMentalState,
		&position.ExitImageUrl,
		&position.ExitedAt,
		&notes,
		&position.CreatedAt,
		&position.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := side.Validate(); err != nil {
		return nil, err
	}
	position.Side = side

	position.Notes, err = s.decryptString(notes)
	if err != nil {
		return nil, err
	}

	return &position, nil
}

func (s *PostgresStorage) encryptString(value string) (string, error) {
	if len(s.key) == 0 || value == "" {
		return value, nil
	}
	return Encrypt(value, s.key)
}

func (s *PostgresStorage) decryptString(value string) (string, error) {
	if len(s.key) == 0 || value == "" {
		return value, nil
	}
	return Decrypt(value, s.key)
}

var _ StorageContract = (*PostgresStorage)(nil)
