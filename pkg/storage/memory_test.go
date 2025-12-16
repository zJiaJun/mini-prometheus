package storage

import (
	"mini-promethues/pkg/model"
	"sync"
	"testing"
	"time"
)

// 辅助函数：创建测试用的 Metric
func createTestMetric(name string, labels ...string) *model.Metric {
	var labelList model.Labels
	for i := 0; i < len(labels); i += 2 {
		if i+1 < len(labels) {
			labelList = append(labelList, model.Label{
				Name:  labels[i],
				Value: labels[i+1],
			})
		}
	}
	return &model.Metric{
		Name:   name,
		Labels: labelList,
	}
}

// 辅助函数：创建测试用的 Sample
func createTestSample(offset time.Duration, value float64) *model.Sample {
	return &model.Sample{
		Timestamp: time.Now().Add(offset).UnixMilli(),
		Value:     value,
	}
}

// TestMemoryStorage_Append 测试追加样本功能
func TestMemoryStorage_Append(t *testing.T) {
	t.Run("追加第一个样本", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("http_requests_total", "method", "GET", "status", "200")
		sample := createTestSample(0, 100)
		err := storage.Append(metric, sample)
		if err != nil {
			t.Fatalf("追加样本失败: %v", err)
		}

		// 验证样本已存储
		series, err := storage.QueryRange(metric, 0, time.Now().Add(time.Second).UnixMilli())
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}
		if len(series.Samples) != 1 {
			t.Errorf("期望 1 个样本，实际得到 %d 个", len(series.Samples))
		}
		if series.Samples[0].Value != 100 {
			t.Errorf("期望值为 100，实际为 %f", series.Samples[0].Value)
		}
	})

	t.Run("追加多个样本到同一个时间序列", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("cpu_usage", "host", "server1")

		// 追加 5 个样本
		for i := 0; i < 5; i++ {
			sample := createTestSample(time.Duration(i)*time.Second, float64(10+i))
			err := storage.Append(metric, sample)
			if err != nil {
				t.Fatalf("追加第 %d 个样本失败: %v", i, err)
			}
		}

		// 验证所有样本都已存储
		series, err := storage.QueryRange(metric, 0, time.Now().Add(10*time.Second).UnixMilli())
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}
		if len(series.Samples) != 5 {
			t.Errorf("期望 5 个样本，实际得到 %d 个", len(series.Samples))
		}

		// 验证值的正确性
		for i, s := range series.Samples {
			expectedValue := float64(10 + i)
			if s.Value != expectedValue {
				t.Errorf("样本 %d: 期望值 %f，实际值 %f", i, expectedValue, s.Value)
			}
		}
	})

	t.Run("追加到不同的时间序列", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric1 := createTestMetric("cpu_usage", "host", "server1")
		metric2 := createTestMetric("cpu_usage", "host", "server2")

		sample1 := createTestSample(0, 50)
		sample2 := createTestSample(0, 75)

		storage.Append(metric1, sample1)
		storage.Append(metric2, sample2)

		// 验证两个时间序列独立存储
		series1, _ := storage.QueryRange(metric1, 0, time.Now().Add(time.Second).UnixMilli())
		series2, _ := storage.QueryRange(metric2, 0, time.Now().Add(time.Second).UnixMilli())

		if len(series1.Samples) != 1 || series1.Samples[0].Value != 50 {
			t.Error("metric1 数据不正确")
		}
		if len(series2.Samples) != 1 || series2.Samples[0].Value != 75 {
			t.Error("metric2 数据不正确")
		}
	})

	t.Run("nil Metric 参数", func(t *testing.T) {
		storage := NewMemoryStorage()
		sample := createTestSample(0, 100)
		err := storage.Append(nil, sample)
		if err != ErrNilMetric {
			t.Errorf("期望错误 ErrNilMetric，实际得到 %v", err)
		}
	})

	t.Run("nil Sample 参数", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("test", "label", "value")
		err := storage.Append(metric, nil)
		if err != ErrNilSample {
			t.Errorf("期望错误 ErrNilSample，实际得到 %v", err)
		}
	})
}

