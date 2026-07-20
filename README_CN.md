# rmb

[English](README.md) / 简体中文

**面向 AI 智能体的长期记忆系统。**

rmb 从 Cursor、Claude Code 等编码智能体捕获每一轮对话，在后台提炼为结构化事实，让智能体跨会话召回已学到的内容。服务由你自行部署，数据留在你的基础设施上。

## 为什么需要 rmb？

会话结束后，AI 智能体会忘记一切。rmb 通过分层记忆流水线解决这个问题：

1. **捕获** — Cursor、Claude Code、Codex 或 Pi 的 hook 将每一轮对话上传到你的服务器。
2. **提炼** — 后台 worker 提取原子（atom）、归纳为场景（scene），并跨会话沉淀为长期记忆。
3. **召回** — 智能体通过 `rmb` CLI 搜索和浏览记忆，避免反复问你同样的问题。

所有数据存储在 PostgreSQL（配合 pgvector 做语义搜索）中，通过稳定的 `rmb://` URI 寻址。

## 记忆金字塔

知识按层级组织，从原始对话到长期事实：

```
                    ┌─────────────────────────────────────┐
                    │  T3 — 记忆（跨会话）                 │
                    │  profile · preferences · entities   │
                    └──────────────────▲──────────────────┘
                                       │
              ┌────────────────────────┴─────────────────────────┐
              │  会话 · rmb://sessions/<sid>                     │
              │  turns (T0) → atoms (T1) → scenes (T2)           │
              └──────────────────────────────────────────────────┘
```

| 层级 | 含义 | URI 示例 |
|------|------|----------|
| T0 | 原始用户 + 助手对话 | `rmb://turns/<uuid>` |
| T1 | 提炼出的小事实 | `rmb://atoms/<uuid>` |
| T2 | 会话内的叙事片段 | `rmb://scenes/<uuid>` |
| T3 | 跨会话的长期记忆 | `rmb://profile`、`rmb://entities/<slug>` |

完整模型见 [`docs/concept/pyramid.md`](docs/concept/pyramid.md)（英文）。

## 功能

- **智能体 Hook** — `rmb hook-submit --source=<cursor|cc|codex|pi|opencode>` 从支持的工具摄入对话轮次。
- **后台 Worker** — T1 提取、T2 场景合成、T3 记忆汇总（默认开启）。
- **混合召回** — 向量 + 全文检索，经倒数排名融合（RRF）合并。
- **Skills** — 存放在 rmb 中的智能体技能手册（`rmb://skills/<name>`）。
- **纠错** — 人工覆盖，智能体必须遵守。
- **Web UI** — 在 `/ui/` 浏览会话、轮次、原子、场景、记忆和流水线状态。
- **CLI** — `search`、`cat`、`tree`、`meta`、`correction`、`skill` 等命令，供智能体和运维使用。

## 快速开始

### 前置条件

