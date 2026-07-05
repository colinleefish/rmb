# 什么是 rmb？

**rmb** 是面向 AI 智能体对话的**长期记忆**存储。

它位于你的 Agent（Cursor、Claude Code 等）与 PostgreSQL 之间：

1. **采集** — Hook 在每轮结束后触发；`rmb hook-submit` 上传原始对话。
2. **蒸馏** — 后台 Worker 抽取事实、聚合成场景、rollup 为跨会话长期记忆。
3. **召回** — `rmb search` 混合向量与关键词，让下一次会话找到你已说过的话。

设计目标不是「让模型在当前聊天里更聪明」，而是**跨聊天、跨工具携带 durable 知识**——配置、人物、偏好、决策——无需你反复说明。

## 设计原则

### 采集侧工具无关

只要能跑 Shell Hook 并 POST JSON，就受支持。Agent 无需加载 SDK，也无需为摄入改 system prompt。

### 产物可检视

没有藏在 embedding 里的不透明块。每条蒸馏结果都有 URI（`rmb://…`）、元数据，以及回到源轮次的溯源链。

### 运维简单

单一 Go 二进制 + PostgreSQL（+ pgvector）。Worker 是轮询 pipeline 状态的 goroutine，无需独立队列集群。

### 默认追加式整合

Worker **插入**新行，不静默改写历史。人工**纠正**覆盖机器蒸馏的事实，且永远优先。见[人工纠正](/zh/guide/corrections)。

## rmb 不是什么

- **任务内短期记忆** — 压缩当前聊天上下文是另一个问题。
- **多租户 SaaS** — 当前是个人 / 单用户部署。
- **替代 Agent 上下文窗口** — 召回是补充，不会把整库灌进每次 prompt。

## 延伸阅读

- [URI 方案](/zh/concept/uri-scheme) — 扁平 scope 与溯源
- [金字塔（T0–T3）](/zh/concept/pyramid) — 核心心智模型
- [数据如何流动](/zh/concept/pipeline) — Hook → Worker → 召回
- [完整设计文档](/zh/design/l0-l3) — 目标、URI、存储布局