// TestMemoryStorage_Query 测试即时查询功能
func TestMemoryStorage_Query(t *testing.T) {
	t.Run("查询存在的样本-精确匹配时间戳", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("temperature", "city", "beijing")

		now := time.Now()
		sample := &model.Sample{
			Timestamp: now.UnixMilli(),
			Value:     25.5,
		}
		storage.Append(metric, sample)

		// 查询相同时间戳
		series, err := storage.Query(metric, now.UnixMilli())
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}
		if len(series.Samples) != 1 {
			t.Fatalf("期望 1 个样本，实际得到 %d 个", len(series.Samples))
		}
		if series.Samples[0].Value != 25.5 {
			t.Errorf("期望值 25.5，实际值 %f", series.Samples[0].Value)
		}
	})

	t.Run("查询存在的样本-使用 Lookback", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("memory_usage", "host", "server1")

		// 插入样本（当前时间 - 1分钟）
		pastTime := time.Now().Add(-1 * time.Minute)
		sample := &model.Sample{
			Timestamp: pastTime.UnixMilli(),
			Value:     1024.0,
		}
		storage.Append(metric, sample)

		// 查询当前时间（应该能找到 1 分钟前的样本，因为在 5 分钟 lookback 内）
		series, err := storage.Query(metric, time.Now().UnixMilli())
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}
		if len(series.Samples) != 1 {
			t.Fatalf("期望找到 1 个样本，实际得到 %d 个", len(series.Samples))
		}
		if series.Samples[0].Value != 1024.0 {
			t.Errorf("期望值 1024.0，实际值 %f", series.Samples[0].Value)
		}
	})

	t.Run("查询最新的样本-多个样本在范围内", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("requests", "endpoint", "/api")

		now := time.Now()
		// 插入 3 个样本（过去 3 分钟内）
		samples := []*model.Sample{
			{Timestamp: now.Add(-3 * time.Minute).UnixMilli(), Value: 100},
			{Timestamp: now.Add(-2 * time.Minute).UnixMilli(), Value: 200},
			{Timestamp: now.Add(-1 * time.Minute).UnixMilli(), Value: 300},
		}

		for _, s := range samples {
			storage.Append(metric, s)
		}

		// 查询当前时间，应该返回最新的样本（1分钟前的）
		series, err := storage.Query(metric, now.UnixMilli())
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}
		if len(series.Samples) != 1 {
			t.Fatalf("期望 1 个样本，实际得到 %d 个", len(series.Samples))
		}
		if series.Samples[0].Value != 300 {
			t.Errorf("期望返回最新样本（值 300），实际值 %f", series.Samples[0].Value)
		}
	})

	t.Run("查询时间范围外的数据-无结果", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("old_metric", "type", "test")

		// 插入 10 分钟前的样本（超出默认 5 分钟 lookback）
		oldTime := time.Now().Add(-10 * time.Minute)
		sample := &model.Sample{
			Timestamp: oldTime.UnixMilli(),
			Value:     999,
		}
		storage.Append(metric, sample)

		// 查询当前时间（应该找不到，因为超出 lookback 范围）
		series, err := storage.Query(metric, time.Now().UnixMilli())
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}
		if len(series.Samples) != 0 {
			t.Errorf("期望 0 个样本（超出 lookback 范围），实际得到 %d 个", len(series.Samples))
		}
	})

	t.Run("查询不存在的时间序列", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("nonexistent", "label", "value")

		_, err := storage.Query(metric, time.Now().UnixMilli())
		if err != ErrSeriesNotFound {
			t.Errorf("期望错误 ErrSeriesNotFound，实际得到 %v", err)
		}
	})

	t.Run("nil Metric 参数", func(t *testing.T) {
		storage := NewMemoryStorage()
		_, err := storage.Query(nil, time.Now().UnixMilli())
		if err != ErrNilMetric {
			t.Errorf("期望错误 ErrNilMetric，实际得到 %v", err)
		}
	})
}

