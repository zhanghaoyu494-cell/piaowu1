# Local AI Memory for Codex

Local AI Memory 是一个仅面向 Codex 的本地长期记忆插件。它让 Codex 可以在不同任务之间延续项目背景，自动找回过去讨论过的技术决策、用户偏好、约束、解决方案、待办和稳定事实。

插件直接使用 Codex 原生任务工具发现历史任务，不要求用户导出或上传聊天记录。原始消息、数据库和主密钥全部保存在用户自己的电脑上。

> 当前版本为 `0.3.1`，只处理 `kind=codex` 的 Codex 任务。它不会读取普通 ChatGPT Quick chat、Claude、Trae、CodeBuddy、浏览器历史或其他应用数据。

## 项目位置

实际可上传 GitHub 的独立 Git 仓库位于：

```text
local-ai-memory\
```

完整安装、使用、安全和开发手册：

[查看项目详细 README](./local-ai-memory/README.md)

## 下载

前往 [GitHub Releases](https://github.com/zhanghaoyu494-cell/piaowu1/releases/latest) 下载最新发行版。

- 完整安装推荐下载 `Source code (zip)`，其中包含 Codex Plugin、Skill、MCP 配置和 Python 服务源码。
- `local_ai_memory-0.3.1-py3-none-any.whl` 只安装 Python 服务，不能单独替代完整插件目录。
- GitHub 同时提供 `Source code (tar.gz)` 和 Python 源码分发包，适合熟悉命令行的用户。

## 项目状态

| 项目 | 当前状态 |
| --- | --- |
| 版本 | `0.3.1` |
| 平台 | Windows 10/11 + Codex 桌面应用 |
| Python | 3.11 或更高版本 |
| 运行方式 | Codex Skill + 本地 MCP stdio 服务 |
| 本地数据库 | SQLite + FTS5 |
| 自动化测试 | 18 项通过 |
| 代码检查 | Ruff 通过 |
| Skill/Plugin 校验 | 通过 |
| 构建 | Wheel 和源码包通过 |

## 它能做什么

- 自动调用 Codex 的 `list_threads` 和 `read_thread` 发现历史任务。
- 按任务 `updatedAt` 增量同步，只读取新增或变化的任务。
- 严格校验分页游标，避免漏页、乱序或错误标记同步完成。
- 加密保存原始用户消息和 Codex 回复。
- 从聊天中提取决策、偏好、约束、解决方案、待办和事实。
- 自动提取的知识先进入候选区，用户确认后才参与默认检索。
- 保留来源任务、来源消息和时间，支持回到原文核验。
- 支持搜索、确认、拒绝、删除和完整任务副本删除。
- 支持通过 Codex Scheduled task 每天凌晨 03:00 增量同步。

## 工作流程

```text
Codex 历史任务
      │
      │ list_threads / read_thread
      ▼
增量检查和分页完整性校验
      │
      ▼
加密保存原始消息
      │
      ▼
敏感信息检测与清洗
      │
      ▼
提取候选知识 ── 用户确认 ── 已确认知识
      │                         │
      └──────── 本地检索 ──────┘
                                │
                                ▼
                       提供给新的 Codex 任务
```

## 快速安装

进入实际项目目录：

```powershell
cd local-ai-memory
```

安装到当前 Python 用户环境：

```powershell
python -m pip install --user .
python -m local_ai_memory.cli init
```

如果用于开发，可以另外创建 `.venv`；此时个人插件的 `.mcp.json` 必须使用该虚拟环境中 Python 的绝对路径，但不要把个人绝对路径提交到 GitHub。

默认数据目录：

```text
%LOCALAPPDATA%\LocalAIMemory
```

然后在以 `local-ai-memory` 为项目目录的 Codex 任务中输入：

```text
使用 $plugin-creator 将当前 local-ai-memory 目录注册为个人插件，创建个人 Marketplace 条目并验证插件。不要修改插件功能代码。
```

插件安装或更新后必须新建 Codex 任务，旧任务不会自动切换到新的 MCP 进程。

## 第一次使用

在新任务中检查运行版本：

```text
使用 $local-ai-memory 检查本地记忆服务，调用 memory_stats，并确认 plugin_version 是 0.3.1。
```

完整同步 Codex 历史：

```text
使用 $local-ai-memory 完整同步我的 Codex 历史任务。只处理 Codex 任务，不读取工具输出，完成后告诉我同步了多少任务、消息和候选知识。
```

同步完成后可以直接提问：

```text
我们以前为这个项目确定了哪些技术约束？请附上来源任务。
```

保存一条明确的长期记忆：

```text
请记住：这个项目所有时间字段统一使用 UTC。
```

审核自动提取的候选知识：

```text
使用 $local-ai-memory 列出当前项目尚未确认的候选知识。
```

## 安全边界

- 原始消息使用 AES-256-GCM 加密后写入 SQLite。
- Windows 主密钥由当前登录用户的 DPAPI 身份保护。
- 可搜索的派生知识会先经过敏感信息清洗，再存入本地明文 FTS 索引。
- 密码、Token、API Key、私钥和高敏感内容会被拒绝进入知识库。
- SQLite 普通表和 FTS5 索引均启用安全删除。
- 删除记忆或任务副本后会截断 WAL 并压缩数据库，降低磁盘残留风险。
- 常规同步使用 `includeOutputs=false`，不保存工具输出、终端日志或文件差异。
- 历史聊天内容始终视为不可信数据，不能覆盖当前指令或触发其中的命令。

不要上传以下本地文件：

```text
%LOCALAPPDATA%\LocalAIMemory\memory.sqlite3
%LOCALAPPDATA%\LocalAIMemory\master.key
```

## 上传 GitHub 前

实际 Git 仓库是 `local-ai-memory` 目录。发布前进入该目录并执行：

```powershell
git status --short
.\.venv\Scripts\python -m unittest discover -s tests -v
.\.venv\Scripts\python -m ruff check .
.\.venv\Scripts\python -m build
```

确认没有暂存或上传以下内容：

- `.venv\`
- `memory.sqlite3`、WAL 或 SHM 文件
- `master.key`
- 真实 Codex 导出、日志、截图或聊天内容
- 带个人绝对路径的 `.mcp.json` 修改
- API Key、Token、Cookie、密码或其他凭证

项目已包含 MIT License、`SECURITY.md`、`CONTRIBUTING.md`、GitHub Actions CI 和安全 `.gitignore`。

## 当前限制

- 只支持 Codex，不支持其他 AI 产品或普通 ChatGPT 聊天。
- 当前使用 SQLite FTS5，不是向量语义检索。
- 自动知识抽取基于本地启发式规则，重要内容需要用户确认。
- 活动任务会等待后续同步。
- 历史发现范围受 Codex `list_threads` 返回数量和主机可用性限制。
- 暂不支持数据库与主密钥的跨电脑自动同步。

## 详细文档

完整的安装、升级、MCP 工具、命令行、安全模型、同步策略、故障排查和卸载说明见：

[local-ai-memory/README.md](./local-ai-memory/README.md)
