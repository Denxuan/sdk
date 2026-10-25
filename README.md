# sdk

`sdk` 是一个参考 SDKMAN 使用方式的开发工具版本管理 CLI。管理范围包括 Java（默认 Eclipse Temurin）、Maven、Maven mvnd、Gradle、Rust、Go 和 Node.js。

## 文档

- [功能变更记录](CHANGELOG.md)
- [发布与 Homebrew 说明](RELEASE.md)

## 快速开始

```bash
go build -o sdk .
./sdk install go 1.26.4
./sdk install java       # 安装最新 LTS Java
./sdk install java 26.0.2-zulu     # 安装 Azul Zulu JDK
./sdk install java 26.0.2-tem      # 安装 Eclipse Temurin
./sdk install mvnd       # 安装最新 Maven mvnd
./sdk install gradle     # 安装最新 Gradle
./sdk install rust       # 通过 rustup 安装 stable toolchain
./sdk default go 1.26.4
./sdk list go
./sdk current go
./sdk uninstall go 1.26.4
./sdk remote nodejs
```

安装并选择默认版本后，执行下面的命令一次，将当前版本发布到 shell 环境变量：

```bash
eval "$(./sdk env)"
```

也可只配置一次，让新的 zsh 终端自动加载当前版本，并在切换目录及相关版本命令成功后自动刷新当前终端环境：

```bash
./sdk setup zsh
source ~/.zshrc
```

Bash 用户则执行 `./sdk setup bash` 后运行 `source ~/.bashrc`。Windows PowerShell 用户执行 `sdk setup powershell`，然后重启 PowerShell。

Windows PowerShell 下可以用 `go build -o sdk.exe .` 构建，再通过 `.\sdk.exe install go 1.26.4` 和 `.\sdk.exe setup powershell` 安装工具并启用自动环境切换；重启后直接使用 `sdk`。

配置完成后请使用 `sdk default java 21.0.12`，不要使用 `./sdk default ...`；前者是 shell 函数，能自动应用新的环境变量。

macOS/Linux 上 `sdk default` 会更新每个工具目录中的 `current` 软链接，例如 `~/.sdk/tools/java/current`；Windows 上默认版本保存在 SDK 状态中，不需要创建软链接。Java 安装目录会作为 `JAVA_HOME`，例如 `~/.sdk/tools/java/25.0.4/bin/java` 可直接执行。`sdk env` 在 Windows 输出 PowerShell 命令，其他系统输出 POSIX shell 命令。

项目可用 `.sdk-version` 固定工具版本：`sdk project init` 从当前默认版本创建文件，`sdk project set <tool> <version>` 设置当前目录的版本，`sdk project install` 安装最近配置文件中缺少的版本但不改全局默认，`sdk project status` 检查版本是否已安装且主执行文件可运行，`sdk project unset <tool>` 移除单个工具版本，`sdk project remove` 删除当前目录或最近父目录中的配置文件。`sdk env --project` 可手动应用项目配置。

Rust 由官方 `rustup` 管理。执行 `sdk install rust <toolchain>` 时，如果系统没有 `rustup`，SDK 会自动下载安装 `rustup-init`；也可以直接透传命令，例如 `sdk rustup target add aarch64-apple-darwin`。

Java 支持 Eclipse Temurin 和 Azul Zulu JDK：版本后缀分别是 `-tem` 和 `-zulu`。Zulu 的具体包特性由内部下载策略选择，不作为版本后缀的一部分。不指定版本时默认安装 Eclipse Temurin 的最新 LTS 版本；直接输入不带后缀的 Java 版本时也按 `-tem` 处理。旧的 `.fx-zulu` 后缀不再支持。

状态默认保存在 `~/.sdk/state.json`。开发和测试时可用 `SDK_HOME` 隔离状态：

```bash
SDK_HOME=/tmp/sdk ./sdk list
```

## 当前边界

