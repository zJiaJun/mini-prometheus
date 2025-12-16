# Instant Query 的 Lookback Delta 策略

## 问题

在实现 Prometheus 的即时查询（Instant Query）时，遇到一个核心问题：**查询的时间戳几乎不可能和存储的样本时间戳完全匹配**。

### 问题场景

```go
// 存储中的样本（每 15 秒抓取一次）
samples := []Sample{
    {Timestamp: 1702450800000, Value: 10},  // 2023-12-13 10:00:00.000
    {Timestamp: 1702450815000, Value: 15},  // 2023-12-13 10:00:15.000
    {Timestamp: 1702450830000, Value: 20},  // 2023-12-13 10:00:30.000
    {Timestamp: 1702450845000, Value: 25},  // 2023-12-13 10:00:45.000
}

// 用户查询：timestamp = 1702450820000（10:00:20.000）
result := storage.Query(metric, 1702450820000)

// 问题：1702450820000 这个时间点没有样本！应该返回什么？
```

### 为什么会出现这个问题？

1. **采样间隔固定**：Prometheus 默认每 15 秒抓取一次指标
2. **查询时间任意**：用户可以查询任意时间点的数据
3. **时间戳不对齐**：查询时间点和采样时间点几乎不会完全一致

**类比**：就像你每隔 15 秒拍一张照片，但用户想看第 20 秒时的画面 —— 这张照片不存在！

---

## Prometheus 的解决方案：Lookback Delta

### 核心思想

Prometheus 使用**回溯时间窗口（Lookback Delta）**策略：在查询时间点之前的一段时间内查找最新的样本。

```
查询时间点: T = 1702450820000 (10:00:20.000)

回溯窗口: [T - lookback, T]
        = [1702450820000 - 300000, 1702450820000]  // lookback = 5 分钟
        = [1702450520000, 1702450820000]
        = [10:00:20 之前 5 分钟, 10:00:20]

在这个范围内查找最新的样本：
  - 1702450800000 (10:00:00) ✅ 在范围内
  - 1702450815000 (10:00:15) ✅ 在范围内，更新
  - 1702450830000 (10:00:30) ❌ 超出查询时间点

返回: {Timestamp: 1702450815000, Value: 15}
```

### 关键参数

| 参数           | 默认值        | 说明                            |
| -------------- | ------------- | ------------------------------- |
| Lookback Delta | 5 分钟        | 默认回溯时间窗口                |
| 范围           | `[T - 5m, T]` | 查找范围                        |
| 选择策略       | 最新样本      | 如果有多个样本，返回最接近 T 的 |

---

## 方案对比

### 方案 A：精确匹配（❌ 不可行）

```go
func Query(metric *Metric, timestamp int64) (*Sample, error) {
    // 只查找时间戳完全相等的样本
    for _, s := range samples {
        if s.Timestamp == timestamp {
            return &s, nil
        }
    }
    return nil, ErrNotFound  // 几乎总是找不到
}
```

**问题**：

- ❌ 几乎永远找不到匹配的样本
- ❌ 只有 1/15000 的概率能精确匹配（15 秒 = 15000 毫秒）

**适用场景**：无

---

### 方案 B：查找最接近的样本（⚠️ 有风险）

```go
func Query(metric *Metric, timestamp int64) (*Sample, error) {
    var closest *Sample
    minDiff := int64(math.MaxInt64)

    for _, s := range samples {
        diff := abs(s.Timestamp - timestamp)
        if diff < minDiff {
            minDiff = diff
            closest = &s
        }
    }

    return closest, nil
}
```

**优点**：

- ✅ 总能找到样本
- ✅ 实现简单

**问题**：

- ❌ **数据时效性无保障**：即使最近的样本在 1 小时前，也会返回
- ❌ 可能返回过时的数据

**示例问题**：

```go
// 服务在 10:00 宕机，最后一个样本是 09:59:45
// 用户在 11:00 查询当前值
// 这个方案会返回 1 小时前的旧数据！❌
```

**适用场景**：数据连续性有保证的场景

---

### 方案 C：Lookback Delta（✅ Prometheus 方案）

```go
const DefaultLookbackDelta = 5 * 60 * 1000  // 5 分钟（毫秒）

func Query(metric *Metric, timestamp int64) (*Sample, error) {
    return QueryWithLookback(metric, timestamp, DefaultLookbackDelta)
}

func QueryWithLookback(metric *Metric, timestamp, lookback int64) (*Sample, error) {
    // 在 [timestamp - lookback, timestamp] 范围内
    // 查找最新的样本

    minTime := timestamp - lookback
    maxTime := timestamp

    var result *Sample
    for _, s := range samples {
        if s.Timestamp >= minTime && s.Timestamp <= maxTime {
            if result == nil || s.Timestamp > result.Timestamp {
                result = &s
            }
        }
    }

    if result == nil {
        return nil, ErrNoDataInRange
    }

    return result, nil
}
```

**优点**：

