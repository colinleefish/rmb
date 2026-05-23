# 持续整合 vs 情景保留——一份对照阅读

> 状态：草稿。围绕 arXiv 论文 *Useful Memories Become Faulty When Continuously Updated by LLMs* (Zhang et al., 2026, arXiv:2605.12978v1)，对照评估 TencentDB Agent Memory（TDAI）、OpenViking、以及本仓库 mypast 的当前设计。
>
> 配套阅读：[`design-l0-l4.zh.md`](./design-l0-l4.zh.md)。

## 1. 论文一段话总结

Agent 的"长期记忆"通常被实现成一份可由 LLM 反复重写的文本备忘——每次新轨迹都让 LLM 把过去蒸馏一次，覆盖旧版本。论文的核心实证是：**这种持续整合并不可靠**。

- 在 ScienceWorld、WebShop、ALFWorld、AppWorld、ARC-AGI 等任务上，记忆效用随整合次数*先升后降*，部分场景甚至跌破"无记忆"基线。
- 最干净的反例：GPT-5.4 先在一组 ARC-AGI 题上以 100% 正确率裸解；之后即便喂入**这些题的标准答案**做整合，再次回访时仍有 46–54% 失败。错的不是经验，而是整合本身。
- 三种失败模式：
  1. **错误分组**——把不同结构的经验拢到一处再抽象。
  2. **剥离适用条件**——抽象后的"教训"过于泛化，反而干扰邻近任务。
  3. **窄流过拟合**——输入流不够多样时，抽象过拟合到具体实例。
- 在显式提供 *Retain / Delete / Consolidate* 三种动作的对照实验中，agent 默认*保留*原始 episode，准确率约为强制整合方案的两倍；而完全关闭整合（仅做 episodic 管理）与"自由选择"方案旗鼓相当。

> 结论：**把原始 episode 视为一等证据，显式控制整合时机，不要在每次交互后都重写记忆。情景层与抽象层应当分开持有，而不是塌缩为一个重写循环。**

## 2. 评估视角

把任意一种"agent 记忆"系统拆成两类层：

- **情景层（episodic）**——append-only，写入后不可被 LLM 改写。
- **抽象层（consolidation）**——由 LLM 重写产生的浓缩物。

围绕三个问题打分：

1. 原始 episode 是否被永久保留？
2. 重写发生在哪一层？多久一次？
3. 抽象层与原始 episode 之间是否有可追溯链路？

## 3. 三方对照

| 维度 | TDAI | OpenViking | mypast（当前设计） |
|---|---|---|---|
| 原始 episode 永久保留 | 有（L0） | 有（session turns） | **有**（`session_turns` 永久；`source_turn_ids` 上行透传） |
| 上层结构 | L1 atom → L2 scene → L3 persona | 每节点 L0 abstract / L1 overview / L2 detail（同一节点的多视图） | T1 atom → T2 scene → T3 memory（4 类） |
| 整合触发 | 三计时器调度 | session 结束时抽取 | **T1：每 N 轮 + idle + warmup ramp；T2：单向下沉计时器；T3：session-pending 或定时 rollup** |
| 顶层重写方式 | persona 整体重新生成 | 每 session 抽取 → 8 类用户记忆滚动更新 | T3 按 `(category, slug)` upsert；`profile` 为单例行 |
| 反向溯源链路 | atom→turns | URI 树 | 每层都带 `source_*_uris` 数组 |
| 显式不可变层 | 无 | 无 | **`events` 按规则不可变** |
| 审计 / 回滚 | 无 | 无 | 已搁置（`memory_diff.json` 暂未做） |

## 4. 论文映射到三方

### 4.1 TDAI——风险最高

L3 persona 是会被*整体重新生成*的产物，喂入它的 L2 scene 又是 L1 atom 的 LLM 聚合。三个计时器在繁忙 session 上会被频繁触发，等价于"每次交互后都跑一遍整合"，这就是论文最直接命中的场景。

### 4.2 OpenViking——风险中等

OpenViking 的 L0/L1/L2 facet 描述的是**单节点的多个展示粒度**，与"重写策略"基本正交，并不直接落入论文的失败模式。真正的暴露面是 session 结束时的"记忆抽取"以及 8 类用户记忆的滚动更新；不过抽取按 session 结界（更接近论文里没塌的 *Static* 方案），单次 session 内不会反复重写同一条记忆。

