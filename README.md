# Local AI Memory for Codex

Local AI Memory 是一个仅面向 Codex 的本地长期记忆插件。

它通过 Codex 原生任务工具自动发现并增量读取历史任务，把原始消息加密保存在本地，从中提取可审核的长期知识，并在以后处理相关项目时自动检索。用户不需要导出、上传或手工导入聊天记录。

> 当前版本只处理 `kind=codex` 的 Codex 任务，不读取普通 ChatGPT Quick chat、Claude、Trae、CodeBuddy、浏览器历史或其他应用数据。

## 项目目录

实际项目位于：

```text
local-ai-memory\
```

完整使用手册：

[查看 Local AI Memory 详细 README](./local-ai-memory/README.md)

仓库结构：

```text
piaowu1\
├── .gitignore
├── README.md
└── local-ai-memory\
    ├── pyproject.toml
    ├── README.md
    ├── src\local_ai_memory\
    ├── skills\local-ai-memory\
    └── tests\
```

## 核心能力

- 自动调用 Codex 的 `list_threads` 和 `read_thread` 读取历史任务。
- 根据任务 `updatedAt` 进行增量同步。
- 严格校验多页任务的分页游标，防止漏页和乱序。
- 原始消息使用 AES-256-GCM 加密。
- Windows 主密钥使用当前用户的 DPAPI 保护。
- 常规同步不保存命令日志、工具输出或文件差异。
- 自动抽取的知识默认进入候选区，需要用户确认后才参与普通检索。
- 支持来源核验、候选确认、拒绝、单条删除和完整任务删除。
- 支持每天凌晨 03:00 自动同步 Codex 历史。
- 所有数据默认保存在本机 SQLite 数据库中。

## 快速开始

### 1. 创建虚拟环境

在 PowerShell 中运行：

```powershell
cd F:\piaowu\piaowu1\local-ai-memory
python -m venv .venv
.\.venv\Scripts\python -m pip install -e .
.\.venv\Scripts\lam init
```

默认数据目录：

```text
%LOCALAPPDATA%\LocalAIMemory
```

### 2. 安装 Codex 插件

当前个人插件源码位于：

```text
C:\Users\v_hyuazhang\plugins\local-ai-memory
```

个人 Marketplace 位于：

```text
C:\Users\v_hyuazhang\.agents\plugins\marketplace.json
```

点击下面的链接，在 Codex 中打开并安装插件：

