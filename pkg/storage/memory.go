package storage

import (
	"mini-promethues/pkg/model"
	"sync"
)

type MemoryStorage struct {
	series map[uint64]*model.Series
	mutex  sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		series: make(map[uint64]*model.Series),
	}
}

func (ms *MemoryStorage) Append(m *model.Metric, s *model.Sample) error {
	if m == nil {
		return ErrNilMetric
	}
	if s == nil {
		return ErrNilSample
	}
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	fp := m.Fingerprint()
	if series, ok := ms.series[fp]; ok {
		series.Samples = append(series.Samples, *s)
	} else {
		newSeries := &model.Series{Metric: *m, Samples: model.Samples{*s}}
		ms.series[fp] = newSeries
	}
	return nil
}

func (ms *MemoryStorage) Query(m *model.Metric) (model.Series, error) {
	if m == nil {
		return model.Series{}, ErrNilMetric
	}
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	fp := m.Fingerprint()
	if series, ok := ms.series[fp]; ok {
		return *series, nil
	}
	return model.Series{}, ErrSeriesNotFound
}

func (ms *MemoryStorage) Delete(m *model.Metric) error {
	return nil
}