- Go 1.26+（从源码构建）
- PostgreSQL 16+ 及 [pgvector](https://github.com/pgvector/pgvector)
- OpenAI 兼容的 LLM API Key（用于提炼 worker）
- Embedding API Key（用于语义搜索）

### 1. 克隆并配置

```bash
git clone https://github.com/colinleefish/rmb.git
cd rmb
cp .env.example .env
```

编辑 `.env`，按用途配置（`cp .env.example .env` 后改 `replace_me` 和连接串）：

**数据库**

- `RMB_DB_URL` — PostgreSQL 连接串  
  - 源码 + 本机 Postgres：按你的实例填写（示例见 `.env.example`）  
  - `docker compose`：容器内已预设 `postgres://rmb:rmb@postgres:5432/rmb_db`；若在宿主机用 `make run` 连 compose 里的库，用 `postgres://rmb:rmb@127.0.0.1:5433/rmb_db`

**对话 API（T1–T3 后台 worker，三者缺一不可）**

- `RMB_LLM_API_BASE` — OpenAI 兼容接口地址  
- `RMB_LLM_API_KEY` — API Key  
- `RMB_LLM_MODEL` — 模型名（如 `gpt-4o-mini`、`glm-4.7`）

**Embedding API（语义搜索与 embed worker）**

- `RMB_EMBED_API_KEY` — API Key（未设置时 `rmb search` 不可用）  
- `RMB_EMBED_API_BASE` — 接口地址（默认智谱；换厂商时必改）  
- `RMB_EMBED_MODEL` — 模型名（默认 `embedding-3`）  
- `RMB_EMBED_DIMENSIONS` — 向量维度（默认 `1024`，须与模型一致）

**HTTP 认证**

- 监听地址为 `127.0.0.1` 时可不设；绑定 `0.0.0.0` 或 `:8080` 等公网可达地址时，**必须**同时设置 `USERNAME` 与 `PASSWORD`（或 `RMB_USERNAME` / `RMB_PASSWORD`）

用根目录 `docker compose` 时，还需在 `docker-compose.yml` 的 `rmb` 服务加 `env_file: .env`（或逐项 `environment`），否则 LLM / Embed 密钥不会注入容器。

### 2. 启动服务

**Docker Compose**（PostgreSQL + 应用）：

```bash
docker compose up -d
curl http://localhost:8080/healthz
```

**从源码运行**：

```bash
make run
# 或
make build && ./bin/rmb serve
```

在 `http://localhost:8080/ui/` 打开观察器 UI。

### 3. 构建 CLI

```bash
make build
./bin/rmb help
```

将 `./bin/rmb` 加入 PATH，或在 hook 中直接指向该二进制。

### 4. 配置 CLI 指向你的服务器

创建 `~/.rmb.conf` 或 `~/.rmb/config.yaml`：

```ini
RMB_URL=http://127.0.0.1:8080
RMB_USERNAME=your-user
RMB_PASSWORD=your-password
```

召回类命令（`search`、`cat`、`tree`、`meta`、`correction`、`skill`）通过 HTTP 调用服务器；`hook-submit` 向同一 URL 上传轮次。

### 5. 注册智能体 Hook

**Cursor** — `~/.cursor/hooks.json`：

```json
{
  "hooks": {
    "afterAgentResponse": [
      {
        "command": "/path/to/rmb hook-submit --source=cursor",
        "timeout": 5
      }
    ]
  },
  "version": 1
}
```

**Claude Code** — `~/.claude/settings.json`：

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          { "type": "command", "command": "/path/to/rmb hook-submit --source=cc" }
        ]
      }
    ]
  }
}
```

**Pi** — 从 [`integrations/pi/`](integrations/pi/README.md) 安装扩展（无 shell hook，使用 `agent_settled` 事件）。

**OpenCode** — 从 [`integrations/opencode/`](integrations/opencode/README.md) 安装插件（使用 `session.status` idle / `session.idle` 事件）。

`--source` 为必填项；载荷与来源不匹配时以非零退出。

### 6. 验证

1. 与智能体进行一段简短对话。
2. 打开 `/ui/` → Sessions → 选择会话 → 确认轮次已出现。
3. 等待后台 worker 提取原子和场景。
4. 通过 CLI 召回：

```bash
rmb search "你了解我什么"
rmb cat rmb://profile
rmb tree rmb://sessions/<session-uuid>/
```

## 部署到你的服务器

rmb 设计为在你掌控的基础设施上运行。典型生产架构：

```
Internet
   │
   ▼
┌─────────┐     ┌──────────────┐     ┌────────────┐
│  Caddy  │────►│  rmb :8080   │────►│ PostgreSQL │
│  :443   │     │  (Docker)    │     │  + pgvector│
└─────────┘     └──────────────┘     └────────────┘
```

### 方案 A — Docker Compose（一体化开发 / 小规模部署）

根目录 [`docker-compose.yml`](docker-compose.yml) 同时运行 PostgreSQL 和 rmb，适合单机 VM 或 homelab：

```bash
docker compose up -d
```

可在 `docker-compose.yml` 或同目录 `.env` 中自定义环境变量。

### 方案 B — 数据库与应用分离

生产环境建议在宿主机或托管服务上运行 PostgreSQL，仅部署应用容器。参见 [`deploy/docker-compose.yml`](deploy/docker-compose.yml) 的最小双服务布局（rmb + Caddy 反向代理）。

1. 将 `deploy/` 复制到服务器（如 `/app/rmb`）。
2. 创建 `.env`，填写 `RMB_DB_URL`、LLM Key 和认证凭据。
3. 编辑 [`deploy/config/Caddyfile`](deploy/config/Caddyfile)，将域名改为你的：

```
memory.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

4. 构建并推送镜像，或在服务器上构建：

```bash
docker build -t rmb:local .
```

5. 启动：

```bash
cd /app/rmb
docker compose up -d
curl -fsS https://memory.example.com/healthz
```

6. 在运行 hook 或召回的机器上，将客户端 `RMB_URL` 设为 `https://memory.example.com`。

### 安全清单

- 服务器非 localhost 绑定时，务必启用 `USERNAME` / `PASSWORD`（或 `RMB_USERNAME` / `RMB_PASSWORD`）。
- 在反向代理（Caddy、nginx、Traefik）处终止 TLS。
- PostgreSQL 不要暴露到公网；rmb 通过内网或 localhost 连接。
- API Key 放在服务器 `.env` 中，不要写进 hook 配置。

## 架构

