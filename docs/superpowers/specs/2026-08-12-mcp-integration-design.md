# SDK MCP 集成设计

## 1. 目标

为 `sdk` 增加本地 MCP Server，使 AI 客户端能够安全地读取和管理开发工具环境。

第一阶段只实现只读能力，覆盖 Java、Node.js、Maven、Go，以及项目级 `.sdk-version`。
第二阶段再增加安装、卸载、切换默认版本和配置迁移等修改操作。

MCP Server 通过 stdio 启动：

```bash
sdk mcp serve
```

stdout 只输出 MCP 协议消息；诊断日志和调试信息输出到 stderr。

## 2. 设计原则

- CLI 和 MCP 不维护两套业务逻辑。
- MCP 层只负责协议适配、参数校验和结果序列化。
- 版本、安装、状态、环境诊断等逻辑抽取到共享 Service 层。
- 第一阶段的所有 MCP Tool 都是只读的，不修改文件、软链接、环境或远程状态。
- 第二阶段的修改 Tool 必须支持 `dry_run`，并要求显式 `confirm: true`。
- 所有 MCP 请求继承 context，支持取消和超时。
- 使用现有 `SDK_HOME` 规则，不允许 MCP 请求任意指定状态目录。

## 3. 总体架构

```text
CLI ───────────────┐
                   ├── internal/service
MCP stdio server ──┘          │
                              ├── catalog
                              ├── installer
                              ├── store
                              ├── environment
                              └── project
```

建议新增目录：

```text
internal/service/
internal/mcp/
```

### Service 层

Service 层提供结构化的 Go API，不依赖命令行输出，也不读取 stdin。CLI 命令和 MCP Handler
共同调用 Service。

第一阶段至少需要这些服务接口：

```go
ListInstalled(ctx context.Context, tool *model.Tool) ([]InstalledTool, error)
CurrentVersions(ctx context.Context) ([]CurrentTool, error)
AvailableVersions(ctx context.Context, tool model.Tool) ([]catalog.Version, error)
Doctor(ctx context.Context) (DoctorReport, error)
ProjectVersions(ctx context.Context, directory string) (ProjectReport, error)
CheckUpdates(ctx context.Context, tool *model.Tool) ([]UpdateInfo, error)
ToolPath(ctx context.Context, tool model.Tool) (ToolPath, error)
```

Service 返回结构化结果；CLI 再负责格式化文本，MCP 再负责 JSON Schema 结果。

## 4. MCP Server

使用官方 Go MCP SDK 的 `github.com/modelcontextprotocol/go-sdk/mcp` 包，采用
`mcp.Server` 和 `mcp.StdioTransport`。第一阶段不实现 HTTP transport，也不监听网络端口。

命令路由增加：

```text
sdk mcp serve
```

启动时初始化：

1. 解析 `SDK_HOME`；
2. 创建共享 Service；
3. 创建 MCP Server，名称为 `sdk`；
4. 注册第一阶段 Tools、Resources 和 Prompts；
5. 运行 stdio transport，直到客户端断开或 context 取消。

## 5. 第一阶段 Tools

### `sdk_list_installed`

列出已注册的本地版本。

输入：

```json
{ "tool": "nodejs" }
```

`tool` 可选；省略时返回全部工具。

结果包含：工具名、版本、安装路径、是否当前默认、是否由 SDK 管理。

### `sdk_current_versions`

返回所有工具当前默认版本、current 软链接和实际路径。

### `sdk_available_versions`

查询官方正式版本。

输入：

```json
{ "tool": "java" }
```

结果保留版本号、LTS 标识、已安装标识和当前使用标识。

### `sdk_doctor`

返回结构化诊断报告，包括：

- SDK home 和状态文件；
- 安装目录；
- current 软链接；
- Java、Maven、Go、Node.js 及 npm/npx 可执行文件；
- PATH 和工具环境变量；
- 错误、警告和通过项统计。

### `sdk_project_versions`

读取当前目录或指定项目目录向上查找的最近 `.sdk-version`。

输入：

```json
{ "directory": "/absolute/path/to/project" }
```

目录必须是绝对路径，并且默认限制在 MCP 启动时的工作目录或用户明确授权的项目目录内。

### `sdk_check_updates`

检查远程最新正式版本，不下载、不安装、不切换默认版本。

