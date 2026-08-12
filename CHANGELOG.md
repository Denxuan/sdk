# 功能变更记录

## 当前版本

`sdk` 是一个面向 Java、Maven、Go 和 Node.js 的开发工具版本管理命令行程序。

## 未发布

- 新增 `sdk mcp serve`，通过官方 Go MCP SDK 以 stdio 提供只读工具和资源。
- MCP 工具、参数、资源和错误提示增加中英文描述，中文 AI 客户端可以直接理解工具用途。
- 抽取共享只读服务层，供 CLI 与 MCP 查询已安装版本、当前版本、远程版本、路径、项目版本、doctor 和更新检查。
- 新增 MCP 服务协议测试和共享服务测试。

## 已实现功能

### 工具与版本查询

- 支持 Java、Maven、Go、Node.js 四种工具。
- `sdk remote <tool>` / `sdk available <tool>`：从官方源列出可用正式版本。
- 自动过滤 Alpha、Beta、RC、Milestone、Snapshot、EA 等预发布版本。
- Java 显示每个功能版本线的最新正式 Temurin 版本，例如 `25.0.4`。
- Java、Node.js 的 LTS 版本附加 `LTS` 标识。
- Java 列表按远程平台资产判断可用性；没有当前平台正式包的版本不会显示。

### 安装与版本切换

- `sdk install <tool> [version]`：下载、解压并安装指定版本。
- 未指定版本时，Java 和 Node.js 优先安装最新 LTS；Maven 和 Go 安装最新正式版本。
- `sdk install <tool> <version> --path <directory>`：登记已有的本地安装目录。
- `sdk default <tool> <version>` / `sdk use <tool> <version>`：设置默认版本。
- `sdk current [tool]`：显示全部已下载工具的当前版本，或显示单个工具的当前版本。
- `sdk list [tool]`：查看已管理版本并标记默认版本。
- `sdk uninstall <tool> <version>`：卸载非默认版本。
- Java 在 macOS 上安装后会规整目录，使版本目录本身就是 `JAVA_HOME`，例如 `~/.sdk/tools/java/25.0.4/bin/java`。
- 下载完成后强制校验官方摘要：Java、Go、Node.js 使用 SHA-256；Maven 使用 SHA-512。校验失败的包不会解压或安装。

### 环境变量与 Shell 集成

- 每个工具都有 `~/.sdk/tools/<tool>/current` 软链接，指向当前默认版本。
- `sdk env` 输出 `JAVA_HOME`、`MAVEN_HOME`、`M2_HOME`、`GOROOT`、`NODE_HOME` 和对应的 `PATH`。
- `sdk doctor`：检查安装目录、`current` 软链接、当前二进制及环境变量配置。
- `sdk path <tool>`：输出当前工具目录；`sdk which <tool>`：输出当前工具实际二进制路径。
- 项目级 `.sdk-version`：`sdk project init` 根据全局默认版本创建文件，`sdk project set <tool> <version>` 设置项目版本，`sdk env --project` 使用最近项目文件覆盖全局版本；zsh 初始化后切换目录会自动刷新环境。
- `sdk setup zsh` 会在 `$SDK_HOME/init.zsh` 创建 Shell 函数和目录切换钩子，并以幂等方式在 `~/.zshrc` 中引入该脚本。
- 初始化后的 `sdk()` shell 函数会在 `default`、`use`、`install`、`update` 成功后自动刷新当前终端的环境变量，无需手动 `source ~/.zshrc`。

### 更新与下载可靠性

- `sdk update [tool]`：更新全部已管理工具，或仅更新指定工具。
- `sdk update [tool] --check`：仅检查可用更新，不下载、不切换版本或删除旧版本。
- `sdk update nodejs`：升级后自动迁移旧默认 Node.js 中第三方 `npm -g` 包到新版本；跳过 npm 与 corepack 自带组件。
- `sdk update maven`：迁移旧 Maven 版本中的 `conf/settings.xml` 和 `conf/toolchains.xml`，并保留新版本默认配置备份。
- Go 的 `go env -w` 配置位于用户目录，本来就是跨 Go 版本共享；Java 新版本的 `cacerts` 不会被旧文件覆盖。
- `sdk env` 会将 `GOBIN` 或 `GOPATH/bin` 加入 PATH，确保 `go install` 安装的命令行工具可以直接执行。
- 更新后会询问是否将新版本设为默认版本。
- 更新后会询问是否删除旧的 SDK 管理版本；手工通过 `--path` 登记的目录不会删除，当前默认版本也不会删除。
- 下载过程显示进度，包括百分比、已下载大小和总大小。
- 下载遇到临时网络错误、408、429、5xx 或中断时，自动进行最多 3 次指数退避重试。
- TAR 解压会安全保留归档内的相对软链接，确保 Node.js 安装同时包含 `npm`、`npx` 和 `corepack`。

### SDK 自身更新

- `sdk selfupdate [version]`：从 GitHub Release 下载当前系统匹配的 SDK 发布包，并原子替换自身二进制。
- 不传版本时更新到最新 Release；传入版本时更新到指定 Release。
- 通过 Homebrew 安装时应使用 `brew upgrade sdk`，而不是自身更新。

### 发布打包

- `scripts/package-release.sh <version>`：构建 macOS、Linux、Windows 的预编译发布包和 SHA-256 清单。
- 预编译包可直接上传 GitHub Release，供 `sdk selfupdate` 和 Homebrew Formula 下载，无需在用户电脑上编译 Go。

## 存储位置

默认根目录为 `~/.sdk`，可通过 `SDK_HOME` 环境变量覆盖。

```text
~/.sdk/
├── state.json
└── tools/
    ├── java/<version>/
    ├── maven/<version>/
    ├── go/<version>/
    └── nodejs/<version>/
```

## 后续建议

- 下载缓存与断点续传。
- 安装后的可执行文件验证。
- 远程版本索引缓存。
- 项目级 `.sdk.json` 版本配置。
