package mtl

import (
	"context"
	"strconv"
	"time"
)

type CSVStorage struct {
	dirPath string

	positions []*Position
	exchanges []*Exchange
}

func newCSVStorage(dirPath string) *CSVStorage {
	return &CSVStorage{
		dirPath: dirPath,
	}
}

func (s *CSVStorage) Init(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	err := s.initPositions(ctx)
	if err != nil {
		return err
	}

	err = s.initExchanges(ctx)
	if err != nil {
		return err
	}

	return nil
}

func stringToFloat64Ptr(v string) (*float64, error) {
	if v == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func stringToTimePtr(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func float64PtrToString(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}

func dateToString(v *time.Time) string {
	if v == nil || v.IsZero() {
		return ""
	}
	return v.Format(time.RFC3339Nano)
}

var _ StorageContract = (*CSVStorage)(nil)
