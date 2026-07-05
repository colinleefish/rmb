# URI 方案

rmb 中每个产物都用 `rmb://{scope}/{path}` 寻址。CLI、API 与 Agent 通过 URI 引用 turn、atom、scene 与 memory。

## 扁平 scope

T0–T2 实体各自拥有**顶级 scope**。URI 描述的是*这是什么*，而不是*它在目录树的哪一层*。

| Scope | 层级 | URI | 与 session 的关系 |
|-------|------|-----|-------------------|
| `sessions` | 容器 | `rmb://sessions/<sid>` | — |
| `turns` | T0 | `rmb://turns/<uuid>` | `rmb meta` → `session_id` |
| `atoms` | T1 | `rmb://atoms/<uuid>` | `rmb meta` → `session_id` |
| `scenes` | T2 | `rmb://scenes/<uuid>` | `rmb meta` → `session_id` |
| `profile` | T3 | `rmb://profile` | 跨 session |
| `preferences` | T3 | `rmb://preferences/<slug>` | 跨 session |
| `entities` | T3 | `rmb://entities/<slug>` | 跨 session |
| `events` | T3 | `rmb://events/<date-slug>` | 跨 session |

**一共 8 个公开 scope。** `sessions` 只承载对话摘要。Turn、atom、scene **不会**出现在 `rmb://sessions/<sid>/…` 路径下——尽管数据库里的 `session_id` 仍将它们绑在同一条 session 上。

### 为何扁平？

- **稳定 ID** — turn URI 用行的 uuidv7 `id`；atom URI 用 atom 的 UUID。无序号重排问题。
- **统一模式** — `rmb://turns/…`、`rmb://atoms/…`、`rmb://scenes/…` 读起来一致。
- **召回句柄更短** — 搜索结果与 `source_*_uris` 数组更紧凑。

归属关系在 **元数据**（`meta` 里的 `session_id`），不在路径里。

## 容器与 `tree`

末尾 `/` 表示*容器*（列出子项）：

```bash
rmb tree rmb://                  # 列出所有 scope
rmb tree rmb://sessions/<sid>/   # session 摘要 + 该 session 下的 turn/atom 扁平 URI
rmb tree rmb://turns/            # 列出 turn（全局）
rmb tree rmb://atoms/            # 列出 atom（全局）
rmb tree rmb://entities/         # 列出活跃的 entity 记忆
```

`rmb cat rmb://sessions/<sid>` 打印 session **abstract**（不是 turn 正文）。  
`rmb cat rmb://turns/<uuid>` 打印原始 `messages_jsonl`。

## 短格式

CLI 接受省略 scheme 的路径：

```text
/turns/<uuid>     →  rmb://turns/<uuid>
sessions/<sid>/   →  rmb://sessions/<sid>/
```

旧的嵌套路径如 `rmb://sessions/<sid>/turns/0` **不再合法**。

## 溯源链

层与层之间靠外键与 URI 数组连接——不靠路径嵌套：

```text
memory.source_scene_uris  →  scene.source_atom_uris  →  atom.source_turn_ids  →  turn 行
```

`rmb search` 之后典型下钻：

```bash
rmb meta rmb://entities/foo          # source_scene_uris
rmb cat rmb://scenes/<uuid>
rmb meta rmb://atoms/<uuid>          # source_turn_ids, session_id
rmb cat rmb://turns/<uuid>           # 原始证据
```

## 迁移

- **Atom：** 迁移 `00013_flat_atom_uris` 将库内 `atoms.uri` 与 `scenes.source_atom_uris` 从旧嵌套形式改写为扁平形式。
- **Turn：** 无需迁移——URI 始终在读取时由 `id` 计算；新上传直接返回 `rmb://turns/<id>`。

## 延伸阅读

- [金字塔（T0–T3）](/zh/concept/pyramid)
- [设计 §5](/zh/design/l0-l3#_5-uri-方案)