// TestMemoryStorage_QueryRange 测试范围查询功能
func TestMemoryStorage_QueryRange(t *testing.T) {
	t.Run("查询指定时间范围内的所有样本", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("disk_usage", "mount", "/data")

		now := time.Now()
		baseTime := now.Add(-10 * time.Minute)

		// 插入 10 个样本（每分钟一个）
		for i := 0; i < 10; i++ {
			sample := &model.Sample{
				Timestamp: baseTime.Add(time.Duration(i) * time.Minute).UnixMilli(),
				Value:     float64(50 + i*5),
			}
			storage.Append(metric, sample)
		}

		// 查询中间 5 分钟的数据（索引 2-6）
		start := baseTime.Add(2 * time.Minute).UnixMilli()
		end := baseTime.Add(7 * time.Minute).UnixMilli()

		series, err := storage.QueryRange(metric, start, end)
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}

		// 应该返回 6 个样本（索引 2,3,4,5,6,7）
		expectedCount := 6
		if len(series.Samples) != expectedCount {
			t.Errorf("期望 %d 个样本，实际得到 %d 个", expectedCount, len(series.Samples))
		}

		// 验证第一个和最后一个样本的值
		if series.Samples[0].Value != 60 {
			t.Errorf("第一个样本值错误，期望 60，实际 %f", series.Samples[0].Value)
		}
		if series.Samples[len(series.Samples)-1].Value != 85 {
			t.Errorf("最后一个样本值错误，期望 85，实际 %f", series.Samples[len(series.Samples)-1].Value)
		}
	})

	t.Run("查询边界时间-包含边界样本", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("boundary_test", "id", "1")

		now := time.Now()
		samples := []*model.Sample{
			{Timestamp: now.Add(-3 * time.Second).UnixMilli(), Value: 1},
			{Timestamp: now.Add(-2 * time.Second).UnixMilli(), Value: 2},
			{Timestamp: now.Add(-1 * time.Second).UnixMilli(), Value: 3},
		}

		for _, s := range samples {
			storage.Append(metric, s)
		}

		// 查询精确包含第一个和最后一个样本的时间范围
		start := samples[0].Timestamp
		end := samples[2].Timestamp

		series, err := storage.QueryRange(metric, start, end)
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}

		if len(series.Samples) != 3 {
			t.Errorf("期望 3 个样本（包含边界），实际得到 %d 个", len(series.Samples))
		}
	})

	t.Run("查询空时间范围-无数据", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("empty_range", "test", "true")

		now := time.Now()
		// 插入当前时间的样本
		sample := &model.Sample{
			Timestamp: now.UnixMilli(),
			Value:     100,
		}
		storage.Append(metric, sample)

		// 查询过去的时间范围（不包含任何样本）
		start := now.Add(-10 * time.Minute).UnixMilli()
		end := now.Add(-5 * time.Minute).UnixMilli()

		series, err := storage.QueryRange(metric, start, end)
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}

		if len(series.Samples) != 0 {
			t.Errorf("期望 0 个样本，实际得到 %d 个", len(series.Samples))
		}
	})

	t.Run("无效的时间范围-start > end", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("test", "label", "value")

		now := time.Now()
		start := now.UnixMilli()
		end := now.Add(-1 * time.Hour).UnixMilli()

		_, err := storage.QueryRange(metric, start, end)
		if err != ErrTimeRange {
			t.Errorf("期望错误 ErrTimeRange，实际得到 %v", err)
		}
	})

	t.Run("查询不存在的时间序列", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("nonexistent", "label", "value")

		now := time.Now()
		_, err := storage.QueryRange(metric, now.Add(-1*time.Hour).UnixMilli(), now.UnixMilli())
		if err != ErrSeriesNotFound {
			t.Errorf("期望错误 ErrSeriesNotFound，实际得到 %v", err)
		}
	})

	t.Run("nil Metric 参数", func(t *testing.T) {
		storage := NewMemoryStorage()
		now := time.Now()
		_, err := storage.QueryRange(nil, now.Add(-1*time.Hour).UnixMilli(), now.UnixMilli())
		if err != ErrNilMetric {
			t.Errorf("期望错误 ErrNilMetric，实际得到 %v", err)
		}
	})
}

