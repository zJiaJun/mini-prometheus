# Mini Prometheus 设计文档

## 项目目标

通过实现一个精简版的 Prometheus，深入理解 Prometheus 的核心工作原理和架构设计。

## 核心功能模块

### 1. 数据模型（Data Model）

#### 1.1 时间序列（Time Series）

- **指标名称（Metric Name）**: 表示被监控的系统特征，如 `http_requests_total`
- **标签（Labels）**: 键值对形式的维度信息，如 `{method="GET", path="/api", status="200"}`
- **样本（Sample）**: 由时间戳和浮点数值组成的数据点
- **时间序列标识**: 指标名称 + 标签集合唯一确定一条时间序列

#### 1.2 数据结构设计

```
Sample {
    Timestamp int64   // Unix时间戳（毫秒）
    Value     float64 // 指标值
}

Label {
    Name  string
    Value string
}

Metric {
    Name   string
    Labels []Label
}

Series {
    Metric  Metric
    Samples []Sample
}
```

---

### 2. 指标类型（Metric Types）

#### 2.1 Counter（计数器）

- **特性**: 单调递增的累计指标
- **用途**: 请求总数、错误总数等
- **操作**: 只能增加或重置为 0
- **示例**: `http_requests_total`

#### 2.2 Gauge（仪表盘）

- **特性**: 可以任意增减的指标
- **用途**: 当前内存使用量、并发请求数等
- **操作**: 可以增加、减少、设置为任意值
- **示例**: `memory_usage_bytes`, `active_connections`

#### 2.3 Histogram（直方图）

- **特性**: 对观察结果进行采样并在可配置的桶中计数
- **用途**: 请求持续时间、响应大小等
- **输出指标**:
  - `<basename>_bucket{le="<上界>"}`：累积计数器
  - `<basename>_sum`：所有观察值的总和
  - `<basename>_count`：观察值总数
- **示例**: `http_request_duration_seconds`

#### 2.4 Summary（摘要）

- **特性**: 类似 Histogram，但在客户端计算分位数
- **用途**: 请求延迟的百分位数
- **输出指标**:
  - `<basename>{quantile="<φ>"}`：观察到的分位数
  - `<basename>_sum`：所有观察值的总和
  - `<basename>_count`：观察值总数

---

### 3. 数据抓取（Scraping）

#### 3.1 拉取模型（Pull Model）

- **原理**: Prometheus 主动从被监控目标拉取指标数据
- **优点**:
  - 集中控制抓取频率
  - 更容易检测目标是否存活
  - 可以手动访问目标进行调试

#### 3.2 抓取流程

1. 从配置文件读取目标列表（静态配置或服务发现）
2. 定期向每个目标的 `/metrics` 端点发起 HTTP GET 请求
3. 解析返回的指标数据（Prometheus 文本格式）
4. 添加时间戳并存储到 TSDB
5. 记录抓取元数据（耗时、样本数等）

#### 3.3 配置结构

```yaml
scrape_configs:
  - job_name: "my-app"
    scrape_interval: 15s # 抓取间隔
    scrape_timeout: 10s # 抓取超时
    static_configs:
      - targets:
          - "localhost:8080"
          - "localhost:8081"
```

#### 3.4 指标格式解析

- **文本格式**: Prometheus exposition format
- **示例**:

```
# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 1027
http_requests_total{method="post",code="400"} 3
```

---

### 4. 时间序列数据库（TSDB）

#### 4.1 存储架构

- **内存存储**: 最近的活跃数据（head block）
- **磁盘存储**: 持久化的历史数据（persisted blocks）
- **数据分块**: 按时间范围分块存储（默认 2 小时一个块）

#### 4.2 数据压缩

- **时间戳压缩**: Delta-of-delta 编码
- **值压缩**: XOR 浮点压缩（Gorilla 算法）
- **标签索引**: 倒排索引加速查询

#### 4.3 核心操作