输入中的 `tool` 可选；结果包含当前版本、推荐版本、是否已安装和升级建议。

### `sdk_tool_path`

返回当前工具版本根目录。

### `sdk_tool_which`

返回当前工具实际可执行文件路径。Node.js 结果同时返回 `node`、`npm` 和 `npx` 路径。

所有 Tool 的输入都必须校验工具名，只允许 `java`、`nodejs`、`maven`、`go`。

## 6. Resources

第一阶段提供以下只读 Resources：

```text
sdk://state
sdk://current
sdk://doctor
sdk://project/.sdk-version
```

Resource 内容使用 JSON 或纯文本：

- `sdk://state`：完整状态文件的脱敏结构；
- `sdk://current`：当前默认版本和路径；
- `sdk://doctor`：最近一次诊断报告；
- `sdk://project/.sdk-version`：最近项目版本文件内容及解析结果。

不存在的项目文件返回明确的“未找到”，不当作协议错误。

## 7. 第二阶段修改能力

第二阶段增加：

```text
sdk_install
sdk_uninstall
sdk_set_default
sdk_update
sdk_migrate_config
```

所有修改 Tool 使用统一参数：

```json
{
  "dry_run": true,
  "confirm": false
}
```

规则：

- `dry_run=true` 只返回计划，不改变文件或网络状态；
- `dry_run=false` 且 `confirm` 不是 `true` 时拒绝执行；
- 默认版本切换、卸载、配置迁移都要返回将要影响的路径和版本；
- 删除操作只能作用于 SDK 管理的安装目录；
- 安装继续使用现有官方校验和、重试和进度能力；
- Node.js 全局包迁移和 Maven 配置迁移必须在结果中列出成功、跳过和失败项。

## 8. 错误处理

MCP Handler 不把错误格式化成 CLI 文本，而是返回结构化错误：

```json
{
  "code": "version_not_installed",
  "message": "nodejs 22.0.0 is not installed",
  "details": {}
}
```

错误类别至少包括：

- `invalid_tool`；
- `invalid_path`；
- `state_error`；
- `remote_catalog_error`；
- `version_not_installed`；
- `confirmation_required`；
- `operation_cancelled`。

远程查询错误不应泄露访问令牌、完整本地环境变量或用户目录之外的敏感路径。

## 9. 安全边界

- 默认只使用 stdio，不暴露 HTTP 端口；
- MCP Server 不提供任意 Shell 执行 Tool；
- 不允许通过 Tool 参数覆盖 `SDK_HOME`；
- 项目目录参数必须做绝对路径和权限范围校验；
- 修改操作必须显式确认；
- 所有安装包继续执行 SHA-256/SHA-512 校验；
- stdout 禁止写入日志，避免破坏 JSON-RPC 通道；
- 错误和调试信息写 stderr。

## 10. 测试计划

### Service 测试

- 只读服务返回正确的结构化数据；
- 无状态文件、无默认版本、断裂 current 链接；
- `.sdk-version` 的父目录查找和格式错误；
- 远程版本失败、超时和取消；
- Node.js 的 node/npm/npx 路径。

### MCP 测试

- Tools 能被列出且 Input Schema 正确；
- 合法参数返回结构化结果；
- 非法工具名、非法路径返回结构化错误；
- stdout 没有非 MCP 日志；
- `dry_run` 和 `confirm` 规则；
- 客户端断开时请求能取消。

### CLI 回归测试

- 原有 `list`、`current`、`remote`、`doctor`、`update --check` 行为不变；
- CLI 和 MCP 调用同一个 Service，不产生状态差异。

## 11. 分阶段交付

### 第一阶段

- 抽取只读 Service；
- 增加 `sdk mcp serve`；
- 注册只读 Tools；
- 注册 Resources；
- 增加 stdio 集成测试；
- 更新 README、CHANGELOG 和 MCP 客户端配置示例。

### 第二阶段

- 抽取安装、卸载、默认版本和迁移 Service；
- 增加 `dry_run` 和 `confirm`；
- 注册修改 Tools；
- 增加操作审计输出和取消测试。

### 暂不实现

- HTTP MCP Server；
- OAuth；
- 任意 Shell 执行；
- 自动替用户确认高风险操作；
- 远程集中式 SDK 管理服务。
