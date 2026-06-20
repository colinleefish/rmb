# mem9 — L0 → L3 知识蒸馏设计

> 状态：草稿。综合了 TencentDB Agent Memory (TDAI) 与 OpenViking 的经验，结合 mem9 以"工具无关的 hook 采集"为核心的定位。
>
> 英文版见 [`design-l0-l4.md`](./design-l0-l4.md)。
>
> 历史：早期版本提出过 T0 → T4 五层并包含独立的 `identity` 产物。在把"3 类 vs 8 类"合并为统一的 4 类、并把 scope 概念暂时搁置之后，T4 已经没有 T3 `profile` 不能覆盖的职责，金字塔随之缩到 T0 → T3。文档名沿用以保持连续性。

## 1. 目标

1. 将 Agent 原始对话逐级蒸馏为可检索的知识——从单轮 Q/A，一直到一份能自描述用户的档案。
2. **采集侧保持工具无关**——任何能触发 hook 并 POST JSON 的 Agent 都受支持，Agent 端不引入 SDK。
3. **可观察**——每个产物都有 URI，CLI 可直接拉出，没有不透明格式。
4. **运维简单**——单 Go 二进制 + 单个 Postgres，不维护文件树。

## 2. 暂不在本设计范围内

- 任务内短期记忆 / Mermaid 符号化压缩（TDAI 有，是另一个问题）。
- 按 scope 切分多上下文（work / personal / project-X）。暂用单一命名空间，等到真有需求再回来。
- 独立的 T4 `identity` 层。在分类法收敛 + scope 搁置之后，T3 `profile` 已经承担了"用户自描述"的单例产物职责，再多一层没有独立工作。
- 生产级多租户隔离（OpenViking 有，按需再加）。
- 静态加密。
- 审计 / 回滚基础设施（`memory_diff.json`）。暂未证明值得做，将来需要可以加一张追加表。
- 用文件系统承载产物（`~/.mem9/`）。事实源就是 DB 列。
- 任何需要修改 Agent 运行时的能力。

## 3. 背景：分别吸收什么

| 想法                                                                      | 来源                             | 为何吸收                                          |
| ------------------------------------------------------------------------- | -------------------------------- | ------------------------------------------------- |
| T0 → T3 四层金字塔                                                        | TDAI 的 L0–L3 分层               | 分层正契合"先看顶层、按需下钻"的回忆模式。        |
| 带 `category / priority / scene_name / source_turn_ids` 的 Atom 记录      | TDAI L1（更名）                  | 结构化的 atom 可检索；散文式综述不可检索。        |
| 同一次 LLM 调用同时做情境切分                                             | TDAI                             | 一次调用搞定抽取 + 切分，省钱。                   |
| 每个节点的 abstract / detail facet                                        | OpenViking（裁剪）               | 小 abstract 便宜地做向量召回；rerank 和展示直接用 body。曾考虑的 overview 已舍弃，详见 §4.2。 |
| URI 作为统一寻址层                                                        | OpenViking                       | 用一个命名空间贯通 session、atom、scene、memory。 |
| T1 和 T3 共用一套 4 类分类法——`profile / preferences / entities / events` | 从 OpenViking 8 类用户侧瘦身而来 | 名字自解释；抽取层和存储层同名，无需路由翻译。    |
| 两阶段提交（`task_id` + 异步抽取）                                        | OpenViking                       | Hook 秒回，抽取过程可轮询观察。                   |
| 带 score propagation 的分层检索                                           | OpenViking                       | 检索不要直接用平铺 top-K。                        |
| 每 session 的 idle-debounce + 阈值 + warmup ramp                          | TDAI                             | Hook 触发频繁，不能每轮都掏 LLM 钱。              |
| **工具无关的 hook 采集**                                                  | mem9                           | 这是已有的差异化能力，保留。                      |
| **单 Go 二进制 + Postgres 原生**                                          | mem9                           | 运维上最简单。保留。                              |

## 4. 两轴模型

蒸馏分布在两条正交的轴上。

### 4.1 纵轴 —— **层级（Tiers）** T0 → T3

