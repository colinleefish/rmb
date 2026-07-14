# 长期记忆

**T3 记忆**是**跨 session** 蒸馏出的 durable 知识。当你问「rmb 对我了解多少？」时，Agent 通常需要的是这一层。

## 四类

T1 原子与 T3 记忆共用同一分类法：

| 类别 | URI 形态 | 数量 | 内容 |
|------|----------|------|------|
| **profile** | `rmb://profile` | 单例 | 关于用户的稳定事实 — 所在地、角色、健康、核心特质 |
| **preferences** | `rmb://preferences/<slug>` | 多条 | 长期倾向 — 技术选型、工作习惯、**对 AI 的行为规则** |
| **entities** | `rmb://entities/<slug>` | 多条 | 第三方 — 人、项目、主机、公司 |
| **events** | `rmb://events/<slug>` | 多条 | 带日期的决策与里程碑 — **只追加、不可变** |

### profile

逻辑上是一份「用户是谁」的文档。无 slug，固定 `rmb://profile`。

> 「Colin 住在北京。」·「对花生过敏。」

### preferences

反复出现的「偏好 X / 想要 X / 总是 Y」——包括对 AI 助手的行为要求。

> `rmb://preferences/go-services` — 「偏好单二进制 Go 服务。」  
> `rmb://preferences/answer-length` — 「偏好简短回答。」

`instruction` 类规则归入 **preferences**，不设第五类。

### entities

**不是用户本人**的、有名字的持久事物。

> `rmb://entities/jenkins` — 家目录、定时器迁移等  
> `rmb://entities/yao-qiankun` — 同事信息

slug 是事物本身的**规范名**（`jenkins`，而非 `fix-jenkins-timer`）。

### events

不可变里程碑。slug 常带日期前缀。

> `rmb://events/2026-05-17-postgres-only-decision`

Worker **只 INSERT**，events 不 merge、不原地改写。

## 记忆如何创建

**T3 Worker**（全局互斥）：

1. 收集 `t3_status` pending 的 session（或定时 rollup）
2. 加载**变化的场景**，按 category + slug 路由原子
3. LLM 蒸馏为目标 memory URI 的 `abstract` + `body`
4. **INSERT** 新版本行；将上一活跃行标为 `superseded_at`
5. 禁止原地 `UPDATE body` — 版本化保留审计轨迹

## 版本化

多条物理行可共享同一逻辑 URI。**活跃**行：`superseded_at IS NULL`。

```bash
rmb cat rmb://profile
rmb meta rmb://profile
```

## 人工纠正

机器蒸馏可能出错。用户可对 memory URI 附加**纠正** — durable 覆盖层，**永远优先**于蒸馏正文。

```bash
rmb correction add rmb://entities/jenkins "Home is /var/lib/jenkins, not /opt"
```

见[人工纠正](/zh/guide/corrections)。

## 优先级

原子带 `priority`（0–100，或 `-1` 表示「永不丢弃」）：

| 区间 | 含义 |
|------|------|
| 80–100 | 关键 — 健康、禁忌、核心特质、严格规则 |
| 50–79 | 普通 |
| &lt; 50 | 弱信号 — 可降权 |
| -1 | 哨兵 — 绝对行为规则（慎用） |

## 召回

```bash
rmb search "jenkins home directory"
rmb cat rmb://entities/jenkins
```

溯源：`meta` → `source_scene_uris` → 场景 body → 原子 URI → 轮次。

## 延伸阅读

- [Agent 用 CLI](/zh/guide/cli-for-agents)
- [设计 §6 分类法](/zh/design/l0-l3#_6-记忆分类法-t1-与-t3-共用)
- [整合策略评述](/zh/design/consolidation) — 为何默认追加
