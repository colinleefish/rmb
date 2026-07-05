---
layout: home

hero:
  name: rmb
  text: 跨越会话的记忆
  tagline: 采集每一次智能体对话，蒸馏为事实与场景，跨工具召回——无需在 Agent 内嵌 SDK。
  actions:
    - theme: brand
      text: 了解理念
      link: /zh/concept/
    - theme: alt
      text: 快速开始
      link: /zh/guide/getting-started

features:
  - icon: 🪝
    title: 工具无关的采集
    details: Cursor 与 Claude Code 通过 Hook 将原始轮次 POST 到单一 Go 服务。除 Hook 配置外无需改动 Agent 运行时。
  - icon: 🔺
    title: 四层蒸馏
    details: 轮次 → 原子 → 场景 → 长期记忆。每一层都有 URI，可用 CLI 检视。
  - icon: 🔍
    title: 混合召回
    details: 摘要向量检索 + 正文全文检索 + 融合排序，兼顾事实与会话语境。
  - icon: 🗄️
    title: Postgres 原生
    details: 单一二进制 + 单一数据库。无文件树产物，一切落在可查询的列里。
---

## 问题

AI 智能体会遗忘。会话上下文会重置。每次新开对话，你都要重新解释服务器、偏好和过往决策。

**rmb** 是个人记忆服务器：在后台摄入对话、蒸馏 durable 知识，并通过 `rmb search` 暴露给下一次会话，让 Agent 召回你已建立的事实。

## 与常见方案的区别

| 典型「记忆」产品 | rmb |
|---|---|
| 模型无法检视的不透明 blob | 每条事实有 `rmb://` URI；`cat`、`tree`、`meta` 可用 |
| Agent 内嵌 SDK | 仅 Hook + HTTP 上传 |
| 每场聊天一份摘要 | 金字塔——从记忆下钻到原始轮次 |
| 激进合并一切 | 默认追加；人工纠正覆盖机器事实 |

## 状态

T1–T3 Worker（原子、场景、记忆）已上线。生产环境：[rmb.colinleefish.com](https://rmb.colinleefish.com)。路线图见[实施计划](/zh/reference/plan)。