| 层级            | 含义                                                                                 | 来源                        | 数量级                             |
| --------------- | ------------------------------------------------------------------------------------ | --------------------------- | ---------------------------------- |
| **T0 — Turn**   | 一次完整的 user + assistant 原始对                                                   | Hook 采集                   | 每个 session 多个                  |
| **T1 — Atom**   | 从 T0 中抽出的、带 `category / priority / scene_name / source_turn_ids` 的结构化事实 | LLM 抽取（TDAI 风格，4 类） | 每个 session 若干                  |
| **T2 — Scene**  | 一组 atom 构成的"我们在做什么"片段，渲染为 Markdown                                  | LLM 在 session 内汇总 T1    | 每个 session 数个                  |
| **T3 — Memory** | 跨 session 的长期蒸馏，4 类                                                          | LLM 跨 session 滚动 T2      | 有界（`profile` 单例；其它类多个） |

**Session 本身也是一个带 facet 的节点。** `sessions` 行（T0 turn 的父表）持有一个 `abstract` 列以供检索；它不是独立的一层，而是*一段完整对话的聚合视图*，URI 为 `mem9://sessions/<sid>`。在 T2 完成某 session 的 scenes 之后，作为一个小后置步骤刷新这一列。session 没有独立的 body 列——它的 body 就是它的 turn，通过 `mem9 tree mem9://sessions/<sid>` 访问。

### 4.2 横轴 —— **Facet（每行内部）**

每个聚合行（T2 scenes、T3 memories）带两列文本，对应不同的检索预算：

| Facet        | 列                      | 预算        | 用途                |
| ------------ | ----------------------- | ----------- | ------------------- |
| **abstract** | `abstract text`         | ≈100 tokens | 向量召回 / 一行过滤 |
| **detail**   | `body text`（Markdown） | 无上限      | 完整内容，rerank 和展示都用它 |

Sessions 只持 `abstract`（detail 就是按时间排序的 `session_turns`）。T0 turn 和 T1 atom 不需要 facet——它们本身就很短：`session_turns.messages_jsonl` 和 `atoms.content` 就是它们的内容。

曾经考虑过中间一层 `~1k token` 的 `overview`（仿 OpenViking）。已舍弃：OpenViking 的 overview 之所以有用，是作为目录节点之间的导航说明；mem9 没有这种树形结构，层间下钻靠 `source_*_uris` 外键数组，不靠散文导航。在 mem9 的预期规模下，rerank 直接吃 body 是可以承受的。每行两个视图而非三个，可减少漂移风险和生成成本。如果将来 rerank 成本成为真正瓶颈，可以再补回来。

`abstract` 是真正喂给 pgvector 的列；`body` 走 FTS（`tsvector`）。

## 5. URI 方案

所有东西的统一寻址：

```
mem9://{scope}/{path}
```

### 5.1 公开 scope 与寻址风格

| Scope         | 层级              | 寻址                                                                  | URI 示例 |
| ------------- | ----------------- | --------------------------------------------------------------------- | -------- |
| `sessions`    | session / T0 / T1 | session UUID（来自 agent）；turn 用序号；atom 用 UUID                | `mem9://sessions/<sid>`（session abstract）<br>`mem9://sessions/<sid>/turns/<n>`（T0）<br>`mem9://sessions/<sid>/atoms/<uuid>`（T1） |
| `scenes`      | T2                | UUID；带可选 `display_name`，`mem9 cat` 渲染时使用                  | `mem9://scenes/<scene-uuid>` |
| `profile`     | T3                | 单例，无路径                                                          | `mem9://profile` |
| `preferences` | T3                | **语义 slug**（话题名）；无 slug 时回退 UUID                          | `mem9://preferences/coffee`<br>`mem9://preferences/ai-tone` |
| `entities`    | T3                | **语义 slug**（实体名）；回退 UUID                                    | `mem9://entities/tesla`<br>`mem9://entities/colin-mom` |
| `events`      | T3                | **日期前缀 + slug**；回退 UUID                                        | `mem9://events/2026-05-17-postgres-only-decision` |

一共 6 个公开顶级 scope。不带 scope 键，不设 T4 命名空间。

内部 scope（如 `tasks`、`_backfill`）保留给服务端使用，CLI 默认不可寻址；URI 工具暴露 `allow_internal=true` 开关供服务端代码使用。

### 5.2 URI 规则

