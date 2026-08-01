# sdk

`sdk` 是一个参考 SDKMAN 使用方式的开发工具版本管理 CLI。管理范围严格限定为 Java（默认 Eclipse Temurin）、Maven、Go 和 Node.js。

## 快速开始

```bash
go build -o sdk .
./sdk install go 1.26.4
./sdk install java       # 安装最新 LTS Java
./sdk default go 1.26.4
./sdk list go
./sdk current go
./sdk uninstall go 1.26.4
./sdk remote nodejs
```

安装并选择默认版本后，执行下面的命令一次，将受管理工具置于当前 shell 的 `PATH` 前方：

```bash
eval "$(./sdk env)"
```

`sdk default` 会生成 `~/.sdk/shims` 下的命令转发器。因此后续执行 `java`、`mvn`、`go`、`node`、`npm` 等命令时，会自动使用 SDK 选定的默认版本。

状态默认保存在 `~/.sdk/state.json`。开发和测试时可用 `SDK_HOME` 隔离状态：

```bash
SDK_HOME=/tmp/sdk ./sdk list
```

## 当前边界

## 命令约定

| 命令 | 作用 |
| --- | --- |
| `sdk remote <tool>` / `sdk available <tool>` | 显示官方远程版本，并标记本机状态。 |
| `sdk install <tool> [version]` | 下载并安装官方发布包。不传版本时优先最新 LTS，否则使用最新正式版；可加 `--path <dir>` 登记本机已有安装。 |
| `sdk list [tool]` | 显示 SDK 已管理的本机版本。 |
| `sdk default <tool> <version>` | 设置全局默认版本。`use` 是兼容别名。 |
| `sdk current [tool]` | 不带参数时显示所有已下载工具的当前版本；指定工具时仅显示该工具。 |
| `sdk uninstall <tool> <version>` | 解除非默认版本的管理登记。 |
| `sdk env` | 输出将 SDK shims 加入当前 shell `PATH` 的命令。 |

## 开发路线

当前版本已经能够下载、解压并管理四种工具的官方发布包。下一阶段将补充 SHA-256 校验、shims、`sdk use` 的当前 shell 集成和项目级 `.sdk.json` 解析。
