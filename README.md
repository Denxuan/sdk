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

也可只配置一次，让新的 zsh 终端自动加载当前版本，并在 `sdk default`、`sdk use`、`sdk install`、`sdk update` 成功后自动刷新当前终端环境：

```bash
./sdk setup zsh
source ~/.zshrc
```

配置完成后请使用 `sdk default java 21.0.12`，不要使用 `./sdk default ...`；前者是 shell 函数，能自动应用新的环境变量。

`sdk default` 会更新每个工具目录中的 `current` 软链接，例如 `~/.sdk/tools/java/current`。Java 安装会被规整为版本目录就是 `JAVA_HOME`，因此 `~/.sdk/tools/java/25.0.4/bin/java` 可直接执行。`sdk env` 会输出 `JAVA_HOME`、`MAVEN_HOME`、`MVND_HOME`、`GRADLE_HOME`、`GOROOT`、`NODE_HOME`，并将 Rust 的 `~/.cargo/bin` 加入 `PATH`。

Rust 由官方 `rustup` 管理。执行 `sdk install rust <toolchain>` 时，如果系统没有 `rustup`，SDK 会自动下载安装 `rustup-init`；也可以直接透传命令，例如 `sdk rustup target add aarch64-apple-darwin`。

Java 支持 Eclipse Temurin 和 Azul Zulu JDK：版本后缀分别是 `-tem` 和 `-zulu`。Zulu 的具体包特性由内部下载策略选择，不作为版本后缀的一部分。不指定版本时默认安装 Eclipse Temurin 的最新 LTS 版本；直接输入不带后缀的 Java 版本时也按 `-tem` 处理。旧的 `.fx-zulu` 后缀不再支持。

状态默认保存在 `~/.sdk/state.json`。开发和测试时可用 `SDK_HOME` 隔离状态：

```bash
SDK_HOME=/tmp/sdk ./sdk list
```

## 当前边界

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
| `sdk remote <tool>` / `sdk available <tool>` | 显示官方远程版本，并标记本机状态。 |
| `sdk install <tool> [version]` | 下载并安装官方发布包。不传版本时优先最新 LTS，否则使用最新正式版；可加 `--path <dir>` 登记本机已有安装。 |
| `sdk update [tool]` | 不带参数时更新所有已管理工具；指定工具时仅更新该工具。 |
| `sdk list [tool]` | 显示 SDK 已管理的本机版本。 |
| `sdk default <tool> <version>` | 设置全局默认版本。`use` 是兼容别名。 |
| `sdk current [tool]` | 不带参数时显示所有已下载工具的当前版本；指定工具时仅显示该工具。 |
| `sdk uninstall <tool> <version>` | 卸载版本；如果是默认版本，同时清除默认设置和对应环境变量。 |
| `sdk env` | 输出当前版本对应的环境变量与 `PATH` 设置。 |

## 自身更新

`sdk selfupdate` 会下载 GitHub 最新 Release 中与当前系统匹配的发布包，并替换当前 `sdk` 二进制。也可指定版本：

```bash
sdk selfupdate
sdk selfupdate v0.0.2
```

通过 Homebrew 安装时请使用 `brew upgrade sdk`，而不是 `sdk selfupdate`。

下载完成后，`sdk install` 会询问是否将该版本设为默认版本；输入 `y` 或 `yes` 即可确认。

`sdk update` 成功安装新版本后，会询问是否删除旧的 SDK 管理版本。手工通过 `--path` 登记的目录不会被删除，当前默认版本也会被保留。

## 开发路线

当前版本已经能够下载、解压并管理多种工具的官方发布包，并通过 rustup 管理 Rust toolchain。下一阶段将补充安装缓存和项目级 `.sdk.json` 解析。