- **末尾斜杠 = 容器。** `mem9://sessions/<sid>` 是 session 实体本身（`mem9 cat` 打印它的 `abstract`）；`mem9://sessions/<sid>/` 是它的容器（`mem9 tree` 列出其下的 turn 和 atom）。所有 scope 都遵循同样的约定。
- **短格式。** CLI 接受 `/sessions/abc/turns/0` 和 `sessions/abc/turns/0`，两者都归一化为标准 `mem9://...` 形式。降低输入摩擦；程序生成的 URI 始终用标准形式。
- **Unicode 安全。** CJK / Cyrillic / Latin extended / Hiragana / Katakana / Hangul 都原样保留（不做百分号编码）；`mem9://entities/李广慧` 是合法 URI。其它特殊字符替换为 `_`。每段最多 50 字符。
- **预留未来语法。** 形如 `{namespace:key}`（例如 `{date:today}`）的写法目前一律拒绝为非法，预留出语法空间以便将来加入路径变量模板而不破坏兼容性。
- **slug 禁用值。** slug 不能等于 scope 名（不允许 `mem9://preferences/profile`）。校验时直接报错，而不是悄悄 mangling。

### 5.3 Slug 与稳定 ID

按层选择 semantic 还是 opaque，标准是"该行是否有内在稳定名字"：

| 行                         | 选择该风格的原因 |
| -------------------------- | ----------------- |
| T0 turn（序号）            | turn 是按时间排的；编号就是名字。 |
| T1 atom（UUID）            | 默认追加；仅显式 `mem9 atom merge` 可合并。内容不应被 worker 静默改写。 |
| T2 scene（UUID + `display_name`） | scene 行随 atom 累积而更新；URI 用 UUID 保持稳定，名字单独露出供展示。 |
| T3 `preferences` / `entities`（slug） | 这些天生就是有名字的话题 / 实体。URI 描述*话题*或*身份*，不是当下的内容——所以 body 演化时 slug 保持稳定。 |
| T3 `events`（日期 + slug） | event 按分类规则不可变；日期前缀自然有序。 |

**来源。** LLM 在 T1 抽取的 prompt 里，对 `preferences` / `entities` / `events` 类别的 atom 同时产出 `slug`。T3 直接路由到对应的 `memories` 行。

**稳定性。** slug 一经创建即稳定。改名必须显式人工操作（`mem9 mv <old-uri> <new-uri>`），原子地更新 URI 及所有 `source_*_uris` 引用。禁止因内容变化而自动改名。

**冲突。** `memories` 持有 `UNIQUE (category, slug) WHERE slug IS NOT NULL`。冲突时 T3 worker 追加 `-2`、`-3`…后缀，并打 warning，方便我们发现"LLM 在不同实体上反复撞同一个 slug"的情况。

**空回退。** 如果 LLM 产出的 slug 为空或不可用，该行回退到 UUID 寻址（`mem9://preferences/<uuid>`）。最坏情况是 URI 难看，永远不会出错。

## 6. 记忆分类法（T1 与 T3 共用）

4 类，抽取层（T1）和存储层（T3）同名。T3 路由是机械的：按 category 聚合 atom/scene，**向目标 memory URI 追加新版本**（见 §7 `memories` 版本列），禁止原地改写 `body`。

### 6.1 整合立场（持续整合论文）

对照 arXiv:2605.12978（*Useful Memories Become Faulty When Continuously Updated by LLMs*）与 [`memory-consolidation-review.zh.md`](./memory-consolidation-review.zh.md)：

- **T0（`session_turns`）** 是 append-only 情景证据，任何 worker **不得**改写。
- **抽象层（T1–T3）** 由 LLM 生成，默认 **Retain**：新事实优先 **插入新行**；整合（merge / 覆盖 body）必须是 **稀疏、显式、可观测** 的动作，而不是 worker 默认分支。
- **T1 dedup 中的 merge 是受控、稀疏的，不是 worker 默认行为。** 默认：embedding top-K 仅用于打 `near_duplicate_of` 或降权，**插入新 atom**；merge 仅通过 `mem9 atom merge`（或等价人工任务）触发。
- **`events`（T1 与 T3）** 一律只追加、不 merge、不删除、不原地更新。
- **T3 `profile` / slug 行** 禁止 in-place 更新 `body`；每次 rollup **INSERT 新版本**，旧行 `superseded_at` 置位；读取与检索默认 `WHERE superseded_at IS NULL`。
- **漂移检测：** `mem9 eval`（§12）在 T3 rollup 后对比「T0+FTS」与「全栈」召回，差值转负即报警。

