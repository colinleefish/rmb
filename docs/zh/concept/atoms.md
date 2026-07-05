# 原子（Atom）

**原子**是 **T1**：从一条或多条轮次中抽出的、带类型的小事实。

原子是第一层**可检索**的蒸馏——轮次是证据，原子是结构化事实。

## 存什么

| 字段 | 用途 |
|------|------|
| `content` | 事实正文 |
| `category` | `profile` \| `preferences` \| `entities` \| `events` |
| `priority` | 0–100 重要度（80+ 为关键） |
| `scene_name` | T2 场景切分的分组提示 |
| `slug` | 可选——带 slug 的类别路由到 T3 |
| `source_turn_ids` | 来自哪些 T0 轮次 |
| `session_id` | 所属会话（`rmb meta`） |

```bash
rmb cat rmb://atoms/<uuid>
rmb meta rmb://atoms/<uuid>
```

## URI

`rmb://atoms/<uuid>` — 不透明 UUID。内容仅能通过显式去重/合并变更，Worker 不得静默改写。

## 与 T3 共用四类

| 分类 | 原子示例 |
|------|----------|
| `profile` | 「Colin 住在北京。」 |
| `preferences` | 「偏好简短回复。」 |
| `entities` | 「财务 Lisa 负责对接。」 |
| `events` | 「2026-05-17：选定 Postgres-only 存储。」 |

T3 rollup 按 category 写入 `rmb://profile`、`rmb://preferences/<slug>` 等。

## 默认追加

Worker 默认 **INSERT** 新原子；近重复只打标或降权，不静默 merge。`events` 一律只插入。

## 延伸阅读

- [轮次（Turn）](/zh/concept/turns)
- [场景（Scenes）](/zh/concept/scenes)
- [金字塔](/zh/concept/pyramid)