- ✅ **符合监控场景语义**：数据有时效性
- ✅ **可配置容忍度**：可以根据采样间隔调整 lookback
- ✅ **平衡性好**：既能找到数据，又不会返回过时数据
- ✅ **Prometheus 标准做法**

**缺点**：

- ⚠️ 如果 lookback 设置不当，可能找不到数据

**适用场景**：

- ✅ 监控系统（数据有时效性要求）
- ✅ 时序数据查询
- ✅ 需要平衡"找到数据"和"数据新鲜度"的场景

---

### 方案 D：转换为 QueryRange（🤔 变通方案）

```go
func Query(metric *Metric, timestamp int64) (*Sample, error) {
    // 将即时查询转换为小范围的 range 查询
    lookback := 5 * 60 * 1000  // 5 分钟

    series, err := QueryRange(metric, timestamp-lookback, timestamp)
    if err != nil {
        return nil, err
    }

    // 从返回的样本中取最后一个（最新的）
    if len(series.Samples) == 0 {
        return nil, ErrNoDataInRange
    }

    lastSample := series.Samples[len(series.Samples)-1]
    return &lastSample, nil
}
```

**优点**：

- ✅ 复用 QueryRange 的逻辑
- ✅ 减少代码重复

**缺点**：

- ⚠️ 返回整个 Series 再取最后一个，略有浪费
- ⚠️ 如果范围内有很多样本，性能稍差

**适用场景**：

- 快速原型开发
- 代码简化优先的场景

---

## 实际实现

### 完整代码示例

```go
// pkg/storage/memory.go

const DefaultLookbackDelta = 5 * 60 * 1000  // 5 分钟（毫秒）

// Query 即时查询：返回指定时间点的样本
// 在 [timestamp - DefaultLookbackDelta, timestamp] 范围内查找最新样本
func (ms *MemoryStorage) Query(m *model.Metric, timestamp int64) (model.Series, error) {
    if m == nil {
        return model.Series{}, ErrNilMetric
    }
    return ms.queryWithLookback(m, timestamp, DefaultLookbackDelta)
}

// queryWithLookback 使用指定的 lookback 窗口查询
func (ms *MemoryStorage) queryWithLookback(m *model.Metric, timestamp, lookback int64) (model.Series, error) {
    ms.mutex.RLock()
    defer ms.mutex.RUnlock()

    fp := m.Fingerprint()
    series, ok := ms.series[fp]
    if !ok {
        return model.Series{}, ErrSeriesNotFound
    }

    // 计算查找范围
    minTime := timestamp - lookback
    maxTime := timestamp

    // 在范围内查找最新的样本
    var result *model.Sample
    for i := range series.Samples {
        s := &series.Samples[i]
        if s.Timestamp >= minTime && s.Timestamp <= maxTime {
            if result == nil || result.Timestamp < s.Timestamp {
                result = s
            }
        }
    }

    // 没找到数据，返回空 Samples（不是错误）
    if result == nil {
        return model.Series{Metric: series.Metric}, nil
    }

    // 找到数据，返回包含单个样本的 Series
    return model.Series{
        Metric:  series.Metric,
        Samples: model.Samples{*result},
    }, nil
}
```

### 测试用例

```go
func TestMemoryStorage_Query_Lookback(t *testing.T) {
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

    // 测试 1: 查询当前时间，应该返回最新的样本（1分钟前）
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

    // 测试 2: 查询 10 分钟前的数据（超出 lookback 范围）
    series, err = storage.Query(metric, now.Add(-10*time.Minute).UnixMilli())
    if err != nil {
        t.Fatalf("查询失败: %v", err)
    }
    if len(series.Samples) != 0 {
        t.Errorf("期望 0 个样本（超出 lookback 范围），实际得到 %d 个", len(series.Samples))
    }
}
```

---

## 实际应用场景

### 场景 1：监控仪表盘显示当前值

```go
// 仪表盘查询：显示当前 CPU 使用率
metric := &Metric{Name: "cpu_usage_percent", Labels: ...}

// 查询"现在"的值
series, err := storage.Query(metric, time.Now().UnixMilli())

if len(series.Samples) > 0 {
    currentCPU := series.Samples[0].Value
    fmt.Printf("当前 CPU: %.2f%%", currentCPU)
} else {
    fmt.Println("无最近数据（可能服务已停止）")
}
```

### 场景 2：告警规则评估

```go
// 告警规则：CPU 使用率 > 80%
metric := &Metric{Name: "cpu_usage_percent", Labels: ...}

// 评估当前时刻
series, err := storage.Query(metric, time.Now().UnixMilli())

if len(series.Samples) > 0 {
    if series.Samples[0].Value > 80 {
        // 触发告警
        sendAlert("CPU usage too high: %.2f%%", series.Samples[0].Value)
    }
} else {
    // 无数据也是一种告警（服务可能宕机）
    sendAlert("No data received in last 5 minutes")
}
```

### 场景 3：对比不同采样间隔的处理