| 分类          | T1 worker                         | T3 worker                                      | 捕获什么                                                             | 例子                                                                       |
| ------------- | --------------------------------- | ---------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `profile`     | 插入 atom；不 merge               | 追加新版本（逻辑 URI `mem9://profile`）      | 稳定的身份属性——基本信息、健康 / 禁忌、核心特质                      | "Colin 住在北京。" "对花生过敏。"                                          |
| `preferences` | 插入 atom；不 merge               | 按 slug 追加新版本                             | 反复出现的"更喜欢 X / 想要 X / 习惯这样做"，**包括对 AI 的行为规则** | "偏好单二进制 Go 服务。" "回复永远要短。"                                  |
| `entities`    | 插入 atom；不 merge               | 按 slug 追加新版本                             | 第三方：人、项目、公司、地点                                         | "财务 Lisa 喜欢用邮件。" "Tesla 总部在 Austin，代码 TSLA。"                |
| `events`      | **只插入**；禁止 merge / 更新     | **只插入**；禁止 merge / 更新 / supersede 旧行 | 带时间的事实、决定、里程碑——历史记录，不可变                         | "2026-05-17：mem9 选择 Postgres-only 存储（弃用 ~/.mem9/ 文件方案）。" |

**TDAI 的 `instruction` 并入 `preferences`。** "回复永远要短" 在使用语义上就是一条对 AI 行为的偏好。如果未来需要区分 AI 行为规则和日常生活偏好（例如系统 prompt 只想加载行为规则），在 `preferences` 上加一列 `subkind text`，取值 `lifestyle | ai-behavior`，而不是再多开一个分类。

`priority` 语义（沿用 TDAI，4 类通用）：

- 80–100：关键（健康 / 禁忌 / 核心特质，重要事件 / 计划，严格规则）。
- 50–79：普通。
- < 50：信号弱，候选丢弃或降权。
- `-1`：哨兵值，"绝不丢弃"（只用于绝对的行为规则，谨慎使用）。

## 7. 存储布局

全部落 Postgres，不维护文件树。

| 表                   | 层级 | 关键列                                                                                                                                                  |
| -------------------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `sessions`           | session 聚合 | （已有，扩展）session 元数据；**新增** `abstract text` 与 `embedding vector(1024)`，由 T2 后置步骤刷新                                          |
| `session_turns`      | T0   | （已有）`messages_jsonl text`                                                                                                                           |
| **`atoms`**          | T1   | `uri`、`session_id`、`category`（4 值 CHECK）、`priority int`、`scene_name`、`slug text?`（带 slug 类别的 atom 携带，T3 直接路由用）、`content text`、`source_turn_ids uuid[]`、`embedding vector(1024)`、时间戳 |
| **`scenes`**         | T2   | `uri`、`session_id`、`display_name text?`、`abstract text`、`body text`、`source_atom_uris text[]`、`embedding vector(1024)`、**`version int`**、**`superseded_at timestamptz?`**、时间戳（见 §7.1；worker 默认追加版本而非原地改 body） |
| **`memories`**       | T3   | **`id uuid` PK**、`uri`（逻辑 URI，稳定）、`category`、`slug text?`、**`version int`**、**`superseded_at timestamptz?`**、`abstract text`、`body text`、`source_scene_uris text[]`、`embedding vector(1024)`、时间戳；**活跃行** `UNIQUE (uri) WHERE superseded_at IS NULL`；slug 活跃行 `UNIQUE (category, slug) WHERE slug IS NOT NULL AND superseded_at IS NULL`（迁移草图：`00003_memories_versioning.sql`） |
| **`pipeline_state`** | —    | `session_id`、`t1_status`、`t1_advanced_at`、`t2_status`、`t2_advanced_at`、`t3_status`、`t3_advanced_at`、`warmup_threshold int`                       |
| **`tasks`**          | —    | `id`、`kind`（`t1` / `t2` / `t3` / `backfill`）、`status`、`progress`、`result_uri`、`error`、`session_id?`、时间戳                                     |

T0–T2 仍以 `uri text primary key`；**`memories` 以 `id uuid` 为主键**（同一逻辑 `uri` 可有多版本行）。`body text` 列建 FTS；`embedding` 列为 `vector(1024)`。

### 7.1 `memories` 版本化（实施前迁移）

Phase A 的 `memories` 表以 `uri` 为 PK，**在 Phase D（T3 worker）之前** 应用 `00003_memories_versioning.sql`：

