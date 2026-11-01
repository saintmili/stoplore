package mtl

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const positionsColumnsLen = 26

var positionsHeader = []string{
	"ID",
	"EXCHANGE",
	"STRATEGY",
	"SYMBOL",
	"SIDE",
	"ORDERTYPE",
	"ENTRYTYPE",
	"LEVERAGE",
	"TIMEFRAME",
	"ENTRYPRICE",
	"ENTRYMENTALSTATE",
	"POSITIONMARGIN",
	"ENTRYFEERATE",
	"STOPLOSSPRICE",
	"TAKEPROFITPRICE",
	"ENTRYIMAGEURL",
	"OPENEDAT",
	"EXITPRICE",
	"EXITMARGINPERCENT",
	"EXITFEERATE",
	"EXITMENTALSTATE",
	"EXITIMAGEURL",
	"EXITEDAT",
	"NOTES",
	"CREATEDAT",
	"UPDATEDAT",
}

func (s *CSVStorage) initPositions(ctx context.Context) error {
	filePath := s.dirPath + "/positions.csv"
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
		if err := w.Write(positionsHeader); err != nil {
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

	positions := []*Position{}
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

		position, err := mapRecordToPosition(record)
		if err != nil {
			return err
		}
		positions = append(positions, position)
	}

	s.positions = positions

	return nil
}

func (s *CSVStorage) CreatePosition(ctx context.Context, position Position) (*Position, error) {
	if err := position.Validate(); err != nil {
		return nil, err
	}
	if position.ID == uuid.Nil {
		position.ID = uuid.New()
	}
	if position.CreatedAt.IsZero() {
		position.CreatedAt = time.Now()
	}

	positions, err := s.ListPositions(ctx)
	if err != nil {
		return nil, err
	}
	positions = append(positions, &position)
	s.positions = positions

	if err := s.writePositions(ctx); err != nil {
		return nil, err
	}

	return &position, nil
}

func (s *CSVStorage) GetPosition(ctx context.Context, id uuid.UUID) (*Position, error) {
	for _, position := range s.positions {
		if position.ID == id {
			return position, nil
		}
	}

	return nil, ErrPositionNotFound
}

func (s *CSVStorage) EditPosition(ctx context.Context, req Position) (*Position, error) {
	positions, err := s.ListPositions(ctx)
	if err != nil {
		return nil, err
	}

	found := false
	for i, position := range positions {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if position.ID == req.ID {
			positions[i] = &req
			found = true
			break
		}
	}
	if !found {
		return nil, ErrPositionNotFound
	}

	s.positions = positions

	if err := s.writePositions(ctx); err != nil {
		return nil, err
	}

	return &req, nil
}

func (s *CSVStorage) ListPositions(ctx context.Context) ([]*Position, error) {
	return s.positions, nil
}