- **写入（Append）**: 追加新样本到时间序列
- **查询（Query）**: 根据时间范围和标签选择器检索数据
- **压缩（Compaction）**: 合并小块为大块
- **保留（Retention）**: 删除超过保留期的旧数据

#### 4.4 索引设计

- **正向索引**: SeriesID -> Metric + Samples
- **倒排索引**: Label -> SeriesID 列表
- **用途**: 快速定位符合标签选择器的时间序列

---

### 5. 查询引擎（Query Engine）

#### 5.1 PromQL 基础

- **即时查询（Instant Query）**: 返回某个时间点的数据
- **范围查询（Range Query）**: 返回一段时间内的数据
- **选择器（Selector）**:
  - 即时向量选择器: `http_requests_total{job="api"}`
  - 范围向量选择器: `http_requests_total[5m]`

#### 5.2 基本查询操作

- **标签匹配器**:
  - `=`: 精确匹配
  - `!=`: 不等匹配
  - `=~`: 正则匹配
  - `!~`: 正则不匹配
- **示例**: `http_requests_total{method="GET",status=~"2.."}`

#### 5.3 聚合函数

- **sum**: 求和
- **avg**: 平均值
- **max/min**: 最大/最小值
- **count**: 计数
- **示例**: `sum(rate(http_requests_total[5m])) by (status)`

#### 5.4 运算符

- **算术运算**: +, -, \*, /, %
- **比较运算**: ==, !=, >, <, >=, <=
- **逻辑运算**: and, or, unless

#### 5.5 函数

- **rate()**: 计算范围向量的每秒平均增长率（适用于 Counter）
- **irate()**: 计算范围向量的瞬时增长率
- **increase()**: 计算范围向量的增长量
- **delta()**: 计算范围向量的变化量（适用于 Gauge）

---

### 6. HTTP API

#### 6.1 查询 API

- **即时查询**:

  - `GET/POST /api/v1/query`
  - 参数: `query`, `time`
  - 返回指定时间点的查询结果

- **范围查询**:
  - `GET/POST /api/v1/query_range`
  - 参数: `query`, `start`, `end`, `step`
  - 返回时间范围内的查询结果

#### 6.2 元数据 API

- **标签名称**: `GET /api/v1/labels`
- **标签值**: `GET /api/v1/label/<label_name>/values`
- **时间序列元数据**: `GET /api/v1/series`

#### 6.3 目标管理 API

- **目标状态**: `GET /api/v1/targets`
- **显示所有抓取目标及其状态**

#### 6.4 响应格式

```json
{
  "status": "success",
  "data": {
    "resultType": "vector|matrix|scalar|string",
    "result": []
  }
}
```

---

### 7. 配置管理

#### 7.1 全局配置

```yaml
global:
  scrape_interval: 15s # 默认抓取间隔
  evaluation_interval: 15s # 规则评估间隔
  external_labels: # 外部标签
    cluster: "prod"
```

#### 7.2 抓取配置

- 支持静态配置（static_configs）
- 可扩展服务发现（Kubernetes、Consul 等，mini 版本可选）

#### 7.3 配置重载

- 支持热重载：通过 HTTP POST 请求或发送 SIGHUP 信号
- 验证配置文件格式

---

## 实现优先级

### Phase 1: 基础架构（核心中的核心）

1. **数据模型定义**: Metric, Label, Sample, Series
2. **简单的内存 TSDB**: 基于 map 的存储结构
3. **指标类型实现**: Counter 和 Gauge（最基础的两种）
4. **配置文件解析**: 读取 YAML 配置

### Phase 2: 数据采集

1. **HTTP 抓取器**: 从目标拉取指标
2. **Prometheus 文本格式解析器**: 解析 `/metrics` 返回的数据
3. **定时任务调度**: 按配置的间隔定期抓取

### Phase 3: 查询功能

1. **基础 PromQL 解析器**: 支持简单的选择器查询
2. **查询执行引擎**: 从 TSDB 检索数据
3. **HTTP 查询 API**: 提供 REST 接口

