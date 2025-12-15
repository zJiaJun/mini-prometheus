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
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	fp := m.Fingerprint()
	if _, ok := ms.series[fp]; ok {

	} else {

	}

	return nil
}

func (ms *MemoryStorage) Query(s *model.Metric) (model.Series, error) {
	return model.Series{}, nil
}

func (ms *MemoryStorage) Delete(s *model.Metric) error {
	return nil
}
