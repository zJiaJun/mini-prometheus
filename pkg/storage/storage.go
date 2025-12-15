package storage

import "mini-promethues/pkg/model"

type Storage interface {
	Append(m *model.Metric, s *model.Sample) error

	Query(s *model.Metric) (model.Series, error)

	Delete(s *model.Metric) error
}