### Phase 4: 高级功能

1. **聚合函数**: sum, avg, max, min
2. **rate/increase 函数**: Counter 相关计算
3. **Histogram 和 Summary**: 更复杂的指标类型
4. **持久化存储**: 将数据写入磁盘

### Phase 5: 优化和扩展

1. **数据压缩**: 实现 Gorilla 压缩算法
2. **倒排索引**: 优化标签查询性能
3. **更完整的 PromQL**: 支持更多函数和运算符
4. **Web UI**: 简单的查询界面

---

## 技术要点

### 1. 并发处理

- 多个目标并发抓取
- 使用 goroutine 和 channel
- 控制并发数量，避免资源耗尽

### 2. 时间处理

- 统一使用 Unix 时间戳（毫秒）
- 对齐抓取时间，避免时钟偏移

### 3. 错误处理

- 抓取失败重试机制
- 记录错误日志
- 标记目标健康状态

### 4. 性能考虑

- 使用高效的数据结构（map、slice）
- 标签排序和规范化（保证唯一性）
- 限制内存中的时间序列数量

### 5. 可测试性

- 单元测试覆盖核心逻辑
- Mock HTTP 服务器测试抓取
- 基准测试验证性能

---

## 项目结构建议

```
mini-prometheus/
├── cmd/
│   └── prometheus/          # 主程序入口
│       └── main.go
├── pkg/
│   ├── model/              # 数据模型
│   │   ├── metric.go
│   │   ├── sample.go
│   │   └── series.go
│   ├── storage/            # 存储引擎
│   │   ├── tsdb.go         # TSDB实现
│   │   ├── memory.go       # 内存存储
│   │   └── index.go        # 索引
│   ├── scrape/             # 数据抓取
│   │   ├── scraper.go      # 抓取器
│   │   ├── target.go       # 目标管理
│   │   └── parser.go       # 指标解析
│   ├── promql/             # 查询引擎
│   │   ├── parser.go       # PromQL解析
│   │   ├── engine.go       # 执行引擎
│   │   └── functions.go    # 内置函数
│   ├── api/                # HTTP API
│   │   ├── v1/
│   │   │   ├── query.go
│   │   │   └── metadata.go
│   │   └── server.go
│   └── config/             # 配置管理
│       ├── config.go
│       └── loader.go
├── docs/
│   └── DESIGN.md          # 本设计文档
├── go.mod
├── go.sum
└── README.md               # 项目说明
```

---

## 学习收获

通过实现这个 mini Prometheus，你将深入理解：

1. **时间序列数据库的设计原理**: 如何高效存储和查询时间序列数据
2. **监控系统的架构**: 拉取模型 vs 推送模型的优劣
3. **数据压缩算法**: Gorilla 算法等针对时序数据的压缩技术
4. **查询语言设计**: PromQL 的设计思想和实现方式
5. **高性能 Go 编程**: 并发、内存管理、性能优化等实践
6. **分布式系统概念**: 标签、维度、聚合等核心概念

---

## 参考资料

- [Prometheus 官方文档](https://prometheus.io/docs/)
- [Prometheus GitHub 仓库](https://github.com/prometheus/prometheus)
- [Gorilla 时序压缩论文](http://www.vldb.org/pvldb/vol8/p1816-teller.pdf)
- [Writing a Time Series Database from Scratch](https://fabxc.org/tsdb/)
- [PromQL 教程](https://prometheus.io/docs/prometheus/latest/querying/basics/)

---

## 开发建议

1. **从简单开始**: 先实现内存存储，再考虑持久化
2. **迭代开发**: 按 Phase 分步实现，每个阶段都能运行
3. **多写测试**: 时序数据处理容易出错，测试很重要
4. **参考源码**: 遇到困难时查看 Prometheus 源码
5. **性能测试**: 定期进行性能测试，及时发现瓶颈

---

**注**: 这是一个教学项目，重点在于理解原理。生产环境请使用官方 Prometheus。