- 增加 `id uuid`、`version int`、`superseded_at timestamptz`。
- 逻辑 URI 仍对用户可见（`mem9 cat mem9://profile` → 最新 `superseded_at IS NULL` 的行）。
- Worker **禁止** `UPDATE … SET body = …`；rollup 一律 `INSERT` + 将上一活跃行 `superseded_at = now()`。
- `mem9 cat` / 检索 / `mem9 meta` 默认只读活跃行；`mem9 cat <uri> --version=N` 或 `--all-versions` 用于审计（CLI 随 Phase D 落地）。

`scenes` 可选同样模式（T2 频繁改 body 时有同类风险）；Phase C 可先原地更新，若 eval 显示漂移再升为版本化。

观察用 CLI：

- `mem9 cat <uri>` —— 打印该行的 `body`（T0 打印 `messages_jsonl`）。
- `mem9 tree <uri-prefix>` —— 列出子 URI。
- `mem9 meta <uri>` —— 打印元数据。

## 8. 采集流程（两阶段）

### 阶段一 —— 同步

```
POST /api/v1/sessions/:id/upload
  → 写入 T0 turn 行
  → 设置 pipeline_state.t1_status = 'pending'
  → 返回 202 { task_id, turn_uri }
```

### 阶段二 —— 异步（worker）

```
T1 worker（按 session）
  触发：(自上次 T1 以来的 turn 数 >= everyN)
       或 (idle 达 idle_seconds)
       或 (warmup：2 → 4 → 8 → … → everyN)
  动作：读未处理 T0 turns
       → 一次 LLM 调用：情境切分 + atom 抽取（TDAI 提示词，4 类）
       → 与既有 atom 比对（embedding top-K）：默认 **INSERT 新 atom**
           · 近重复：写 metadata（如 `near_duplicate_uri`）或降权，**不** LLM merge
           · `events` 类别：永远 INSERT，不参与 dedup merge
           · merge 仅 `mem9 atom merge <uri-a> <uri-b>`（显式、稀疏）
       → 设置 pipeline_state.t2_status='pending'

T2 worker（按 session）
  触发：向下单向定时器
        fire = max(now + delay_after_t1, last_t2 + min_interval)
        硬上限 last_t2 + max_interval
  动作：读 session 内有变化的 atom
       → LLM 调用：生成 scene（abstract + body）
       → 默认 **追加 scene 版本**（或 atom 增量足够大时才改写；见 §7.1）
       → 后置：基于该 session **活跃** scene abstract 刷新
         sessions.{abstract, embedding}
         （短 LLM 调用；scene 很少时也可纯模板拼接）
       → 设置 pipeline_state.t3_status='pending'

T3 worker（全局互斥）
  触发：任一 session 的 t3_status='pending'
       或 定期 rollup
  动作：收集变化的 scenes
       → LLM 调用：蒸馏到分类记忆行（abstract + body）
       → **INSERT 新 memories 行** + supersede 同 URI 的上一活跃行（禁止 UPDATE body）
       → 可选：触发 `mem9 eval`（见 §12）
```

## 9. 触发与节制

| 层级 | 触发                                       | 配置项                                                                          |
| ---- | ------------------------------------------ | ------------------------------------------------------------------------------- |
| T1   | every-N turn + idle 计时 + warmup ramp     | `extraction.every_n=8`、`extraction.idle_seconds=600`、`extraction.warmup=true` |
| T2   | 向下单向定时器（delay-after-T1, min, max） | `scene.delay_after_t1=90s`、`scene.min_interval=15m`、`scene.max_interval=1h`   |
| T3   | session 待处理 或 定期 rollup              | `memory.poll_interval=15m`                                                      |

协调通过 Postgres 状态列 + advisory lock；不需要内存调度状态，重启天然安全（mem9 当前的 summarizer 已经是这种模式）。

## 10. 检索（草图，延后实现）

两个操作，镜像 OpenViking：

- **`find <query>`** —— 单查询，在 `abstract` embedding 上做向量召回 → rerank → 返回 top-K MatchedContext（uri、abstract、score）。
- **`search <query>`** —— LLM 意图分析 → 0–N 个 TypedQuery（memory / scene / turn）→ 每个 query 做分层下钻（向量从 category 级进入，递归 `α·child + (1-α)·parent` 得分传递，top-K 不再变化即收敛）→ rerank → 合并结果。

两者都按 facet 返回：默认带 `abstract`；`?facet=detail` 扩展为完整 body。URI 是唯一的下钻凭据。

## 11. 从当前状态迁移

当前：

- `sessions`、`session_turns` 已有。
- `sessions.overview_text` 是 15 秒 ticker 跑出来的滚动散文。
- 没有 atom、没有 scene、没有 memory 行。