// TestMemoryStorage_Delete 测试删除功能
func TestMemoryStorage_Delete(t *testing.T) {
	t.Run("删除存在的时间序列", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("to_delete", "status", "active")

		// 插入数据
		sample := createTestSample(0, 123)
		storage.Append(metric, sample)

		// 验证数据存在
		_, err := storage.QueryRange(metric, 0, time.Now().Add(time.Second).UnixMilli())
		if err != nil {
			t.Fatalf("删除前查询失败: %v", err)
		}

		// 删除
		err = storage.Delete(metric)
		if err != nil {
			t.Fatalf("删除失败: %v", err)
		}

		// 验证数据已删除
		_, err = storage.QueryRange(metric, 0, time.Now().Add(time.Second).UnixMilli())
		if err != ErrSeriesNotFound {
			t.Errorf("期望错误 ErrSeriesNotFound，实际得到 %v", err)
		}
	})

	t.Run("删除不存在的时间序列-不报错", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("nonexistent", "label", "value")

		// 删除不存在的序列应该成功（Go map delete 特性）
		err := storage.Delete(metric)
		if err != nil {
			t.Errorf("删除不存在的序列不应该报错，实际得到: %v", err)
		}
	})
}

// TestMemoryStorage_Concurrent 测试并发安全性
func TestMemoryStorage_Concurrent(t *testing.T) {
	t.Run("并发写入", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("concurrent_write", "test", "parallel")

		var wg sync.WaitGroup
		goroutines := 100
		samplesPerGoroutine := 10

		// 启动多个 goroutine 并发写入
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < samplesPerGoroutine; j++ {
					sample := &model.Sample{
						Timestamp: time.Now().Add(time.Duration(id*samplesPerGoroutine+j) * time.Millisecond).UnixMilli(),
						Value:     float64(id*100 + j),
					}
					err := storage.Append(metric, sample)
					if err != nil {
						t.Errorf("goroutine %d: 写入失败: %v", id, err)
					}
				}
			}(i)
		}

		wg.Wait()

		// 验证所有数据都已写入
		series, err := storage.QueryRange(metric, 0, time.Now().Add(time.Hour).UnixMilli())
		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}

		expectedCount := goroutines * samplesPerGoroutine
		if len(series.Samples) != expectedCount {
			t.Errorf("期望 %d 个样本，实际得到 %d 个", expectedCount, len(series.Samples))
		}
	})

	t.Run("并发读写", func(t *testing.T) {
		storage := NewMemoryStorage()
		metric := createTestMetric("concurrent_rw", "test", "mixed")

		// 先写入一些初始数据
		for i := 0; i < 50; i++ {
			sample := createTestSample(time.Duration(i)*time.Millisecond, float64(i))
			storage.Append(metric, sample)
		}

		var wg sync.WaitGroup
		operations := 50

		// 并发写入
		for i := 0; i < operations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sample := createTestSample(time.Duration(50+id)*time.Millisecond, float64(50+id))
				storage.Append(metric, sample)
			}(i)
		}

		// 并发读取
		for i := 0; i < operations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := storage.QueryRange(metric, 0, time.Now().Add(time.Hour).UnixMilli())
				if err != nil && err != ErrSeriesNotFound {
					t.Errorf("查询失败: %v", err)
				}
			}()
		}

		wg.Wait()

		// 验证最终数据正确性
		series, err := storage.QueryRange(metric, 0, time.Now().Add(time.Hour).UnixMilli())
		if err != nil {
			t.Fatalf("最终查询失败: %v", err)
		}

		expectedCount := 100
		if len(series.Samples) != expectedCount {
			t.Errorf("期望 %d 个样本，实际得到 %d 个", expectedCount, len(series.Samples))
		}
	})
}

