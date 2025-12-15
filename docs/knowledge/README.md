# Mini Prometheus 知识点文档

本目录用于记录 Mini Prometheus 项目开发过程中的技术问题、设计决策和知识点总结。

## 📚 文档列表

### 数据模型设计

1. [为什么 Sample 中 Timestamp 使用 int64 而不是 time.Time](./01-timestamp-design.md)
   - 性能和内存效率分析
   - 序列化和网络传输优势
   - 类型转换方法
   - Prometheus 官方实现参考

2. [Series 和 Sample 的关系：为什么是一对多](./02-series-samples-relationship.md)
   - 时间序列的本质
   - 实际场景示例
   - 查询需求分析
   - 正确的数据结构设计

## 📖 阅读顺序

如果你是第一次阅读，建议按以下顺序：

1. 先阅读 [DESIGN.md](../DESIGN.md) 了解整体设计
2. 阅读数据模型相关的知识点（01, 02）
3. 后续根据开发进度阅读其他主题

## 🏷️ 主题分类

### 数据模型（Data Model）
- [01-timestamp-design.md](./01-timestamp-design.md)
- [02-series-samples-relationship.md](./02-series-samples-relationship.md)

### 存储引擎（Storage）
- 待补充...

### 查询引擎（Query Engine）
- 待补充...

### 数据抓取（Scraping）
- 待补充...

### 性能优化（Performance）
- 待补充...

## 💡 如何使用

1. **学习阶段**: 按主题分类阅读相关文档
2. **开发阶段**: 遇到问题时查阅相关知识点
3. **复习阶段**: 定期回顾已学习的内容

## 📝 文档规范

每篇文档包含以下结构：
- **问题**: 清晰描述要解决的问题
- **分析**: 详细的技术分析和对比
- **示例**: 代码示例和实际场景
- **总结**: 关键要点汇总
- **参考资料**: 相关学习资源

## 🔄 更新日志

- 2024-12-13: 创建知识点文档目录
  - 添加 timestamp 设计文档
  - 添加 Series-Sample 关系文档

## 🤝 贡献

如果发现文档中有错误或需要补充的内容，欢迎提出建议！