[View local-ai-memory](codex://plugins/local-ai-memory?marketplacePath=C%3A%5CUsers%5Cv_hyuazhang%5C.agents%5Cplugins%5Cmarketplace.json)

安装后必须新建一个 Codex 任务，使新任务加载 `$local-ai-memory` Skill 和 MCP 工具。

### 3. 验证插件

在新 Codex 任务中输入：

```text
使用 $local-ai-memory 检查本地记忆服务，并列出所有可用的 memory 工具。
```

当前 MCP 应提供 13 个工具：

- `memory_codex_sync_plan`
- `memory_ingest_codex_page`
- `memory_search`
- `memory_remember`
- `memory_candidates`
- `memory_confirm`
- `memory_reject`
- `memory_delete`
- `memory_conversations`
- `memory_delete_conversation`
- `memory_source`
- `memory_stats`
- `memory_consolidate`

### 4. 首次同步 Codex 历史

在新任务中输入：

```text
使用 $local-ai-memory 完整同步我的 Codex 历史任务。只处理 Codex 任务，不读取工具输出，完成后告诉我同步了多少任务、消息和候选知识。
```

同步过程会：

1. 列出 Codex 历史任务。
2. 合并置顶任务和普通任务并去重。
3. 跳过非 Codex 聊天、活动任务和不可用主机。
4. 找出新增或发生变化的任务。
5. 按游标逐页读取历史消息。
6. 加密保存用户消息和 Codex 回复。
7. 提取候选知识并整理本地索引。

同步中断后可以安全重试。消息 ID、任务版本和分页游标会防止重复写入或错误完成。

## 日常使用示例

### 搜索以前的项目结论

```text
使用 $local-ai-memory 搜索我们以前关于供应商禁止合作校验的讨论，附带来源任务。
```

### 继续以前没有完成的工作

```text
继续上次供应商查询问题的排查，先读取以前的结论和未完成事项。
```

### 保存长期记忆

```text
请记住：这个项目所有时间字段统一使用 UTC，页面展示时再转换为本地时区。
```

### 审核自动提取的知识

```text
使用 $local-ai-memory 列出当前项目尚未确认的候选知识。
```

### 核验原始来源

```text
核验这条记忆的原始 Codex 消息和来源任务，再决定是否采用。
```

## 记忆状态

| 状态 | 含义 | 默认参与检索 |
| --- | --- | --- |
| `candidate` | 自动抽取、等待用户审核 | 否 |
| `confirmed` | 用户明确保存或已经确认 | 是 |
| `rejected` | 已判定为错误或无用 | 否 |
| `superseded` | 已被更新内容替代 | 否 |

用户当前指令和当前项目实际状态始终高于历史记忆。

## 凌晨自动同步

当前电脑已经创建 Codex Scheduled task：

```text
名称：Codex 本地记忆同步
时间：每天凌晨 03:00
目录：F:\piaowu
范围：仅 Codex 任务
通知：仅失败时通知
```

可以在 Codex 侧边栏的 **Scheduled** 页面查看、暂停、恢复或删除。

自动同步要求：

- 电脑保持开机。
- Codex 桌面应用正在运行。
- `local-ai-memory` 插件仍然安装。
- 项目目录和 MCP Python 路径仍然有效。

## 本地命令

进入实际项目目录：

```powershell
cd F:\piaowu\piaowu1\local-ai-memory
```

常用命令：

```powershell
.\.venv\Scripts\lam init
.\.venv\Scripts\lam stats
.\.venv\Scripts\lam search "项目数据库约束"
.\.venv\Scripts\lam candidates --limit 50
.\.venv\Scripts\lam confirm <memory-id>
.\.venv\Scripts\lam reject <memory-id>
.\.venv\Scripts\lam delete <memory-id>
.\.venv\Scripts\lam source <message-id>
.\.venv\Scripts\lam conversations --source codex
.\.venv\Scripts\lam delete-conversation <conversation-id>
.\.venv\Scripts\lam consolidate
```

命令行只维护本地缓存，不负责发现 Codex 历史。历史任务发现必须通过 Codex 原生任务工具完成。

## 安全边界

### 会保存

- Codex 用户消息。
- Codex 回复。
- 任务标题、任务 ID、项目信息和时间。
- 自动抽取的候选知识。
- 用户确认的长期记忆和来源关系。

### 默认不会保存

- 普通 ChatGPT 聊天。
- 其他 AI 产品聊天记录。
- 浏览器或其他应用数据。
- Codex 工具输出和命令日志。
- 文件差异。
- Codex 私有数据库内容。

### 加密说明

- 原始消息正文使用 AES-256-GCM 加密。
- Windows 主密钥受当前用户 DPAPI 保护。
- 为支持本地全文搜索，经过敏感信息清洗的派生知识会以本地明文索引保存。
- 不要把 `%LOCALAPPDATA%\LocalAIMemory` 上传到公共仓库或不可信云盘。

## 开发验证

运行单元测试：

```powershell
cd F:\piaowu\piaowu1\local-ai-memory
.\.venv\Scripts\python -m unittest discover -s tests -v
```

运行 Ruff：

```powershell
.\.venv\Scripts\python -m ruff check .
```

构建 Wheel：

```powershell
.\.venv\Scripts\python -m pip install build hatchling
.\.venv\Scripts\python -m build --wheel
```

当前版本已验证：

- 13 项单元与 MCP stdio 集成测试通过。
- Ruff 检查通过。
- Skill 校验通过。
- Plugin 校验通过。
- Wheel 构建通过。
- MCP stdio 启动通过。
- 13 个 MCP 工具可发现。
- 使用 2 个真实 Codex 任务完成 3 页连续同步、幂等、增量、候选隔离和来源回溯验证。
- 真实测试库包含 62 条加密消息和 6 条待确认候选；默认检索不会返回候选。

## 当前限制

- 只支持 Codex。
- 使用 SQLite FTS5，不是向量语义检索。
- 自动知识抽取基于启发式规则，重要内容需要用户确认。
- 活动任务会等待后续同步。
- 常规同步不保存工具输出、日志或文件差异。
- 历史发现范围受 Codex `list_threads` 返回数量和主机可用性限制。
- 暂不支持数据库和加密密钥的跨电脑自动同步。

## 详细文档

安装细节、完整 CLI 参数、MCP 工具说明、数据目录、安全模型、分页策略、删除操作和故障排查，请查看：

[local-ai-memory/README.md](./local-ai-memory/README.md)