```txt
agent (Cursor / Claude Code / Pi)
   │
   │ stdin JSON  ── hooks ──► rmb hook-submit --source=…
   │                                │
   │                                ▼
   │                     POST /api/v1/sessions/:id/upload
   │                                │
   ▼                                ▼
                              ┌──────────────┐
                              │     rmb      │
                              │  (Gin HTTP)  │
                              └──────┬───────┘
                                     │
                    ┌────────────────┼────────────────┐
                    ▼                ▼                ▼
             ┌────────────┐   ┌────────────┐   ┌────────────┐
             │ T1 extract │   │ T2 scene   │   │ T3 memory  │
             │  worker    │   │  worker    │   │  worker    │
             └─────┬──────┘   └─────┬──────┘   └─────┬──────┘
                   │                │                │
                   └────────────────┼────────────────┘
                                    ▼
                              ┌──────────────┐
                              │  PostgreSQL  │
                              │  + pgvector  │
                              └──────────────┘
```

后台 worker 在 `rmb serve` 进程内运行，需要 OpenAI 兼容的对话 API。召回用的 embed worker 使用独立的 embedding 端点。

## CLI 参考

| 命令 | 用途 |
|------|------|
| `rmb serve` | 启动 HTTP 服务（在拥有数据库的主机上运行） |
| `rmb hook-submit --source=<src>` | 摄入 hook 载荷（始终是 HTTP 客户端） |
| `rmb search "<query>"` | 跨记忆、场景、技能的混合召回 |
| `rmb cat <uri>` | 打印记忆产物正文 |
| `rmb tree <uri-prefix>` | 列出某 scope 下的子项 |
| `rmb meta <uri>` | 查看元数据（溯源、会话关联） |
| `rmb correction add <uri> "…"` | 附加人工纠错 |
| `rmb skill ls` / `pull` / `put` | 管理智能体技能 |

智能体应先运行 `rmb search`，再对相关 URI 执行 `rmb cat`。详见 [`docs/guide/cli-for-agents.md`](docs/guide/cli-for-agents.md)（英文）。

## API

| 端点 | 说明 |
|------|------|
| `POST /api/v1/sessions/:id/upload` | 追加一轮对话（hook 目标） |
| `GET /api/v1/search?q=…&k=…` | 混合召回 |
| `GET /api/v1/inspect/{cat,tree,meta}?uri=…` | 检视（CLI 使用） |
| `GET /healthz` | 数据库 ping + pgvector 检查 |

上传请求体：

```json
{
  "started_at": "optional RFC3339",
  "messages": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ]
}
```

## 配置

解析顺序（后者覆盖前者）：

1. `internal/config` 中的默认值
2. `~/.rmb.conf` 或 `~/.rmb/config.yaml`（客户端）；服务端使用工作目录下的 `.env`
3. 进程环境变量

主要服务端变量（见 [`.env.example`](.env.example)）：

| 变量 | 默认值 | 用途 |
|------|--------|------|
| `RMB_DB_URL` | `postgres://admin@127.0.0.1:5432/rmb_db?sslmode=disable` | PostgreSQL |
| `RMB_ADDR` | `:8080` | 监听地址 |
| `RMB_LLM_API_BASE` / `_API_KEY` / `_MODEL` | — | Worker 用的对话 API |
| `RMB_EMBED_API_KEY` | — | 召回 embedding（搜索必需） |
| `RMB_EXTRACTION_ENABLED` | `true` | T1 原子提取 |
| `RMB_SCENE_ENABLED` | `true` | T2 场景合成 |
| `RMB_MEMORY_ENABLED` | `true` | T3 记忆汇总 |

客户端变量：`RMB_URL`、`RMB_USERNAME`、`RMB_PASSWORD`。

## 项目结构

```
cmd/rmb/              CLI 入口（serve / hook-submit / recall）
internal/
  hook/               hook 载荷 → 上传适配（cursor、cc、codex、pi、opencode）
  http/               Gin 路由、handler、内嵌 Web UI
  service/
    extract/          T1 提取 worker
    scene/            T2 场景 worker
    memory/           T3 记忆 worker
    embed/            embedding worker
  llm/                OpenAI 兼容客户端
ui-next/              Next.js 观察器 UI（编译进二进制）
integrations/pi/      Pi 智能体扩展
integrations/opencode/ OpenCode 插件
deploy/               生产 compose + Caddy 示例
docs/                 完整文档站（make docs-dev）
```

## 开发

```bash
make ci          # go test ./... + 编译检查
go test ./...    # 仅测试
make docs-dev    # 本地文档站 localhost:5173
```

路线图：[`docs/reference/plan.md`](docs/reference/plan.md)。计划中：MCP 封装、`rmb eval` 漂移探测。

## 文档

- [快速上手](docs/zh/guide/getting-started.md)（中文）
- [记忆金字塔](docs/zh/concept/pyramid.md)（中文）
- [URI 方案](docs/zh/concept/uri-scheme.md)（中文）
- [Hook 集成](docs/zh/guide/getting-started.md)（中文）

运行 `make docs-dev` 可在本地浏览完整文档站。

## 许可证

待定。