步骤：

1. **阶段 A（仅新增）** —— 新增 Postgres 表（`atoms`、`scenes`、`memories`、`pipeline_state`、`tasks`），以及 `sessions` 表的新列（`abstract`、`embedding`）。其余已有表不动。观察用 CLI 同步落地。
2. **阶段 B** —— 上线 T1 worker（§6.1 追加策略）；新 turn 进入即填 `atoms`。旧 turn 可用 `mem9 t1 backfill` 回灌。生产环境 **关闭** legacy `overview_text` summarizer（`MEM9_SUMMARIZER_ENABLED=false`），避免与 T2 `abstract` 双轨叙事。
3. **阶段 C** —— 上线 T2 worker，包括基于 scene 刷新 `sessions.{abstract, embedding}` 的后置步骤；`scenes` 和 session 级 facet 开始有产物。
4. **阶段 B+（Phase D 之前）** —— 应用 `00003_memories_versioning.sql`，再上线 T3。
5. **阶段 D** —— 上线 T3 worker；`memories` 开始有产物；rollup 后跑 `mem9 eval`。
6. **阶段 E** —— 弃用 `sessions.overview_text` 列与 summarizer worker。session 级叙事改由 `sessions.abstract` 加各 scene 的 `body`（通过 `mem9 tree mem9://sessions/<sid>` 列出）承担。

各阶段独立可发布。采集面（`POST /sessions/:id/upload`）始终不变。

## 12. 待决问题

### 12.1 已锁定（整合策略）

以下在实施 T1/T3 worker **之前** 视为已定，详见 §6.1 与 [`memory-consolidation-review.zh.md`](./memory-consolidation-review.zh.md)：

| 决策 | 立场 |
|------|------|
| T0 证据 | append-only，worker 不改写 |
| T1 dedup | 默认 **追加 atom**；merge **非** worker 默认 |
| T1/T3 `events` | 只插入，不 merge |
| T3 `memories` | 版本化行 + `superseded_at`；禁止 in-place `body` 更新 |
| 论文依据 | Zhang et al., arXiv:2605.12978v1 — 持续 LLM 整合不可靠；保留情景层、稀疏整合 |

### 12.2 实施前仍须确认

1. **Embedding 模型与维度。** 建议：与 LLM 抽取共用 OpenAI 兼容 provider，维度 **1024**（与 Phase A 迁移一致）。
2. **Warmup ramp 数值。** 建议 `2 → 4 → 8 → N=8`（`extraction.every_n=8`），替代 TDAI 的 1→2→4→5。
3. **`mem9 eval` 探针集。** 至少 5 条固定 recall 查询（手写即可），每次 T3 rollup 后对比：
   - **基线：** 仅 `session_turns` + FTS；
   - **全栈：** T0–T3 向量 + FTS + 溯源下钻。
   若全栈命中持续低于基线 → 告警 / 暂停 T3 rollup，调查整合漂移。

### 12.3 `mem9 eval`（最小规格）

```
mem9 eval [--queries=path] [--baseline=t0-fts|full]
```

- 默认读取 `scripts/eval_queries.txt`（或仓库内等价路径），每行一条自然语言查询 + 期望 URI 前缀（可选）。
- 输出：各查询的 hit@k、基线 vs 全栈差分；非零 exit code 表示回归。
- 首版可不接 CI；T3 worker 完成后台触发或文档要求人工跑。

## 13. 参考

- TencentDB Agent Memory (`tmp/TencentDB-Agent-Memory/`)：
  - `README.md` —— 分层记忆 + 符号记忆主张
  - `src/core/prompts/l1-extraction.ts` —— atom 提示词
  - `src/utils/pipeline-manager.ts` —— 三定时器调度
  - `src/core/persona/persona-generator.ts` —— 等价于 T3 的生成器
- OpenViking (`tmp/OpenViking/`)：
  - `docs/en/concepts/01-architecture.md` —— 系统总览
  - `docs/en/concepts/03-context-layers.md` —— 节点内 L0/L1/L2 facet
  - `docs/en/concepts/04-viking-uri.md` —— URI 方案
  - `docs/en/concepts/08-session.md` —— 两阶段提交 + 8 类记忆
  - `docs/en/concepts/07-retrieval.md` —— 带分数传播的分层检索
- 持续整合对照：[`memory-consolidation-review.zh.md`](./memory-consolidation-review.zh.md)（arXiv:2605.12978）
