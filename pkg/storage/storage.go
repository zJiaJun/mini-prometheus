package storage

import "mini-promethues/pkg/model"

type Storage interface {
	Append(m *model.Metric, s *model.Sample) error

	Query(m *model.Metric, timestamp int64) (model.Series, error)

	QueryRange(m *model.Metric, start, end int64) (model.Series, error)

	Delete(m *model.Metric) error
}