### 4.3 mypast——总体风险最低，但有两处具体雷区

✅ 站在论文那边的部分：

- **T0 永久保留。** `source_turn_ids / source_atom_uris / source_scene_uris` 让每一层都能向下追到原始轨迹——论文里那个"仅 episodic"的对照组本就内置于设计中。
- **整合显式分级触发。** 不是每轮都跑：T1 用 every-N + idle + warmup，T2 是单向下沉计时器，T3 等 pending 或定时 rollup。这正是论文给出的处方。
- **`events` 不可变。** 这是设计中最接近"Retain by default"的分类规则。

⚠️ 雷区：

1. **T1 dedup 用"embedding top-K + LLM merge decision"。**  
   每一次 merge 都是一次小型重写，正是论文所说的"剥离适用条件"。建议把 dedup 默认行为偏向 *append + tag*，merge 设为显式的、稀疏的、可人工触发的动作；这就是论文实验里击败强制整合的 *Retain / Delete / Consolidate* 拆分。
2. **T3 `profile` 是被原地改写的单例。**  
   单例 + 原地改写 = 微型重写循环。第 N 轮如果丢掉一条稳定身份属性，没有任何机制能发现。建议 `memories` 改为内部追加（加 `version int` 与 `superseded_at timestamptz`），对外仍只看最新版（`WHERE superseded_at IS NULL`）。被搁置的"审计日志"思路就是这条的天然落点。
3. **T3 `preferences` / `entities` 按 slug upsert。**  
   同类风险，因为 slug 锚定了行身份，比 `profile` 缓和；但 body 列同样建议版本化。
4. **缺少漂移检测。**  
   论文最重要的曲线是"先升后降"——不主动度量就察觉不到拐点。可以利用 TODO 中已经规划的"五条 sanity recall 查询"：把它们做成 `mypast eval`，每次 T3 rollup 后跑一次，对比 `T0+FTS` vs `T0–T3 全栈`，差值转负即报警。

## 5. 给 mypast 的最小改动建议（按 ROI 排序）

所有项与现有设计兼容，不破坏 §11 中的迁移阶段。

> **状态（2026-05）：** 下列 1–4 项已写入 [`design-l0-l4.zh.md`](./design-l0-l4.zh.md) §6.1、§7.1、§8、§12 与 [`design-l0-l4.md`](./design-l0-l4.md)；`memories` 迁移草图见 [`00003_memories_versioning.sql`](../internal/db/migrations/00003_memories_versioning.sql)；探针草案见 [`scripts/eval_queries.txt`](../scripts/eval_queries.txt)。

1. **T1 dedup 改为默认追加。**  
   merge 不再是 worker 默认动作，而是显式 CLI 命令或人工任务；prompt 与 worker 分支各改一处即可。
2. **T3 行版本化。**  
   `memories` 增加 `version int` 与 `superseded_at timestamptz`，禁止 in-place 修改 body；读取最新版用 `WHERE superseded_at IS NULL`。
3. **加一个最小评测命令。**  
   `mypast eval` 跑 N 条固定 recall 查询，对比"裸 T0+FTS"与"完整栈"的命中差值；T3 rollup 后自动跑，回归即报警。
4. **在设计文档中显式锁定立场。**  
   在 [`design-l0-l4.zh.md`](./design-l0-l4.zh.md) §12 中引用本论文，并写明："T1 dedup 中的 merge 是受控、稀疏的，不是 worker 默认行为。"

## 6. 一句话定位

> mypast 当前设计是三者中**唯一显式按触发纪律分级整合**的方案，已经站在这篇论文的正确一侧；剩下的两处实质性风险——T1 LLM merge 与 T3 `profile` 原地改写——都可以靠"让整合更稀疏 + 默认追加"修补，而不动四层金字塔的整体形状。

## 7. 参考

- Zhang, Lin, Wu, Sun, Li, Li, Peng. *Useful Memories Become Faulty When Continuously Updated by LLMs*. arXiv:2605.12978v1, 2026-05-13.
- 配套设计文档：[`design-l0-l4.zh.md`](./design-l0-l4.zh.md) / [`design-l0-l4.md`](./design-l0-l4.md)
- 周边材料：`grove/20260418.openviking/README.md`（OpenViking 评估笔记）。
