# 场景（Scenes）

**场景**是 T2：在**一个 session** 内的一段连贯工作。

例如：「调试 Cursor Hook」「讨论 L0–L3 设计」「规划生产部署」。

## 一个 session，多个场景

长对话常覆盖多个主题。rmb 期望**每个 session 有数个场景**，既不是整场一个，也不是每轮一个。

```
1 session
  ├── 多个 turns
  ├── 多个 atoms
  └── 数个 scenes        ← 通常 2–5 段
```

## 场景如何创建

分**两步**流水线。

### 第一步 — T1 给原子打 `scene_name`

T1 Worker 运行时，**一次 LLM 调用**同时做原子抽取与场景切分：

- 每个原子带短标签 `scene_name`（如 `people`、`infra`、`decisions`）
- 此时只是分组提示，尚未写入 scene 行
- T1 将 `pipeline_state.t2_status = pending`

### 第二步 — T2 生成场景行

T2 Worker 在短延迟后运行（`delay_after_t1`，默认约 90s）：

1. 加载该 session **全部**原子
2. **按 `scene_name` 分组**（无名称 → `"General"`）
3. **LLM** 输出 `display_name`、`abstract`、`body`、`atom_uris`
4. **Upsert** 场景行（URI 由 session + 名称稳定派生）
5. **修剪** 不再存在的场景段
6. 用场景 abstract 刷新 `sessions.abstract`
7. 标记 T3 pending，等待跨会话 rollup

1. Hook 追加 turn (T0)
2. T1 Worker 抽取 atoms 并设置 `scene_name`；`t2_status = pending`
3. T2 Worker 分组 atoms 写入 scenes，并刷新 `sessions.abstract`

## 场景存什么

| 字段 | 用途 |
|------|------|
| `display_name` | 人类可读标签（如「Hook 调试」） |
| `abstract` | ~100 tokens，向量检索 |
| `body` | Markdown 叙事 |
| `source_atom_uris[]` | 由哪些原子构成 |
| `session_id` | 始终绑定一个 session |

## 场景 vs 记忆

| | 场景 (T2) | 记忆 (T3) |
|---|-----------|-----------|
| 范围 | 单个 session | 跨 session |
| 回答问题 | 「这场聊天在干什么？」 | 「关于用户我长期知道什么？」 |
| 分类 | 无 — 叙事，非类型化 | profile / preferences / entities / events |
| `rmb search` | 是（`scene` 层） | 是（`memory` 层） |

场景是**会话内事实**与 **durable 知识**之间的桥梁。T3 读取变化的场景，蒸馏为记忆行。

## 重建行为

有新原子时，T2 会**按当前全部原子重建**该 session 的场景，而非按原子增量追加。URI 稳定，内容刷新。

手动触发：在服务器上 `rmb t2 backfill`。

## 延伸阅读

- [金字塔](/zh/concept/pyramid) — 场景在 T0–T3 中的位置
- [数据如何流动](/zh/concept/pipeline) — Worker 触发与节奏
- [设计文档 §8 流水线](/zh/design/l0-l3#_8-流水线草图)
