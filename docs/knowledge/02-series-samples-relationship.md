# Series 和 Sample 的关系：为什么是一对多

## 问题

在 Mini Prometheus 的数据模型中，`Series` 和 `Sample` 的关系应该是一对一还是一对多？为什么设计文档中 `Series.Samples` 是数组？

## 当前实现（需要修改）

```go
package model

type Series struct {
    Metric Metric
    Sample Sample  // ❌ 错误：应该是数组
}
```

## 正确的设计

```go
package model

type Series struct {
    Metric  Metric
    Samples []Sample  // ✅ 正确：应该是数组
}
```

## 为什么是一对多关系？

### 1. 时间序列的本质

**时间序列（Time Series）的定义：**
- 一条时间序列由**唯一的指标标识**确定（Metric Name + Labels）
- 同一个时间序列在**不同时间点**会产生**多个数据点**（Samples）
- 这些数据点共享相同的指标标识，但时间戳和值不同

**类比理解：**
```
时间序列 = 一条线
样本点 = 线上的多个点
```

就像记录一个城市的气温：
- **时间序列**：`temperature{city="beijing"}`（固定不变）
- **样本点**：在不同时间的温度值（持续变化）

### 2. 实际场景示例

#### 示例 1：CPU 使用率监控

```go
series := &Series{
    Metric: Metric{
        Name: "cpu_usage_percent",
        Labels: []Label{
            {Name: "host", Value: "server1"},
            {Name: "cpu", Value: "0"},
        },
    },
    Samples: []Sample{
        {Timestamp: 1702450800000, Value: 45.2},  // 2023-12-13 10:00:00
        {Timestamp: 1702450815000, Value: 48.7},  // 2023-12-13 10:00:15
        {Timestamp: 1702450830000, Value: 52.1},  // 2023-12-13 10:00:30
        {Timestamp: 1702450845000, Value: 49.8},  // 2023-12-13 10:00:45
    },
}
```

这是**同一条时间序列**在不同时间点的测量值！

#### 示例 2：HTTP 请求计数器

```go
series := &Series{
    Metric: Metric{
        Name: "http_requests_total",
        Labels: []Label{
            {Name: "method", Value: "GET"},
            {Name: "status", Value: "200"},
        },
    },
    Samples: []Sample{
        {Timestamp: 1702450800000, Value: 1027},  // 累计 1027 次
        {Timestamp: 1702450815000, Value: 1035},  // 累计 1035 次
        {Timestamp: 1702450830000, Value: 1042},  // 累计 1042 次
    },
}
```

Counter 类型的指标随时间单调递增，需要记录多个时间点的值。

### 3. 查询需求分析

#### 范围查询（Range Query）

PromQL 查询：
```promql
cpu_usage_percent{host="server1"}[5m]
```

这个查询需要返回**最近 5 分钟内的所有样本点**，可能有几十个甚至上百个样本：

```go
// 返回结果
Series{
    Metric: {...},
    Samples: []Sample{
        // 5 分钟内的所有样本点
        {Timestamp: 1702450800000, Value: 45.2},
        {Timestamp: 1702450815000, Value: 48.7},
        {Timestamp: 1702450830000, Value: 52.1},
        // ... 可能有 20 个样本（15秒抓取间隔）
    },
}
```

如果 `Series` 只能存一个 `Sample`，就无法表示这种查询结果！

#### 即时查询（Instant Query）

即使是即时查询，某些计算也需要历史数据：

```promql
rate(http_requests_total[5m])
```

- `rate()` 函数需要计算 5 分钟内的变化率
- 需要访问多个样本点才能计算
- 内部仍然需要处理多个 Sample

### 4. 存储和缓存需求

#### 内存缓存（Head Block）

Prometheus 的内存缓存会保存最近一段时间的数据：

```go
type MemoryStorage struct {
    series map[uint64]*Series  // SeriesID -> Series
}

// 每条时间序列都包含多个最近的样本
series := storage.series[seriesID]
// series.Samples 包含最近 2 小时的所有样本点
```

#### 磁盘持久化（Persisted Blocks）

数据持久化时，也是以时间块（Block）为单位：

```
Block (2小时的数据)
├── Series 1: Samples[0...480]  // 15秒间隔，2小时=480个样本
├── Series 2: Samples[0...480]
└── Series 3: Samples[0...480]
```

### 5. 数据结构对比

#### 错误设计：一对一

```go
// ❌ 如果是一对一关系
type Series struct {
    Metric Metric
    Sample Sample  // 只能存一个样本
}

// 存储 10 个样本需要创建 10 个 Series 对象
// 每个 Series 都重复存储相同的 Metric 信息！
series1 := Series{Metric: metric, Sample: sample1}
series2 := Series{Metric: metric, Sample: sample2}  // 浪费内存！
series3 := Series{Metric: metric, Sample: sample3}
// ...
```

