# 会话（Session）

**会话**指一次 Agent 对话——Cursor 或 Claude Code 里由 session UUID 标识的那场聊天。

它**不是金字塔的一层**，而是把 turn、session 内的 atom 与 scene 归到同一 `session_id` 下的**容器**。

## 存什么

| 字段 | 用途 |
|------|------|
| `session_key` | Agent 传来的对话 UUID |
| `abstract` | 可检索的短摘要——T2 场景完成后刷新 |
| `status` | 会话生命周期（`active` 等） |

没有单独的 `body` 列。会话的「正文」是按时间排列的轮次，以扁平 URI 列出：

```bash
rmb cat rmb://sessions/<sid>           # 仅 abstract
rmb tree rmb://sessions/<sid>/         # 扁平 rmb://turns/… 与 rmb://atoms/…
rmb meta rmb://sessions/<sid>
```

## URI

`rmb://sessions/<sid>` — 会话实体本身。

`rmb://sessions/<sid>/` — 容器视图（末尾 `/`）。Turn 与 atom **不**嵌在路径里，见 [URI 方案](/zh/concept/uri-scheme)。

## 与轮次的区别

| | 会话 | 轮次（T0） |
|---|------|------------|
| 范围 | 整场对话 | 一次 user + assistant 交换 |
| URI | `rmb://sessions/<sid>` | `rmb://turns/<uuid>` |
| 层级 | 容器（非 T0–T3） | T0 — 只追加的证据 |
| 数量 | 每场聊天一条 | 每会话多条 |

**为何不用「对话」指 turn？** 中文里「对话」常指整场聊天；**轮次**专指其中一轮问答，与代码里的 `turn` 一致。

## 延伸阅读

- [轮次（Turn）](/zh/concept/turns)
- [金字塔](/zh/concept/pyramid)
- [URI 方案](/zh/concept/uri-scheme)