- 自动环境集成支持 macOS/Linux 上的 Bash 与 Zsh，以及 Windows PowerShell；Windows CMD 尚未支持。
- 发布包包含 Windows amd64 CLI。PowerShell profile 会在目录切换时应用最近的 `.sdk-version`，并在 `sdk default/install/update/uninstall` 等命令后刷新环境。

## MCP 集成

SDK 提供只读的 Model Context Protocol 服务，方便 AI 客户端查询本机开发环境：

```bash
sdk mcp serve
```

服务使用 stdio 传输，支持列出已安装版本、当前版本、远程稳定版本、工具路径、项目版本、doctor、更新检查，以及 `sdk://state`、`sdk://current`、`sdk://doctor` 和 `sdk://project/.sdk-version` 资源。工具包括 Java、Maven、Maven mvnd、Gradle、Rust、Go 和 Node.js。将 `sdk mcp serve` 配置为 MCP 客户端的 command 即可连接；当前阶段不会通过 MCP 修改或删除本机安装。

MCP 工具的名称保持英文，便于客户端稳定调用；工具描述、参数说明、资源名称和错误提示同时提供中文与英文，因此中文 AI 客户端可以直接理解并调用。例如，`sdk_available_versions` 的含义是“列出指定工具的远程正式稳定版本”。

## 命令约定

| 命令 | 作用 |
| --- | --- |
| `sdk remote <tool> [--refresh]` / `sdk available <tool> [--refresh]` | 显示官方远程版本并标记本机状态；`--refresh` 强制联网并更新缓存。 |
| `sdk install <tool> [version]` | 下载并安装官方发布包。不传版本时优先最新 LTS，否则使用最新正式版；可加 `--path <dir>` 登记本机已有安装。 |
| `sdk update [tool]` | 不带参数时更新所有已管理工具；指定工具时仅更新该工具。 |
| `sdk list [tool]` | 显示 SDK 已管理的本机版本。 |
| `sdk default <tool> <version>` | 设置全局默认版本。`use` 是兼容别名。 |
| `sdk current [tool]` | 不带参数时显示所有已下载工具的当前版本；指定工具时仅显示该工具。 |
| `sdk uninstall <tool> <version>` | 卸载版本；如果是默认版本，同时清除默认设置和对应环境变量。 |
| `sdk uninstall rustup` | 删除 rustup 以及 Rust toolchain。 |
| `sdk env [--project] [--powershell]` | 输出当前版本对应的环境变量与 `PATH` 设置；PowerShell 使用 `--powershell`。 |
| `sdk clean` | 删除 SDK 工具目录中的未完成下载，不影响已安装版本。 |
| `sdk project status` | 检查最近 `.sdk-version` 中的版本是否已安装且可运行。 |

下载因网络中断失败后，重新执行同一 `sdk install` 或 `sdk update` 命令会尝试从已下载位置续传；如果下载源不支持 HTTP Range，会自动从头下载。完成后仍会校验官方 checksum。

远程版本列表会在 SDK home 中缓存一小时，网络不可用时会使用已有缓存。

## 自身更新

`sdk selfupdate` 会下载 GitHub 最新 Release 中与当前系统匹配的发布包，并替换当前 `sdk` 二进制。也可指定版本：

```bash
sdk selfupdate
sdk selfupdate v0.0.2
```

通过 Homebrew 安装时请使用 `brew upgrade sdk`，而不是 `sdk selfupdate`。

下载完成后，`sdk install` 会询问是否将该版本设为默认版本；输入 `y` 或 `yes` 即可确认。

`sdk update` 成功安装新版本后，会询问是否删除旧的 SDK 管理版本。手工通过 `--path` 登记的目录不会被删除，当前默认版本也会被保留。

## 开发

发布包与支持的目标平台见 [RELEASE.md](RELEASE.md)。环境自动切换支持 macOS/Linux 上的 Bash、Zsh，以及 Windows PowerShell。
