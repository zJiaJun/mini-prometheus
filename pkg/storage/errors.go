package storage

import "errors"

var (
	ErrNilMetric      = errors.New("metric cannot be nil")
	ErrNilSample      = errors.New("sample cannot be nil")
	ErrSeriesNotFound = errors.New("series not found")
	ErrOutOfOrder     = errors.New("sample timestamp out of order")
)
