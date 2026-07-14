# 数据如何流动

从 Hook 触发到 Agent 召回一条事实的端到端路径。

```
Hook → hook-submit → POST /upload → session_turns (T0)
                                      ↓
                    T1 Worker → atoms (T1) → T2 Worker → scenes (T2)
                                      ↓                        ↓
                    T3 Worker → memories (T3)          sessions.abstract
                                      ↓
                    embed Worker → search API → rmb CLI / Agent
```

## 阶段 1 — 采集（毫秒级）

每轮结束后 Hook 运行。`rmb hook-submit`：

1. 解析 Cursor 或 Claude Code 载荷
2. 配对 user + assistant 消息
3. `POST /api/v1/sessions/:id/upload`
4. 立即返回 **202** 与 turn URI

热路径上无 LLM。

## 阶段 2 — T1 抽取

**触发**（每 session）：

- 每 N 轮（`extraction.every_n`，默认 8）
- 空闲超时（`extraction.idle_seconds`，默认 600）
- 预热爬坡（新 session：2 → 4 → 8 → …）

**动作**：读 pending T0 → 一次 LLM 切分+抽取 → INSERT 原子 → `t2_status = pending`

## 阶段 3 — T2 场景

**触发**：`t2_status` pending/failed、T1 未运行、`delay_after_t1` 已过（默认 90s）

**动作**：按 `scene_name` 分组 → LLM 写 abstract/body → upsert 场景 → 刷新 `sessions.abstract` → `t3_status = pending`

## 阶段 4 — T3 记忆

**触发**：任一 session `t3_status` pending，或定时轮询（默认 15m）

**动作**：收集变化场景 → 按类/slug 蒸馏 → INSERT + supersede

## 阶段 5 — 嵌入

Embed Worker 在 `abstract` 变化时为原子、场景、活跃记忆填充 `vector(1024)`。

## 阶段 6 — 召回

| 命令 | 检索范围 |
|------|----------|
| `rmb search` | 记忆 + 场景混合（向量 + FTS） |

CLI 通过 `RMB_URL` 调用服务端 HTTP API。

## 协调

通过 Postgres：`pipeline_state`、`tasks`、session 级 advisory lock。重启安全，无需内存调度器。

## 延伸阅读

- [快速开始](/zh/guide/getting-started)
- [实施计划](/zh/reference/plan)
- [设计：流水线草图](/zh/design/l0-l3#_8-流水线草图)
