package stoplore

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const exchangesColumnsLen = 4

var exchangesHeader = []string{
	"ID",
	"NAME",
	"MAKERFEERATE",
	"TAKERFEERATE",
}

func (s *CSVStorage) initExchanges(ctx context.Context) error {
	filePath := s.dirPath + "/exchanges.csv"
	createFile := true
	if _, err := os.Stat(filePath); err == nil {
		createFile = false
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if createFile {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		w := csv.NewWriter(f)
		if err := w.Write(exchangesHeader); err != nil {
			return err
		}
		w.Flush()

		err = w.Error()
		if err != nil {
			return err
		}
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	if _, err = r.Read(); err != nil {
		return err
	}

	exchanges := []*Exchange{}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		exchange, err := mapRecordToExchange(record)
		if err != nil {
			return err
		}
		exchanges = append(exchanges, exchange)
	}

	s.exchanges = exchanges

	return nil
}

func (s *CSVStorage) CreateExchange(ctx context.Context, exchange Exchange) (*Exchange, error) {
	if exchange.ID == uuid.Nil {
		exchange.ID = uuid.New()
	}

	exchanges, err := s.ListExchanges(ctx)
	if err != nil {
		return nil, err
	}
	exchanges = append(exchanges, &exchange)
	s.exchanges = exchanges

	if err := s.writeExchanges(ctx); err != nil {
		return nil, err
	}

	return &exchange, nil
}

func (s *CSVStorage) GetExchange(ctx context.Context, id uuid.UUID) (*Exchange, error) {
	for _, exchange := range s.exchanges {
		if exchange.ID == id {
			return exchange, nil
		}
	}

	return nil, ErrExchangeNotFound
}

func (s *CSVStorage) EditExchange(ctx context.Context, req Exchange) (*Exchange, error) {
	exchanges, err := s.ListExchanges(ctx)
	if err != nil {
		return nil, err
	}

	found := false
	for i, exchange := range exchanges {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exchange.ID == req.ID {
			exchanges[i] = &req
			found = true
			break
		}
	}
	if !found {
		return nil, ErrExchangeNotFound
	}

	s.exchanges = exchanges

	if err := s.writeExchanges(ctx); err != nil {
		return nil, err
	}

	return &req, nil
}

func (s *CSVStorage) ListExchanges(ctx context.Context) ([]*Exchange, error) {
	return s.exchanges, nil
}

func (s *CSVStorage) writeExchanges(ctx context.Context) error {
	fileName := s.dirPath + "/exchanges.csv"
	dir := filepath.Dir(fileName)
	tmp, err := os.CreateTemp(dir, ".stoplore-*.csv")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	w := csv.NewWriter(tmp)
	if err := w.Write(exchangesHeader); err != nil {
		tmp.Close()
		return err
	}

	for _, exchange := range s.exchanges {
		select {
		case <-ctx.Done():
			tmp.Close()
			return ctx.Err()
		default:
		}

		if err := w.Write(mapExchangeToRecord(exchange)); err != nil {
			tmp.Close()
			return err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, fileName)
}

func mapExchangeToRecord(exchange *Exchange) []string {
	return []string{
		exchange.ID.String(),
		exchange.Name,
		float64PtrToString(exchange.MakerFeeRate),
		float64PtrToString(exchange.TakerFeeRate),
	}
}

func mapRecordToExchange(record []string) (*Exchange, error) {
	if len(record) != exchangesColumnsLen {
		return nil, fmt.Errorf("invalid record length: expected %d got %d", exchangesColumnsLen, len(record))
	}

	recordID, err := uuid.Parse(record[0])
	if err != nil {
		return nil, err
	}
	makerFeeRate, err := stringToFloat64Ptr(record[2])
	if err != nil {
		return nil, err
	}
	takerFeeRate, err := stringToFloat64Ptr(record[3])
	if err != nil {
		return nil, err
	}

	return &Exchange{
		ID:           recordID,
		Name:         record[1],
		MakerFeeRate: makerFeeRate,
		TakerFeeRate: takerFeeRate,
	}, nil
}