// TestMemoryStorage_RealWorldScenario 测试真实场景
func TestMemoryStorage_RealWorldScenario(t *testing.T) {
	t.Run("模拟 CPU 监控场景", func(t *testing.T) {
		storage := NewMemoryStorage()

		// 创建 3 个服务器的 CPU 指标
		servers := []string{"server1", "server2", "server3"}
		baseTime := time.Now().Add(-30 * time.Minute)

		// 模拟每 15 秒采集一次数据，持续 30 分钟
		interval := 15 * time.Second
		duration := 30 * time.Minute
		sampleCount := int(duration / interval)

		for _, server := range servers {
			metric := createTestMetric("cpu_usage_percent", "host", server, "cpu", "total")

			for i := 0; i < sampleCount; i++ {
				timestamp := baseTime.Add(time.Duration(i) * interval)
				// 模拟 CPU 使用率在 30-70% 之间波动
				cpuUsage := 50.0 + 20.0*float64(i%10)/10.0
				sample := &model.Sample{
					Timestamp: timestamp.UnixMilli(),
					Value:     cpuUsage,
				}
				err := storage.Append(metric, sample)
				if err != nil {
					t.Fatalf("写入数据失败: %v", err)
				}
			}
		}

		// 场景 1: 查询最近 5 分钟的数据
		t.Run("查询最近5分钟", func(t *testing.T) {
			metric := createTestMetric("cpu_usage_percent", "host", "server1", "cpu", "total")
			start := time.Now().Add(-5 * time.Minute).UnixMilli()
			end := time.Now().UnixMilli()

			series, err := storage.QueryRange(metric, start, end)
			if err != nil {
				t.Fatalf("查询失败: %v", err)
			}

			// 5分钟 = 300秒，每15秒一个样本，应该有约20个样本
			expectedMin, expectedMax := 18, 22
			actualCount := len(series.Samples)
			if actualCount < expectedMin || actualCount > expectedMax {
				t.Errorf("期望 %d-%d 个样本，实际得到 %d 个", expectedMin, expectedMax, actualCount)
			}

			t.Logf("✓ 查询到 %d 个样本", actualCount)
		})

		// 场景 2: 即时查询（当前值）
		t.Run("查询当前值", func(t *testing.T) {
			metric := createTestMetric("cpu_usage_percent", "host", "server2", "cpu", "total")

			series, err := storage.Query(metric, time.Now().UnixMilli())
			if err != nil {
				t.Fatalf("查询失败: %v", err)
			}

			if len(series.Samples) == 0 {
				t.Error("期望能查询到当前值")
			} else {
				t.Logf("✓ 当前 CPU 使用率: %.2f%%", series.Samples[0].Value)
			}
		})

		// 场景 3: 验证不同服务器的数据独立性
		t.Run("验证多服务器数据独立", func(t *testing.T) {
			for _, server := range servers {
				metric := createTestMetric("cpu_usage_percent", "host", server, "cpu", "total")
				start := baseTime.UnixMilli()
				end := time.Now().UnixMilli()

				series, err := storage.QueryRange(metric, start, end)
				if err != nil {
					t.Fatalf("查询 %s 失败: %v", server, err)
				}

				if len(series.Samples) == 0 {
					t.Errorf("%s 没有数据", server)
				}

				t.Logf("✓ %s: %d 个样本", server, len(series.Samples))
			}
		})
	})
}
