# 为什么 Sample 中 Timestamp 使用 int64 而不是 time.Time

## 问题

在 Mini Prometheus 的数据模型设计中，`Sample` 结构体的 `Timestamp` 字段为什么使用 `int64` 类型存储 Unix 时间戳（毫秒），而不是使用 Go 标准库的 `time.Time` 类型？

## 当前实现

```go
package model

type Sample struct {
    Timestamp int64   // Unix时间戳（毫秒）
    Value     float64 // 指标值
}
```

## 设计原因

### 1. 性能和内存效率

**int64 的优势：**

- `int64` 只占用 **8 字节**
- `time.Time` 结构体包含多个字段：
  - `int64` (秒)
  - `int32` (纳秒)
  - `*Location` 指针（时区信息）
  - 总计占用更多内存（24+ 字节）

**影响分析：**

- 时序数据库需要存储海量样本点（每秒可能数万个）
- 如果每个样本多占用 16 字节，100 万个样本就多占用约 15 MB 内存
- 对于大规模监控系统，内存占用差异会被显著放大

### 2. 序列化和网络传输

**int64 的优势：**

- 可以直接序列化为 JSON、Protocol Buffers 等格式
- 无需额外的类型转换和时区处理
- 网络传输更高效，字节数更少

**Prometheus 生态的一致性：**

- Prometheus 的 HTTP API 使用 Unix 时间戳（毫秒）
- Remote Write/Read 协议使用 int64
- 存储格式（TSDB）也使用 int64
- 保持与生态系统的兼容性

### 3. 比较和排序操作

**int64 的优势：**

```go
// int64 可以直接比较
if sample1.Timestamp < sample2.Timestamp {
    // 处理逻辑
}

// time.Time 需要方法调用
if sample1.Timestamp.Before(sample2.Timestamp) {
    // 处理逻辑
}
```

- 直接使用比较运算符 `<`、`>`、`==`，代码更简洁
- 避免方法调用的开销（虽然很小，但在热路径上累积可观）
- 更适合作为 map 的 key 或进行二分查找

### 4. 时间精度统一

**监控场景的精度需求：**

- Prometheus 使用**毫秒精度**（1ms = 10^-3 秒）
- 监控数据不需要纳秒级精度（time.Time 支持纳秒）
- 避免过高精度带来的额外复杂性

**避免时区问题：**

- Unix 时间戳是 UTC 时间，全球统一
- 不需要考虑时区转换
- 避免夏令时等复杂问题

### 5. 数据压缩友好

**时序数据压缩算法：**

- Prometheus 使用 Delta-of-Delta 编码压缩时间戳
- 该算法针对连续整数设计
- `int64` 可以直接应用压缩算法
- `time.Time` 需要先转换，增加开销

## 类型转换

虽然内部使用 `int64`，但在需要时可以轻松转换为 `time.Time`：

### int64 转 time.Time

```go
// 方法1: 从毫秒时间戳转换
timestamp := int64(1702450800000) // 毫秒
t := time.Unix(0, timestamp*1e6)  // 转换为纳秒

// 方法2: 使用 UnixMilli (Go 1.17+)
t := time.UnixMilli(timestamp)

fmt.Println(t) // 2023-12-13 10:00:00 +0000 UTC
```

### time.Time 转 int64

```go
// 获取当前时间的毫秒时间戳
now := time.Now()
timestamp := now.UnixMilli() // Go 1.17+

// 或者手动计算
timestamp := now.Unix()*1000 + int64(now.Nanosecond())/1e6
```

## Prometheus 官方实现

查看 Prometheus 源码可以发现，官方也是使用 `int64` 存储时间戳：

```go
// github.com/prometheus/prometheus/pkg/labels
type Sample struct {
    Point
    Metric labels.Labels
}

type Point struct {
    T int64   // 时间戳（毫秒）
    V float64 // 值
}
```

## 实际应用示例

```go
package main

import (
    "fmt"
    "time"
)

type Sample struct {
    Timestamp int64
    Value     float64
}

func main() {
    // 创建样本
    sample := Sample{
        Timestamp: time.Now().UnixMilli(),
        Value:     42.5,
    }

    // 显示时需要转换为 time.Time
    t := time.UnixMilli(sample.Timestamp)
    fmt.Printf("时间: %s, 值: %.2f\n",
        t.Format("2006-01-02 15:04:05"),
        sample.Value)

    // 比较两个样本的时间先后
    sample2 := Sample{
        Timestamp: time.Now().UnixMilli(),
        Value:     43.0,
    }

    if sample.Timestamp < sample2.Timestamp {
        fmt.Println("sample 早于 sample2")
    }
}
```

## 总结

使用 `int64` 存储时间戳是时序数据库的常见做法，主要考虑：

| 方面     | int64      | time.Time |
| -------- | ---------- | --------- |
| 内存占用 | 8 字节     | 24+ 字节  |
| 序列化   | 直接序列化 | 需要转换  |
| 比较操作 | 直接比较   | 方法调用  |
| 压缩友好 | 是         | 需要转换  |
| 时区处理 | 无需处理   | 需要考虑  |
| 精度     | 毫秒       | 纳秒      |

对于 Mini Prometheus 这样的时序数据库，**性能和内存效率** 是首要考虑因素，因此选择 `int64` 是更合适的设计决策。

## 参考资料

- [Prometheus TSDB Design](https://github.com/prometheus/prometheus/tree/main/tsdb)
- [Go time package](https://pkg.go.dev/time)
- [Unix Time](https://en.wikipedia.org/wiki/Unix_time)
