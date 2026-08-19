# Local AI Memory

Local AI Memory 是一个仅面向 Codex 的本地长期记忆插件。它让 Codex 能够在不同任务之间延续项目背景，而不是每次新建任务后都从零开始。

安装后，Codex 可以自动发现并增量读取过去的 Codex 任务，从中提取项目决策、用户偏好、约束、解决方案、待办和事实，并在以后处理相关工作时自动检索这些知识。整个过程不要求用户导出、上传或手工导入聊天记录。

> 当前版本只处理 `kind=codex` 的 Codex 任务。它不会读取普通 ChatGPT Quick chat、Claude、Trae、CodeBuddy、浏览器历史或其他应用的数据。

| 项目属性 | 当前情况 |
| --- | --- |
| 当前版本 | `0.3.1` |
| 运行方式 | Codex Skill + 本地 MCP stdio 服务 |
| 数据库 | 本机 SQLite + FTS5 |
| 原始消息保护 | AES-256-GCM；Windows 主密钥由 DPAPI 保护 |
| 支持范围 | 仅 Codex 历史任务 |
| 当前质量状态 | 18 项自动化测试、Ruff、Skill/Plugin 校验和构建均通过 |

## 它解决什么问题

Codex 的每个任务都有自己的上下文。任务结束或新建任务后，之前讨论过的技术决策、排查结论、用户偏好和待办事项不会天然成为新任务的长期记忆。

Local AI Memory 在本机建立一层可核验的长期记忆：

- 自动发现过去的 Codex 任务，不要求手工导出聊天。
- 保存原始来源，同时把可复用内容提取为候选知识。
- 只有用户明确确认的知识才默认参与后续检索。
- 在新任务中先查本地记忆，必要时再回到原始 Codex 任务核验。
- 允许用户查看、确认、拒绝和删除记忆，不把数据库控制权交给云端服务。

它不是云端聊天备份服务，也不是监控所有 AI 软件的后台程序。当前版本不会自动读取 ChatGPT、Claude、Trae、CodeBuddy 或浏览器数据，也不支持跨设备自动同步。

## 最短使用流程

完成安装后，在一个新建的 Codex 任务中依次输入：

```text
使用 $local-ai-memory 检查本地记忆服务是否正常。
```

```text
使用 $local-ai-memory 完整同步我的 Codex 历史任务。只处理 Codex 任务，不读取工具输出。
```

同步完成后即可正常提问，例如：

```text
我们之前为这个项目确定了哪些技术约束？请附上来源任务。
```

如果需要永久保存一条明确结论：

```text
请记住：这个项目所有时间字段统一使用 UTC。
```

后续章节包含完整安装、日常使用、安全边界、命令行和故障排查说明。

## 发布方式

本仓库适合公开上传 GitHub，定位是“开源代码 + 本地 Codex 插件”：

- 用户从 GitHub 克隆代码，在自己的电脑安装 Python 包并注册本地插件。
- 原始消息、SQLite 数据库和主密钥始终留在用户电脑，不属于仓库内容。
- 当前 MCP 使用本地 stdio，不是公网 HTTP 服务，因此不能直接作为 OpenAI 通用插件目录中的公开托管插件提交。
- 若未来要提交到通用目录，需要另外部署稳定的公网 HTTPS MCP，并重新设计认证、隐私政策和数据边界。

公开仓库中的 `.gitignore` 已排除本地数据库、主密钥、虚拟环境、缓存和构建产物。推送前仍应运行 `git status`，人工检查暂存文件。

## 主要能力

