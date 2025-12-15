package storage

import "mini-promethues/pkg/model"

type Storage interface {
	Append(m *model.Metric, s *model.Sample) error

	Query(m *model.Metric) (model.Series, error)

	Delete(m *model.Metric) error
}