**问题：**
- 大量重复存储 Metric 信息（Name + Labels）
- 无法表示"这些样本属于同一条时间序列"的语义
- 查询返回结果时需要手动聚合

#### 正确设计：一对多

```go
// ✅ 正确的一对多关系
type Series struct {
    Metric  Metric
    Samples []Sample  // 可以存多个样本
}

// 一个 Series 对象存储所有相关样本
series := Series{
    Metric: metric,
    Samples: []Sample{sample1, sample2, sample3, ...},
}
```

**优点：**
- Metric 信息只存储一次
- 清晰表达"一条时间序列的多个数据点"的概念
- 符合时序数据库的存储模型

## Prometheus 官方实现

查看 Prometheus 源码：

```go
// github.com/prometheus/prometheus/storage
type Series interface {
    Labels() labels.Labels
    Iterator() chunkenc.Iterator  // 迭代器遍历多个样本
}

// 查询结果
type QueryResult struct {
    Series []Series  // 多条时间序列
}

// 每个 Series 包含多个样本点
```

## 完整的数据模型

```go
package model

// Sample 表示时间序列中的一个数据点
type Sample struct {
    Timestamp int64   // Unix时间戳（毫秒）
    Value     float64 // 指标值
}

// Label 表示一个标签键值对
type Label struct {
    Name  string
    Value string
}

// Metric 表示指标的标识（名称+标签）
type Metric struct {
    Name   string
    Labels []Label
}

// Series 表示一条完整的时间序列
// 包含指标标识和该时间序列的所有样本点
type Series struct {
    Metric  Metric   // 指标标识（唯一确定一条时间序列）
    Samples []Sample // 该时间序列的多个样本点
}
```

## 使用示例

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    // 创建一条时间序列
    series := &Series{
        Metric: Metric{
            Name: "http_requests_total",
            Labels: []Label{
                {Name: "method", Value: "GET"},
                {Name: "path", Value: "/api/users"},
                {Name: "status", Value: "200"},
            },
        },
        Samples: []Sample{},
    }
    
    // 随时间推移，不断添加新的样本点
    for i := 0; i < 10; i++ {
        sample := Sample{
            Timestamp: time.Now().UnixMilli(),
            Value:     float64(1000 + i*10),
        }
        series.Samples = append(series.Samples, sample)
        time.Sleep(1 * time.Second)
    }
    
    // 查询该时间序列的所有样本
    fmt.Printf("时间序列: %s\n", series.Metric.Name)
    fmt.Printf("样本点数量: %d\n", len(series.Samples))
    
    // 遍历所有样本
    for _, sample := range series.Samples {
        t := time.UnixMilli(sample.Timestamp)
        fmt.Printf("  [%s] %.2f\n", 
            t.Format("15:04:05"), 
            sample.Value)
    }
}
```

## 图解关系

```
┌─────────────────────────────────────────────────┐
│ Series (时间序列)                                │
├─────────────────────────────────────────────────┤
│ Metric:                                         │
│   Name: "cpu_usage_percent"                     │
│   Labels:                                       │
│     - host: "server1"                           │
│     - cpu: "0"                                  │
├─────────────────────────────────────────────────┤
│ Samples: (样本数组)                              │
│   [0] {Timestamp: 1702450800000, Value: 45.2}  │
│   [1] {Timestamp: 1702450815000, Value: 48.7}  │
│   [2] {Timestamp: 1702450830000, Value: 52.1}  │
│   [3] {Timestamp: 1702450845000, Value: 49.8}  │
│   [4] {Timestamp: 1702450860000, Value: 51.3}  │
│   ...                                           │
└─────────────────────────────────────────────────┘
      ↑                              ↑
      │                              │
   一个Metric                    多个Sample
  (固定不变)                    (持续增加)
```

## 总结

| 对比项 | 一对一（错误） | 一对多（正确） |
|--------|---------------|---------------|
| 内存效率 | 低（重复存储 Metric） | 高（Metric 只存一次） |
| 语义表达 | 不清晰 | 清晰（时间序列概念） |
| 查询结果 | 需要手动聚合 | 直接返回 |
| 存储模型 | 不符合 | 符合 TSDB 设计 |
| 范围查询 | 难以实现 | 自然支持 |

**结论：** `Series` 和 `Sample` 必须是一对多的关系，这是时序数据库的基本设计原则。一条时间序列（Series）代表一个唯一的指标标识，它在时间轴上会产生多个数据点（Samples）。

## 需要修改的代码

将 `pkg/model/Series.go` 修改为：

```go
package model

type Series struct {
    Metric  Metric
    Samples []Sample  // 改为数组
}
```

## 参考资料

- [Prometheus Data Model](https://prometheus.io/docs/concepts/data_model/)
- [Time Series Database Concepts](https://en.wikipedia.org/wiki/Time_series_database)
- [Prometheus TSDB Format](https://github.com/prometheus/prometheus/tree/main/tsdb/docs/format)