- 自动调用 Codex 原生 `list_threads` 和 `read_thread` 工具发现历史任务。
- 根据任务的 `updatedAt` 进行增量同步，只读取新增或发生变化的任务。
- 严格按分页游标顺序读取任务，漏页、乱序或跨版本分页都会被拒绝。
- 常规同步只保存用户消息和 Codex 回复，不保存命令日志、工具输出或文件差异。
- 使用 AES-256-GCM 加密原始消息。
- Windows 上使用当前登录用户的 DPAPI 保护本地主密钥。
- 自动提取的知识默认进入候选区，不会直接作为可靠事实参与普通检索。
- 用户明确说“请记住”或主动确认的内容会成为已确认记忆。
- 使用 SQLite FTS5 和中文检索词在本地搜索。
- 每条记忆都保留来源任务 ID、来源消息 ID、时间和来源链接。
- 支持确认、拒绝、核验和删除记忆，也支持删除完整的本地任务副本。
- 支持通过 Codex Scheduled task 每天凌晨 03:00 自动同步。

## 工作原理

```text
过去的 Codex 任务
        │
        │ list_threads / read_thread
        ▼
增量与分页完整性检查
        │
        ▼
加密保存原始用户消息和 Codex 回复
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

Local AI Memory 把 Codex 任务视为原始来源，把本地数据库视为可重建的加密缓存。当前用户指令和当前项目文件始终高于历史记忆。

## 系统要求

- Windows 10 或 Windows 11（当前主要测试平台）。
- Codex 桌面应用。
- Python 3.11 或更高版本。
- 安装插件后需要新建一个 Codex 任务，让新任务加载 Skill 和 MCP 工具。
- 凌晨自动同步时，电脑需要保持开机，并且 Codex 桌面应用需要正在运行。

## 安装

### 选择下载方式

前往 [GitHub Releases](https://github.com/zhanghaoyu494-cell/local-ai-memory/releases/latest) 下载最新发行版：

- 完整安装推荐下载 `Source code (zip)` 并解压，其中包含 `.codex-plugin`、`.mcp.json`、`skills` 和 Python 服务源码。
- `local_ai_memory-0.3.1-py3-none-any.whl` 只包含 Python 服务，适合升级 Python 包，但不能单独完成 Codex 插件注册。
- 也可以使用下面的 `git clone` 方式获取完整项目。

### 1. 克隆并安装本地 Python 服务

在 PowerShell 中运行：

```powershell
git clone <你的 GitHub 仓库 URL>
cd local-ai-memory
python -m pip install --user .
python -m local_ai_memory.cli init
```

初始化命令会创建本地数据目录、SQLite 数据库和加密主密钥，并显示当前存储位置和统计信息。

### 2. 检查 MCP 启动配置

仓库根目录已经包含可发布的 `.mcp.json`：

```json
{
  "mcpServers": {
    "local-ai-memory": {
      "command": "python",
      "args": ["-m", "local_ai_memory.mcp_server"]
    }
  }
}
```

运行 `python -m local_ai_memory.mcp_server` 应进入 stdio 等待状态。如果 Codex 使用的 `python` 与安装包时不是同一个解释器，请把 `.mcp.json` 的 `command` 改为对应 Python 可执行文件；不要把包含个人绝对路径的修改提交到公共仓库。

### 3. 在 Codex 中安装插件

在以该仓库为项目的 Codex 任务中输入：

```text
使用 $plugin-creator 将当前 local-ai-memory 目录注册为个人插件，创建个人 Marketplace 条目并验证插件。不要修改插件功能代码。
```

Codex 会把插件注册到当前用户的个人 Marketplace。仓库已经包含 `.codex-plugin/plugin.json`、`.mcp.json` 和 `skills/`，不需要手工复制个人路径配置。

### 4. 新建 Codex 任务

插件安装成功后，不要继续使用安装前已经打开的旧任务。新建一个 Codex 任务，使 Codex 重新加载：

- `$local-ai-memory` Skill。
- `local-ai-memory` MCP 服务。
- 13 个 `memory_*` 工具。

可以在新任务中输入：

```text
使用 $local-ai-memory 检查本地记忆服务是否正常，并列出可用的 memory 工具。
```

预期至少能看到以下核心工具：

- `memory_codex_sync_plan`
- `memory_ingest_codex_page`
- `memory_search`
- `memory_remember`
- `memory_candidates`
- `memory_confirm`
- `memory_reject`
- `memory_source`
- `memory_conversations`
- `memory_delete_conversation`
- `memory_delete`
- `memory_stats`
- `memory_consolidate`

最后调用 `memory_stats`，确认返回结果包含：

```json
{
  "plugin_version": "0.3.1"
}
```

如果没有 `plugin_version`，或者版本不是 `0.3.1`，说明当前任务仍连接升级前的 MCP 进程。重新安装或更新个人插件，然后新建 Codex 任务再检查。

## 从旧版本升级

进入克隆后的仓库目录，并使用 MCP 配置中同一个 Python 解释器升级：

```powershell
cd <克隆后的 local-ai-memory 目录>
python -m pip install --user --upgrade .
```

如果 `.mcp.json` 使用虚拟环境中的 Python，则运行：

```powershell
.\.venv\Scripts\python -m pip install --upgrade -e .
```

然后在以该仓库为项目的 Codex 任务中输入：

```text
使用 $plugin-creator 更新当前 local-ai-memory 个人插件，刷新 cachebuster，验证插件并重新安装。不要修改插件功能代码。
```

升级完成后：

1. 关闭或停止继续使用安装前打开的旧任务。
2. 新建一个 Codex 任务。
3. 调用 `memory_stats`。
4. 确认 `plugin_version` 为 `0.3.1`。
5. 再测试保存、删除或完整同步。

`0.3.1` 增加了 SQLite/FTS5 安全删除、WAL 截断、数据库压缩和运行版本自检。升级不会清空已有数据库，也不会重新生成主密钥。

## 首次同步

插件不会要求你导出聊天记录。安装完成并新建任务后，直接输入：

```text
使用 $local-ai-memory 完整同步我的 Codex 历史任务。只处理 Codex 任务，不读取工具输出，完成后告诉我同步了多少任务、消息和候选知识。
```

Codex 将执行以下步骤：

1. 调用 `list_threads` 列出历史任务。
2. 合并置顶任务和普通任务，并按任务 ID 去重。
3. 丢弃非 Codex 类型的聊天。
4. 跳过仍在运行的活动任务和不可用主机上的任务。
5. 调用 `memory_codex_sync_plan` 找出新增或已更新任务。
6. 使用 `read_thread` 分页读取每个待同步任务。
7. 将每一页原样交给 `memory_ingest_codex_page`。
8. 在所有可读取任务完成后调用 `memory_consolidate`。
9. 返回任务、消息、候选知识和失败任务统计。

如果同步中断，可以重新执行同一个请求。消息 ID 和任务版本会保证重复同步不会创建重复消息；中断的多页任务会从最新页重新开始。

## 日常使用

### 自动检索以前的知识

当当前问题可能依赖以前的讨论、项目决策、约束或解决方案时，Skill 会先搜索本地已确认记忆。结果不足、过期或不明确时，它会自动搜索相关 Codex 历史任务。

你可以像平常一样提问：

```text
我们之前决定这个项目使用什么数据库？
```

```text
继续上次关于供应商查询问题的排查，先找一下之前的结论。
```

```text
查看之前是否讨论过这个接口的分页规则，并给出来源任务。
```

如果希望强制使用本地记忆，可以明确调用 Skill：

```text
使用 $local-ai-memory 搜索以前关于“供应商禁止合作校验”的讨论，只返回已确认记忆并附带来源。
```

### 让 Codex 记住一条信息

直接告诉 Codex：

```text
请记住：这个项目所有时间字段统一使用 UTC，界面展示时再转换为本地时区。
```

Skill 会调用 `memory_remember`，把它保存为已确认记忆。

适合长期保存的内容包括：

- `decision`：项目或技术决策。
- `preference`：用户或团队偏好。
- `constraint`：必须遵守的限制。
- `solution`：已经验证有效的解决方案。
- `todo`：需要跨任务保留的待办。
- `fact`：稳定且经过确认的项目事实。

不要保存密码、Token、API Key、Cookie、私钥、验证码或其他敏感凭证。服务检测到常见密钥格式时会拒绝保存。

### 查看自动提取的候选知识

普通历史同步只会生成候选知识。你可以输入：

```text
使用 $local-ai-memory 列出当前项目尚未确认的候选知识。
```

然后让 Codex：

```text
确认第 2 条和第 4 条候选知识，拒绝第 3 条。
```

候选知识经过确认后，才会参与默认检索。

### 核验记忆来源

当记忆涉及敏感操作、不可逆决策、重要配置或看起来可能过期时，可以输入：

```text
核验这条记忆的原始 Codex 消息和来源任务，再决定是否采用。
```

Skill 会通过 `memory_source` 解密并返回对应的原始消息，同时保留任务 ID 和来源链接。

## 记忆状态

| 状态 | 含义 | 默认参与检索 |
| --- | --- | --- |
| `candidate` | 自动抽取、尚未确认 | 否 |
| `confirmed` | 用户明确保存或已经确认 | 是 |
| `rejected` | 用户判定为错误或无用 | 否 |
| `superseded` | 已被更新内容替代 | 否 |

发生冲突时，采用以下优先级：

1. 用户在当前任务中的明确指令。
2. 当前项目文件和实际运行状态。
3. 更新且由用户确认的记忆。
4. 较旧的已确认记忆。
5. 候选知识或 Codex 自己生成的历史陈述。

## 每天凌晨 03:00 自动同步

安装插件后，可以创建名为“Codex 本地记忆同步”的 Codex Scheduled task：

- 执行时间：每天凌晨 `03:00`。
- 运行目录：选择一个本机 Codex 项目目录。
- 执行环境：本地项目。
- 处理范围：仅 Codex 任务。
- 通知策略：建议仅执行失败时通知。

可以在 Codex 侧边栏的 **Scheduled** 页面查看、暂停、恢复或编辑任务。

如果要在另一台电脑重新创建，可以在 Codex 中输入：

```text
创建一个每天凌晨 3 点运行的独立定时任务。每次使用 $local-ai-memory 完整增量同步 Codex 历史任务，只处理 kind=codex，跳过活动任务，不读取工具输出，最后整理本地索引；只有失败时通知我。
```

注意：

- 电脑关机时任务无法运行。
- Codex 桌面应用未运行时，本地任务可能无法执行。
- 插件未安装或 MCP Python 路径失效时，任务会失败。
- 活动任务会被跳过，等任务结束后在下一次同步中处理。
- 如果错过凌晨 03:00，当前 Codex Scheduled task 不保证当天自动补跑；可以手动要求完整同步。

## 本地命令行

命令行主要用于维护本地缓存，不负责发现 Codex 历史任务。历史发现必须由 Codex 原生任务工具完成。

进入项目目录后运行：

```powershell
cd <克隆后的 local-ai-memory 目录>
```

### 初始化或查看存储位置

```powershell
python -m local_ai_memory.cli init
```

### 查看统计信息

```powershell
python -m local_ai_memory.cli stats
```

### 搜索已确认记忆

```powershell
python -m local_ai_memory.cli search "项目数据库约束"
```

限制项目范围：

```powershell
python -m local_ai_memory.cli search "分页规则" --project "project-id-or-path" --limit 10
```

包含未确认候选：

```powershell
python -m local_ai_memory.cli search "供应商查询" --include-candidates
```

### 手工保存一条已确认记忆

```powershell
python -m local_ai_memory.cli remember "该项目统一使用 Python 3.12" --project "my-project" --kind constraint
```

`--kind` 只接受 `decision`、`preference`、`constraint`、`solution`、`todo`、`fact`。
可以用 `--sensitivity normal|personal|high` 显式标记敏感级别；`high` 内容会被拒绝，不会写入可搜索明文索引。检测到的个人信息会先脱敏，并且不能通过显式传入 `normal` 降级。

### 查看候选知识

```powershell
python -m local_ai_memory.cli candidates --project "my-project" --limit 50
```

### 确认、拒绝或删除记忆

```powershell
python -m local_ai_memory.cli confirm <memory-id>
python -m local_ai_memory.cli reject <memory-id>
python -m local_ai_memory.cli delete <memory-id>
```

### 查看来源消息

```powershell
python -m local_ai_memory.cli source <message-id>
```

该命令会解密并显示原始消息，执行前应确认当前终端输出不会被共享或记录到不安全的位置。

### 查看已同步任务

```powershell
python -m local_ai_memory.cli conversations --source codex --limit 100
```

### 删除完整的本地任务副本

先用 `conversations` 获取本地 `conversation_id`，再运行：

```powershell
python -m local_ai_memory.cli delete-conversation <conversation-id>
```

删除完整任务会删除本地加密原始消息，并清理没有其他来源的派生记忆。它不会删除 Codex 应用中的原始任务。

### 整理本地索引

```powershell
python -m local_ai_memory.cli consolidate
```

### 手工启动 MCP 服务

```powershell
python -m local_ai_memory.cli mcp
```

MCP 使用 stdio 通信。手工启动后终端看起来没有普通交互提示是正常现象，按 `Ctrl+C` 可以停止。

## 数据存储位置

默认数据目录：

```text
%LOCALAPPDATA%\LocalAIMemory
```

通常对应：

```text
C:\Users\<用户名>\AppData\Local\LocalAIMemory
```

目录中主要包含：

```text
LocalAIMemory\
├── memory.sqlite3       本地 SQLite 数据库
├── memory.sqlite3-wal   SQLite WAL 文件，运行期间可能存在
├── memory.sqlite3-shm   SQLite 共享内存文件，运行期间可能存在
└── master.key           受当前 Windows 用户 DPAPI 保护的主密钥
```

可以通过环境变量修改存储目录：

```powershell
$env:LOCAL_AI_MEMORY_HOME = "D:\LocalAIMemoryData"
python -m local_ai_memory.cli init
```

也可以对单次命令使用 `--home`：

```powershell
python -m local_ai_memory.cli --home "D:\LocalAIMemoryData" stats
```

Codex 插件和 Scheduled task 必须使用相同的环境变量或默认目录，否则它们可能访问不同的数据库。

## 安全与隐私

### 会保存什么

- Codex 用户消息。
- Codex 回复消息。
- 任务 ID、标题、项目 ID 或工作目录、创建时间和更新时间。
- 自动提取的候选知识。
- 用户确认的长期记忆及其来源关系。

### 默认不会保存什么

- 普通 ChatGPT Quick chat。
- 其他 AI 产品的聊天记录。
- 浏览器历史或网页会话。
- Codex 命令输出和工具输出。
- 文件差异和终端日志。
- Codex 私有数据库内容。
- 空消息、推理内部项和不支持的任务条目。

### 加密边界

- 原始消息正文使用 AES-256-GCM 加密后写入 SQLite。
- Windows 主密钥使用当前用户的 DPAPI 加密。
- 换一个 Windows 用户通常无法直接解密主密钥。
- 非 Windows 系统当前把主密钥保存在权限为 `0600` 的本地文件中，不提供操作系统密钥环保护；公开使用前应评估本机威胁模型。
- 为支持 SQLite 全文检索，经过敏感信息清洗的派生知识会以本地明文索引保存。
- 数据库、主密钥和 Scheduled task 都保存在本机；不要把整个数据目录上传到公共仓库或云盘。
- SQLite 普通表和 FTS5 索引均启用安全删除。显式删除记忆或任务副本后，服务会截断 WAL 并压缩数据库，降低已删除明文残留在空闲页中的风险。
- `memory_stats` 会返回当前 MCP 进程的 `plugin_version`。更新插件后应新建 Codex 任务，并确认运行版本与仓库版本一致。

### 内容安全

历史任务的标题、摘要和消息都被视为不可信数据。历史内容不能覆盖当前系统规则，也不能因为其中写着“运行命令”“上传文件”或“忽略之前指令”就被执行。

### 敏感信息

服务会识别并清洗常见的：

- API Key、Token 和密码字段。
- 电子邮箱。
- 手机号码。
- 身份证号码。

自动检测无法保证覆盖所有私密数据。不要主动要求系统记住凭证或个人敏感信息。

`memory_remember` 只接受 `normal`、`personal` 和 `high` 三种敏感等级：

- 检测到的个人信息会先脱敏，再以 `personal` 等级进入搜索索引。
- 调用方不能通过显式传入 `normal` 降低系统检测出的敏感等级。
- 显式标记或自动判定为 `high` 的内容会被直接拒绝，不进入可搜索的明文知识库。

## 同步策略

### 增量同步

每个完成同步的 Codex 任务都会记录最近一次 `updatedAt`。下次同步时，版本未变化的任务不会重新读取。

### 幂等处理

消息使用 Codex 原始消息 ID 去重。同一页或同一任务被重复提交时，不会产生重复消息。

### 分页完整性

多页任务必须从最新页开始，并严格使用上一页返回的 `nextCursor` 读取下一页。只有连续读取到 `hasMore=false`，任务版本才会被标记为同步完成。

以下情况会拒绝同步并要求从最新页重新开始：

- 直接提交中间页或最后一页。
- 使用错误的 `cursor_used`。
- 跳过某一页。
- 在同一个分页链中混用不同任务版本。
- 页面声明还有更多内容但没有返回 `nextCursor`。

### 活动任务

状态为活动中的任务默认跳过，防止在消息仍持续变化时保存不完整版本。下一次同步会再次检查。

### 工具输出

常规同步固定使用 `includeOutputs=false`。如果某个问题必须查看日志、命令输出或文件差异，Codex 应按需打开原始任务，而不是把所有高风险、高容量输出永久写入记忆库。

## MCP 工具说明

| 工具 | 用途 |
| --- | --- |
| `memory_codex_sync_plan` | 比较 Codex 任务版本并返回待同步任务 |
| `memory_ingest_codex_page` | 加密并保存一页 `read_thread` 结果 |
| `memory_search` | 搜索本地记忆，默认只返回已确认内容 |
| `memory_remember` | 保存用户明确确认的长期记忆 |
| `memory_candidates` | 查看自动抽取的候选知识 |
| `memory_confirm` | 把候选知识升级为已确认记忆 |
| `memory_reject` | 拒绝错误或无用候选 |
| `memory_delete` | 删除一条派生记忆 |
| `memory_conversations` | 查看已同步任务的元数据 |
| `memory_delete_conversation` | 删除一个任务的本地加密副本 |
| `memory_source` | 解密并核验一条原始来源消息 |
| `memory_stats` | 查看任务、消息、记忆数量和当前运行插件版本 |
| `memory_consolidate` | 处理未完成消息并优化本地索引 |

详细参数和同步约定见：

```text
skills\local-ai-memory\references\tool-contracts.md
```

## 故障排查

### Codex 找不到 `$local-ai-memory`

依次检查：

1. 插件是否已经在 Codex 中安装并启用。
2. 个人 Marketplace 是否仍指向当前插件目录。
3. 安装后是否新建了 Codex 任务。
4. Skill 文件是否存在于插件的 `skills\local-ai-memory\SKILL.md`。

旧任务通常不会自动加载刚安装或刚更新的 Skill。

### 找不到 `memory_*` 工具

检查 `.mcp.json` 中的 Python、工作目录和模块名：

```json
{
  "mcpServers": {
    "local-ai-memory": {
      "command": "python",
      "args": ["-m", "local_ai_memory.mcp_server"]
    }
  }
}
```

然后确认虚拟环境可以启动服务：

```powershell
python -m local_ai_memory.mcp_server
```

如果没有立即报错，说明服务已经通过 stdio 等待 MCP 请求；按 `Ctrl+C` 退出。

### 虚拟环境不存在

重新创建并安装：

```powershell
cd <克隆后的 local-ai-memory 目录>
python -m venv .venv
.\.venv\Scripts\python -m pip install -e .
```

### 搜索不到刚同步的内容

可能原因：

- 自动抽取内容仍是 `candidate`，默认搜索不会返回。
- 查询词与记忆文本差异较大，当前版本使用本地 FTS 而不是向量模型。
- 任务仍是活动状态，因此被跳过。
- 该消息属于工具输出，常规同步不会保存。
- 同步还没有读取到对应的旧分页。

可以先查看：

```powershell
python -m local_ai_memory.cli stats
python -m local_ai_memory.cli candidates --limit 100
```

### Scheduled task 没有运行

检查：

1. Codex 桌面应用是否在凌晨 03:00 正在运行。
2. 电脑是否开机且没有进入无法唤醒的状态。
3. Codex 的 **Scheduled** 页面中任务是否为 Active。
4. 插件是否仍然安装。
5. Scheduled task 的项目目录和 MCP Python 命令是否仍然有效。
6. Scheduled run 是否显示插件或 MCP 错误。

### 数据库被占用

SQLite 已启用 WAL 和 30 秒忙等待。若仍出现锁错误：

1. 停止重复启动的 `lam mcp` 或其他本地服务进程。
2. 等待正在进行的同步结束。
3. 再运行 `lam stats` 检查数据库是否恢复。

不要在服务运行时手工修改 SQLite 文件。

### Windows Store 版 `codex.exe` 提示 Access is denied

这不会影响已经运行的 Codex 桌面应用，但可能阻止通过 CLI 安装本地插件。此时在 Codex 桌面应用中使用 `$plugin-creator` 注册当前仓库，并从插件页面安装。

## 开发与验证

### 运行单元测试

```powershell
cd <克隆后的 local-ai-memory 目录>
.\.venv\Scripts\python -m unittest discover -s tests -v
```

### 运行 Ruff

```powershell
.\.venv\Scripts\python -m ruff check .
```

### 验证 Skill

```powershell
.\.venv\Scripts\python "$env:USERPROFILE\.codex\skills\.system\skill-creator\scripts\quick_validate.py" skills\local-ai-memory
```

### 验证插件

```powershell
.\.venv\Scripts\python "$env:USERPROFILE\.codex\skills\.system\plugin-creator\scripts\validate_plugin.py" .
```

### 构建 Wheel

```powershell
.\.venv\Scripts\python -m pip install build hatchling
.\.venv\Scripts\python -m build --wheel
```

构建产物位于：

```text
dist\local_ai_memory-0.3.1-py3-none-any.whl
```

### 更新本地插件缓存版本

修改插件后，使用官方辅助脚本更新 cachebuster：

```powershell
.\.venv\Scripts\python "$env:USERPROFILE\.codex\skills\.system\plugin-creator\scripts\update_plugin_cachebuster.py" .
```

更新后重新安装插件，并新建 Codex 任务进行验证。

### 0.3.1 验证结果

本版本已通过 18 项自动化测试，覆盖：

- 加密往返、敏感信息识别、数据库明文字节检查、普通表/FTS 安全删除、WAL 清理和首次并发密钥创建。
- Codex-only 来源过滤、活动任务跳过、多页游标完整性和零导入增量同步。
- MCP stdio 启动、13 个工具发现、安全 annotations、任务同步、幂等、增量版本和删除闭环。
- 明确“请记住”直接确认、普通知识候选审核、来源消息解密回溯和密钥拒绝。
- `kind`、`sensitivity` 枚举校验、高敏感内容拒绝和个人信息不可降级。
- Markdown 强调符清理、冒号结尾残句过滤和助手过程性承诺过滤。

同时使用 2 个真实 Codex 任务进行了人工桥接端到端复测：

- 连续读取并同步 3 页历史，共保存 62 条用户/Codex 消息。
- 重复同步时新增消息为 0，验证幂等；同步完成后的计划待处理数量为 0。
- 当前真实测试库生成 6 条候选、0 条已确认知识；默认检索不会返回未确认候选。
- 开启候选检索后可命中候选，并能解密回溯到原 Codex 任务、角色和原始消息。
- 抽样原始回复未以明文出现在 SQLite 数据库字节中。

测试命令的当前结果：

```text
18 tests passed
Ruff passed
Skill validation passed
Plugin validation passed
Wheel and source distribution builds passed
```

MCP 依赖当前会输出一条 Pydantic `IncompleteFieldDefinitionWarning`。它来自第三方依赖的未解析前向引用，不影响工具发现、调用结果或上述测试通过状态。

## 上传 GitHub 前检查

当前 `local-ai-memory` 目录是独立 Git 仓库。首次上传或更新 GitHub 前，先在仓库目录执行：

```powershell
git status --short
.\.venv\Scripts\python -m unittest discover -s tests -v
.\.venv\Scripts\python -m ruff check .
.\.venv\Scripts\python -m build
```

如果修改过 Skill 或插件配置，再执行：

```powershell
.\.venv\Scripts\python "$env:USERPROFILE\.codex\skills\.system\skill-creator\scripts\quick_validate.py" skills\local-ai-memory
.\.venv\Scripts\python "$env:USERPROFILE\.codex\skills\.system\plugin-creator\scripts\validate_plugin.py" .
```

确认以下文件不会进入提交：

- `.venv\`、构建缓存和 `dist\`。
- `memory.sqlite3`、`memory.sqlite3-wal`、`memory.sqlite3-shm`。
- `master.key`。
- 真实聊天导出、命令日志、截图或数据库备份。
- API Key、Token、Cookie、密码和私钥。
- 为本机修改的 Python 绝对路径或个人目录路径。

可以使用下面的命令检查即将提交的内容：

```powershell
git status --short
git diff --check
git diff --cached --stat
```

首次发布 GitHub 时，替换仓库地址后执行：

```powershell
git add .
git commit -m "feat: release local-ai-memory 0.3.1"
git branch -M main
git remote add origin <你的 GitHub 仓库 URL>
git push -u origin main
```

推送前必须人工检查暂存内容。不要仅依赖 `.gitignore` 判断是否安全。

## 当前限制

- 只支持 Codex，不支持其他 AI 产品或普通 ChatGPT 聊天。
- 当前使用 SQLite FTS5，不是向量语义检索；表达差异很大时可能漏检。
- 知识抽取基于本地启发式规则，复杂结论需要用户确认。
- 默认同步不包含工具输出、终端日志和文件差异。
- 活动任务不会立即同步。
- 历史任务发现范围受 `list_threads` 返回数量和主机可用性限制。
- 本地派生知识为了检索会以经过清洗的明文存储。
- 当前没有跨电脑自动同步数据库和密钥。
- 删除本地记忆不会修改 Codex 中的原始任务。

## 卸载

1. 在 Codex 插件页面卸载或禁用 `local-ai-memory`。
2. 在 **Scheduled** 页面暂停或删除“Codex 本地记忆同步”。
3. 根据需要保留或手工清理 `%LOCALAPPDATA%\LocalAIMemory`。

卸载插件不会自动删除本地记忆数据库，防止误删。清理本地数据前应确认不再需要历史记忆，并按需备份数据库和主密钥。

## 项目结构

```text
local-ai-memory\
├── pyproject.toml
├── README.md
├── src\local_ai_memory\
│   ├── cli.py
│   ├── codex_adapter.py
│   ├── config.py
│   ├── database.py
│   ├── extractor.py
│   ├── ingestion.py
│   ├── mcp_server.py
│   ├── security.py
│   ├── service.py
│   └── text.py
├── skills\local-ai-memory\
│   ├── SKILL.md
│   ├── agents\openai.yaml
│   └── references\tool-contracts.md
└── tests\
    ├── test_extractor.py
    ├── test_mcp_stdio.py
    ├── test_security.py
    └── test_service.py
```

## 版本

当前 Python 包和仓库插件版本：`0.3.1`。

个人插件开发副本采用 `0.3.1+codex.<cachebuster>` 形式，在本地更新时通过 cachebuster 强制 Codex 重新加载插件内容。仓库中的正式版本仍保持标准的 `0.3.1`。