```go
// 采样间隔 15 秒的指标
metric1 := &Metric{Name: "fast_metric", ...}
// 默认 5 分钟 lookback 足够

// 采样间隔 1 小时的指标（如备份任务）
metric2 := &Metric{Name: "backup_status", ...}
// 需要更长的 lookback（如 2 小时）
series, err := storage.QueryWithLookback(
    metric2,
    time.Now().UnixMilli(),
    2 * 60 * 60 * 1000,  // 2 小时
)
```

---

## 图解：Lookback Delta 工作原理

```
时间轴：
|-------|-------|-------|-------|-------|-------|-------|
10:00   10:15   10:30   10:45   11:00   11:15   11:30
  ↓       ↓       ↓       ↓
 S1      S2      S3      S4     (采样点)

查询时间: 11:10
         ↓
         Q

Lookback 窗口 [11:05, 11:10]:
|-------|-------|-------|-------|-------|-------|-------|
10:00   10:15   10:30   10:45   11:00   11:15   11:30
                                  ↓       ↓       ↓
                                 S4      [窗口]   Q

结果：返回 S4（11:00 的样本）
      因为它在窗口内且最接近 Q

如果查询时间是 11:20:
|-------|-------|-------|-------|-------|-------|-------|
10:00   10:15   10:30   10:45   11:00   11:15   11:30
                                          ↓       ↓
                                         S5      [窗口] Q

结果：返回 S5（11:15 的样本）
```

---

## 关键设计决策

### 1. 为什么默认 5 分钟？

- **平衡性**：对于大多数监控场景（15-60 秒采样间隔），5 分钟足够宽容
- **时效性**：5 分钟内的数据还有参考价值
- **Prometheus 标准**：与 Prometheus 保持一致

### 2. 为什么返回 Series 而不是 Sample？

- **API 一致性**：与 QueryRange 返回类型统一
- **扩展性**：以后可能需要返回多个值
- **包含元数据**：同时返回 Metric 信息

详见：[Series 和 Sample 的关系](./02-series-samples-relationship.md)

### 3. 找不到数据时返回什么？

**设计决策**：返回空 Samples，而不是错误

```go
if result == nil {
    return model.Series{Metric: series.Metric}, nil  // ✅ 空 Samples
    // 而不是
    // return model.Series{}, ErrNoDataInRange  // ❌ 错误
}
```

**原因**：

- ✅ "时间范围内没数据"是**正常情况**，不是错误
- ✅ Series 不存在才是错误（ErrSeriesNotFound）
- ✅ 调用者可以根据 `len(Samples)` 判断，更灵活

---

## 性能优化考虑

### 当前实现（Phase 1）：线性扫描

```go
// O(n) 时间复杂度
for i := range series.Samples {
    s := &series.Samples[i]
    if s.Timestamp >= minTime && s.Timestamp <= maxTime {
        if result == nil || result.Timestamp < s.Timestamp {
            result = s
        }
    }
}
```

**适用场景**：

- 内存存储
- 样本数量 < 10000

### Phase 4-5 优化：二分查找

如果样本按时间戳排序（Append 保证了顺序）：

```go
// O(log n) 时间复杂度
func binarySearchLookback(samples []Sample, minTime, maxTime int64) *Sample {
    // 1. 二分查找第一个 >= minTime 的位置
    left := sort.Search(len(samples), func(i int) bool {
        return samples[i].Timestamp >= minTime
    })

    // 2. 从该位置向后查找，直到超出 maxTime
    var result *Sample
    for i := left; i < len(samples) && samples[i].Timestamp <= maxTime; i++ {
        result = &samples[i]  // 持续更新，最后一个就是最新的
    }

    return result
}
```

**优化效果**：

- 1000 个样本：10 次比较 vs 1000 次比较
- 100 万个样本：20 次比较 vs 100 万次比较

---

## 总结

| 方面           | 说明                                  |
| -------------- | ------------------------------------- |
| **核心问题**   | 查询时间戳和样本时间戳不会完全匹配    |
| **解决方案**   | Lookback Delta（回溯时间窗口）        |
| **默认窗口**   | 5 分钟                                |
| **查找逻辑**   | 在 [T - lookback, T] 范围内找最新样本 |
| **返回类型**   | Series（包含 0 或 1 个样本）          |
| **空数据处理** | 返回空 Samples，不是错误              |
| **性能**       | Phase 1 线性扫描，Phase 4+ 二分查找   |

### 关键要点

1. ✅ **时效性保证**：只返回 lookback 窗口内的数据
2. ✅ **容错性好**：允许查询时间和采样时间有偏差
3. ✅ **语义清晰**：无数据不是错误，而是正常情况
4. ✅ **符合标准**：与 Prometheus 行为一致

---

## 参考资料

- [Prometheus Staleness and Lookback Delta](https://prometheus.io/docs/prometheus/latest/querying/basics/#staleness)
- [Understanding Prometheus Lookback Delta](https://www.robustperception.io/staleness-and-promql)
- [Instant Query vs Range Query](https://prometheus.io/docs/prometheus/latest/querying/api/)
- [Mini Prometheus Series 设计](./02-series-samples-relationship.md)

---

**注**：本设计基于 Prometheus 2.x 的行为，是监控系统的标准做法。
