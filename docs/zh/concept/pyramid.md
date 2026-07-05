# 金字塔（T0–T3）

rmb 用四个**层级**组织知识。自下而上蒸馏程度递增：底层是原始聊天，顶层是 durable 事实。

```
                    ┌─────────────────────────────────────┐
                    │  T3 — 记忆（跨 session）             │
                    │  profile · preferences · entities   │
                    └──────────────────▲──────────────────┘
                                       │
         ┌─────────────────────────────┴─────────────────────────────┐
         │  会话 · 在单个会话内                                         │
         │    轮次 (T0) → 原子 (T1) → 场景 (T2)                        │
         └───────────────────────────────────────────────────────────┘
```

## T0 — 轮次（Turn）

|          |                                    |
| -------- | ---------------------------------- |
| **含义** | 一次完整的 user + assistant 原始对 |
| **表**   | `session_turns`                    |
| **URI**  | `rmb://turns/<uuid>`               |
| **来源** | Hook 采集（`POST …/upload`）       |
| **数量** | 每个 session 多个                  |

T0 是**只追加的证据**。Worker 不得改写轮次内容。

## T1 — 原子（Atom）

|          |                              |
| -------- | ---------------------------- |
| **含义** | 从轮次抽出的、带类型的小事实 |
| **表**   | `atoms`                      |
| **URI**  | `rmb://atoms/<uuid>`         |
| **来源** | T1 Worker（LLM 抽取）        |
| **数量** | 每个 session 若干            |

每个原子包含：

- `category` — `profile` | `preferences` | `entities` | `events`
- `priority` — 0–100 重要度
- `scene_name` — 属于哪段对话场景
- `source_turn_ids` — 溯源到 T0

原子与长期记忆（T3）**共用四类分类法**，抽取层与存储层同名，无需路由翻译。

## T2 — 场景（Scene）

|          |                                                       |
| -------- | ----------------------------------------------------- |
| **含义** | 一个 session 内连贯的「我们在做什么」片段             |
| **表**   | `scenes`                                              |
| **URI**  | `rmb://scenes/<uuid>`                                 |
| **来源** | T2 Worker（按原子分组后 LLM 写叙事）                  |
| **数量** | **每个 session 数个**（不是每轮一个，也不是整场一个） |

场景是会话内的叙事胶水。创建过程见[场景](/zh/concept/scenes)。

## T3 — 记忆（Memory）

|          |                                                |
| -------- | ---------------------------------------------- |
| **含义** | 跨 session 的 durable 知识                     |
| **表**   | `memories`                                     |
| **URI**  | `rmb://profile`、`rmb://preferences/<slug>` 等 |
| **来源** | T3 Worker（对场景 rollup）                     |
| **数量** | 有界 — `profile` 单例；其余按 slug 多条        |

见[长期记忆](/zh/concept/memories)。

## Session 不是一层

`sessions` 行是一次对话的**容器**：

- 持有 T0 轮次，并关联 session 内的原子 / 场景
- T2 完成后写入可检索的 `abstract`
- URI：`rmb://sessions/<sid>`

其「正文」是按时间排序的轮次列表（`rmb tree rmb://sessions/<sid>/`）。

## 召回方向

检索通常从**上**往下钻：

```
memory → scenes → atoms → turns
```

`rmb meta <uri>` 在各步展示 `source_*_uris`，便于对照原始证据。

## 切面（abstract vs body）

T2 场景与 T3 记忆有两列文本：

| 切面       | 预算            | 用途                   |
| ---------- | --------------- | ---------------------- |
| `abstract` | ~100 tokens     | 向量嵌入、快速过滤     |
| `body`     | 无上限 Markdown | 全文检索、rerank、展示 |

轮次与原子本身很短，内容直接存列，无需单独 abstract。

## 延伸阅读

- [URI 方案](/zh/concept/uri-scheme) — 扁平 scope、容器、溯源
- [实体模型](/zh/reference/entity-model) — 表、链接、实现状态
- [设计：两轴模型](/zh/design/l0-l3#_4-两轴模型) — 层级与切面完整说明