func (s *CSVStorage) writePositions(ctx context.Context) error {
	fileName := s.dirPath + "/positions.csv"
	dir := filepath.Dir(fileName)
	tmp, err := os.CreateTemp(dir, ".mtl-*.csv")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	w := csv.NewWriter(tmp)
	if err := w.Write(positionsHeader); err != nil {
		tmp.Close()
		return err
	}

	for _, position := range s.positions {
		select {
		case <-ctx.Done():
			tmp.Close()
			return ctx.Err()
		default:
		}

		if err := w.Write(mapPositionToRecord(position)); err != nil {
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

func mapPositionToRecord(position *Position) []string {
	return []string{
		position.ID.String(),
		position.Exchange,
		position.Strategy,
		position.Symbol,
		string(position.Side),
		position.OrderType,
		position.EntryType,
		strconv.Itoa(position.Leverage),
		position.TimeFrame,
		strconv.FormatFloat(position.EntryPrice, 'f', -1, 64),
		position.EntryMentalState,
		strconv.FormatFloat(position.PositionMargin, 'f', -1, 64),
		strconv.FormatFloat(position.EntryFeeRate, 'f', -1, 64),
		float64PtrToString(position.StopLossPrice),
		float64PtrToString(position.TakeProfitPrice),
		position.EntryImageUrl,
		dateToString(&position.OpenedAt),
		float64PtrToString(position.ExitPrice),
		float64PtrToString(position.ExitMarginPercent),
		float64PtrToString(position.ExitFeeRate),
		position.ExitMentalState,
		position.ExitImageUrl,
		dateToString(position.ExitedAt),
		position.Notes,
		dateToString(&position.CreatedAt),
		dateToString(position.UpdatedAt),
	}
}

func mapRecordToPosition(record []string) (*Position, error) {
	if len(record) != positionsColumnsLen {
		return nil, fmt.Errorf("invalid record length: expected %d got %d", positionsColumnsLen, len(record))
	}
	if !Side(record[4]).IsValid() && Side(record[2]).IsValid() {
		return mapLegacyRecordToPosition(record)
	}

	recordID, err := uuid.Parse(record[0])
	if err != nil {
		return nil, err
	}
	side := Side(record[4])
	if err := side.Validate(); err != nil {
		return nil, err
	}
	leverage, err := strconv.Atoi(record[7])
	if err != nil {
		return nil, err
	}
	entryPrice, err := strconv.ParseFloat(record[9], 64)
	if err != nil {
		return nil, err
	}
	positionMargin, err := strconv.ParseFloat(record[11], 64)
	if err != nil {
		return nil, err
	}
	entryFeeRate, err := strconv.ParseFloat(record[12], 64)
	if err != nil {
		return nil, err
	}
	openedAt, err := time.Parse(time.RFC3339Nano, record[16])
	if err != nil {
		return nil, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, record[24])
	if err != nil {
		return nil, err
	}

	stopLossPrice, err := stringToFloat64Ptr(record[13])
	if err != nil {
		return nil, err
	}
	takeProfitPrice, err := stringToFloat64Ptr(record[14])
	if err != nil {
		return nil, err
	}
	exitPrice, err := stringToFloat64Ptr(record[17])
	if err != nil {
		return nil, err
	}
	exitMarginPercent, err := stringToFloat64Ptr(record[18])
	if err != nil {
		return nil, err
	}
	exitFeeRate, err := stringToFloat64Ptr(record[19])
	if err != nil {
		return nil, err
	}
	exitedAt, err := stringToTimePtr(record[22])
	if err != nil {
		return nil, err
	}
	updatedAt, err := stringToTimePtr(record[25])
	if err != nil {
		return nil, err
	}

	return &Position{
		ID:                recordID,
		Exchange:          record[1],
		Strategy:          record[2],
		Symbol:            record[3],
		Side:              side,
		OrderType:         record[5],
		EntryType:         record[6],
		Leverage:          leverage,
		TimeFrame:         record[8],
		EntryPrice:        entryPrice,
		EntryMentalState:  record[10],
		PositionMargin:    positionMargin,
		EntryFeeRate:      entryFeeRate,
		StopLossPrice:     stopLossPrice,
		TakeProfitPrice:   takeProfitPrice,
		EntryImageUrl:     record[15],
		OpenedAt:          openedAt,
		ExitPrice:         exitPrice,
		ExitMarginPercent: exitMarginPercent,
		ExitFeeRate:       exitFeeRate,
		ExitMentalState:   record[20],
		ExitImageUrl:      record[21],
		ExitedAt:          exitedAt,
		Notes:             record[23],
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}

func mapLegacyRecordToPosition(record []string) (*Position, error) {
	recordID, err := uuid.Parse(record[0])
	if err != nil {
		return nil, err
	}
	side := Side(record[2])
	if err := side.Validate(); err != nil {
		return nil, err
	}
	leverage, err := strconv.Atoi(record[7])
	if err != nil {
		return nil, err
	}
	entryPrice, err := strconv.ParseFloat(record[9], 64)
	if err != nil {
		return nil, err
	}
	positionMargin, err := strconv.ParseFloat(record[10], 64)
	if err != nil {
		return nil, err
	}
	entryFeeRate, err := strconv.ParseFloat(record[11], 64)
	if err != nil {
		return nil, err
	}
	openedAt, err := time.Parse(time.RFC3339Nano, record[14])
	if err != nil {
		return nil, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, record[24])
	if err != nil {
		return nil, err
	}

	exitPrice, err := stringToFloat64Ptr(record[15])
	if err != nil {
		return nil, err
	}
	exitFeeRate, err := stringToFloat64Ptr(record[16])
	if err != nil {
		return nil, err
	}
	takeProfitPrice, err := stringToFloat64Ptr(record[20])
	if err != nil {
		return nil, err
	}
	stopLossPrice, err := stringToFloat64Ptr(record[21])
	if err != nil {
		return nil, err
	}
	exitMarginPercent, err := stringToFloat64Ptr(record[22])
	if err != nil {
		return nil, err
	}
	exitedAt, err := stringToTimePtr(record[19])
	if err != nil {
		return nil, err
	}
	updatedAt, err := stringToTimePtr(record[25])
	if err != nil {
		return nil, err
	}

	return &Position{
		ID:                recordID,
		Symbol:            record[1],
		Side:              side,
		Exchange:          record[3],
		EntryType:         record[4],
		OrderType:         record[5],
		Strategy:          record[6],
		Leverage:          leverage,
		TimeFrame:         record[8],
		EntryPrice:        entryPrice,
		PositionMargin:    positionMargin,
		EntryFeeRate:      entryFeeRate,
		EntryMentalState:  record[12],
		EntryImageUrl:     record[13],
		OpenedAt:          openedAt,
		ExitPrice:         exitPrice,
		ExitFeeRate:       exitFeeRate,
		ExitMentalState:   record[17],
		ExitImageUrl:      record[18],
		ExitedAt:          exitedAt,
		TakeProfitPrice:   takeProfitPrice,
		StopLossPrice:     stopLossPrice,
		ExitMarginPercent: exitMarginPercent,
		Notes:             record[23],
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}
