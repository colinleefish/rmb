# 快速开始

本地运行 rmb、注册 Hook、验证采集与蒸馏。

## 前置条件

- Go 1.26+
- PostgreSQL 16+（含 pgvector，或 Docker Compose）
- OpenAI 兼容 LLM API Key（Worker 用）

## 本地服务

```bash
git clone https://github.com/colinleefish/rmb.git
cd rmb
cp .env.example .env   # 配置 RMB_DB_URL、RMB_LLM_API_KEY、RMB_EMBED_API_KEY
make run               # 或：docker compose up -d
curl localhost:8080/healthz
```

观测 UI：`http://localhost:8080/ui/` — 浏览 session、轮次、原子、场景、记忆、流水线状态。

## 构建 CLI

```bash
make build
./bin/rmb --help
```

安装到 PATH，或让 Hook 指向 `./bin/rmb`。

## 客户端配置

`~/.rmb.conf`（扁平）或 `~/.rmb/config.yaml`：

```ini
RMB_URL=http://127.0.0.1:8080
```

笔记本召回生产环境：

```ini
RMB_URL=https://rmb.colinleefish.com
RMB_USER=...
RMB_PASS=...
```

## 注册 Hook

### Cursor

`~/.cursor/hooks.json`：

```json
{
  "hooks": {
    "afterAgentResponse": [
      {
        "command": "/path/to/rmb/bin/rmb hook-submit --source=cursor",
        "timeout": 5
      }
    ]
  },
  "version": 1
}
```

### Claude Code

`~/.claude/settings.json`：

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          { "type": "command", "command": "/path/to/rmb/bin/rmb hook-submit --source=cc" }
        ]
      }
    ]
  }
}
```

`--source` 必填；载荷不匹配则非零退出。

## 验证采集

1. 进行一段简短 Agent 对话
2. 打开 `/ui/` → Sessions → 查看 turns
3. 等待 T1 后台 worker 提取 atoms；T2 延迟后生成 scenes
4. 出现 atoms；T2 延迟后出现 scenes

## 召回

```bash
rmb search "你了解我什么"
rmb cat rmb://profile
rmb tree rmb://sessions/<session-uuid>/
rmb cat rmb://turns/<turn-uuid>    # 原始 messages_jsonl
rmb cat rmb://atoms/<atom-uuid>    # 抽取的事实
```

Agent 应阅读 [Agent 用 CLI](/zh/guide/cli-for-agents)，可写入 Cursor 规则或 Skill。

## 生产环境

[rmb.colinleefish.com](https://rmb.colinleefish.com)。部署：`make ci && make deploy`。见[部署](/zh/reference/deploy)。

## 下一步

- [金字塔](/zh/concept/pyramid)
- [实施计划](/zh/reference/plan)
- [完整设计](/zh/design/l0-l3)
