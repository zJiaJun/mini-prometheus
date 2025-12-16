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

const defaultLookbackDelta = 5 * 60 * 1000

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

func (ms *MemoryStorage) Query(m *model.Metric, timestamp int64) (model.Series, error) {
	if m == nil {
		return model.Series{}, ErrNilMetric
	}
	return ms.queryWithLookback(m, timestamp, defaultLookbackDelta)
}

/*
使用Lookback Delta 回溯窗口
在 [timestamp - lookback, timestamp] 范围内查找最新的样本
*/
func (ms *MemoryStorage) queryWithLookback(m *model.Metric, timestamp, lookback int64) (model.Series, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	fp := m.Fingerprint()
	series, ok := ms.series[fp]
	if !ok {
		return model.Series{}, ErrSeriesNotFound
	}
	minTimestamp := timestamp - lookback
	maxTimestamp := timestamp
	var result *model.Sample
	for i := range series.Samples {
		s := &series.Samples[i]
		if s.Timestamp >= minTimestamp && s.Timestamp <= maxTimestamp {
			if result == nil || result.Timestamp < s.Timestamp {
				result = s
			}
		}
	}
	if result == nil {
		return model.Series{Metric: series.Metric}, nil
	}
	return model.Series{Metric: series.Metric, Samples: model.Samples{*result}}, nil
}

func (ms *MemoryStorage) QueryRange(m *model.Metric, start, end int64) (model.Series, error) {
	if m == nil {
		return model.Series{}, ErrNilMetric
	}
	if start > end {
		return model.Series{}, ErrTimeRange
	}
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	fp := m.Fingerprint()
	series, ok := ms.series[fp]
	if !ok {
		return model.Series{}, ErrSeriesNotFound
	}
	filtered := make(model.Samples, 0, len(series.Samples))
	for _, s := range series.Samples {
		if s.Timestamp >= start && s.Timestamp <= end {
			filtered = append(filtered, s)
		}
	}
	return model.Series{Metric: series.Metric, Samples: filtered}, nil
}

func (ms *MemoryStorage) Delete(m *model.Metric) error {
	if m == nil {
		return ErrNilMetric
	}
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	fp := m.Fingerprint()
	delete(ms.series, fp)
	return nil
}
