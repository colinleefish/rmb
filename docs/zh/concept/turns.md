# 轮次（Turn）

**轮次**是 **T0**：一次完整的 user + assistant 原始交换，由 Hook 原样采集。

轮次是**只追加的证据**。Worker 不得改写 `messages_jsonl`。

## 采集

1. 每轮 Agent 回复后 Hook 触发
2. `rmb hook-submit` POST 到 `POST /api/v1/sessions/:id/upload`
3. 服务器写入 `session_turns` 行，返回 `rmb://turns/<uuid>`

URI 使用行的 uuidv7 `id`，不是从 0 开始的序号。

## 存什么

| 字段 | 用途 |
|------|------|
| `messages_jsonl` | 原始 user + assistant 消息 |
| `session_id` | 所属会话（`rmb meta` 里也有） |
| `turn_status` | 旧版逐轮 summarizer 状态（退役中） |

```bash
rmb cat rmb://turns/<uuid>    # 原始 messages_jsonl
rmb meta rmb://turns/<uuid>
```

## URI

`rmb://turns/<uuid>` — 扁平顶级 scope。归属哪条会话在元数据里，不在路径里。

## 向上蒸馏

T1 Worker 读取待处理轮次，抽出 **原子**，并在每个 atom 上记录 `source_turn_ids`。召回向下走：memory → scenes → atoms → **轮次**（地面真相）。

## 延伸阅读

- [会话（Session）](/zh/concept/sessions)
- [原子（Atom）](/zh/concept/atoms)
- [数据如何流动](/zh/concept/pipeline)
