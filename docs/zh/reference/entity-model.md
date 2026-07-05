# 实体模型 — 各部件如何关联

> 各表是什么、聊天如何流入长期记忆的简明指南。  
> 完整设计：[L0→L3 知识蒸馏](/zh/design/l0-l3)。

## 一图流

```
采集：  hook → POST /upload → sessions + session_turns (T0)

蒸馏：  turns → T1 → atoms → T2 → scenes → T3 → memories
        scenes 同时刷新 sessions.abstract

运维：  sessions ↔ pipeline_state；Worker ↔ tasks
```

**真理方向：** 对话沿金字塔**向上**流动（原始 → 事实 → 场景 → 长期记忆）。每一层都保留指回来源的指针。

## 金字塔（T0 → T3）

| 层级 | 表 | 含义 | 父级 |
|------|-----|------|------|
| **Session** | `sessions` | 一次 Agent 对话（Cursor / Claude 的 session UUID） | — |
| **T0** | `session_turns` | 一次 user + assistant 交换 | `sessions` |
| **T1** | `atoms` | 从轮次抽出的小事实 | `sessions` |
| **T2** | `scenes` | 一个 session 内「我们在做什么」，由原子构成 | `sessions` |
| **T3** | `memories` | 跨 session 的 durable 知识 | 全局 |

`sessions` 不是独立一层：它是 T0 轮次与 session 内 T1/T2 的**容器**，T2 后还有短 `abstract`。

## 典型数量

```text
1 session
  ├── 多个 session_turns
  ├── 多个 atoms
  ├── 数个 scenes
  └── 0..1 pipeline_state

跨 session 的 scenes
  └── rollup 为 memories（profile 为单例）
```

## 各实体存什么

### `sessions`

- **标识：** `session_key` = Agent 对话 UUID
- **状态：** `status`（如 `active`）
- **摘要：** `abstract`（新流水线）；`overview_text`（旧 summarizer，退役中）
- **URI：** `rmb://sessions/<session_key>`
- **「正文」：** 无单一 blob — 通过 `rmb tree rmb://sessions/<id>/` 列出扁平 turn URI

### `session_turns` (T0)

- 一行 = 一次采集的 Q/A（`messages_jsonl`）
- **URI：** `rmb://turns/<uuid>`（`session_turns.id`，uuidv7）。`meta` 含 `session_id`。

### `atoms` (T1)

- 一行 = 一条结构化事实
- **分类：** `profile` | `preferences` | `entities` | `events`
- **`scene_name`：** 属于哪段场景（抽取时填写）
- **URI：** `rmb://atoms/<uuid>`（不透明 id；去重时内容可变）。`meta` 中的 `session_id` 指向所属 session。

### `scenes` (T2)

- 一行 = session 内一段连贯主题
- **`abstract` + `body`**；`source_atom_uris[]`
- **URI：** `rmb://scenes/<uuid>`

### `memories` (T3)

- 逻辑 URI = 四类长期记忆之一
- **版本化：** `superseded_at IS NULL` 为活跃行；Worker INSERT + supersede
- **`source_scene_uris[]`** 指向贡献场景

### `pipeline_state` / `tasks`

- 协调异步抽取与可观测任务，**不是记忆层级**。

## 层级如何串联（溯源）

| 从 | 到 | 字段 |
|----|-----|------|
| T0 | T1 | `atoms.source_turn_ids` |
| T1 | T2 | `scenes.source_atom_uris` |
| T2 | T3 | `memories.source_scene_uris` |

召回可**向下**走：memory → scenes → atoms → turns。

## 仓库现状

| 组件 | 状态 |
|------|------|
| `sessions`, `session_turns` | **已上线** |
| `atoms` | **已上线** — T1 Worker |
| `scenes` | **已上线** — T2 Worker |
| `memories` | **已上线** — T3 Worker |
| `/ui/` 观测 | **已上线** |

详见[实施计划](/zh/reference/plan)。

## 参见

- [URI 方案](/zh/concept/uri-scheme) — 扁平 scope、`tree`、溯源
- [URI 方案 §5](/zh/design/l0-l3#_5-uri-方案)
- [Worker 触发 §8–9](/zh/design/l0-l3#_8-流水线草图)
