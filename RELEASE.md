# 发布 SDK

## 构建跨平台二进制包

在发布机器上安装 Go 后执行：

```bash
./scripts/package-release.sh v0.0.2
```

脚本会生成以下资产：

```text
dist/v0.0.2/
├── sdk_0.0.2_darwin_amd64.tar.gz
├── sdk_0.0.2_darwin_arm64.tar.gz
├── sdk_0.0.2_linux_amd64.tar.gz
├── sdk_0.0.2_linux_arm64.tar.gz
├── sdk_0.0.2_windows_amd64.zip
└── checksums.txt
```

这些归档名与 `sdk selfupdate` 的选择规则匹配。将它们作为 GitHub Release `v0.0.2` 的资产上传后，用户即可直接执行：

```bash
sdk selfupdate
sdk selfupdate v0.0.2
```

自身更新会下载预编译的发布包，不会在用户电脑上安装或调用 Go。

## Homebrew Formula

Homebrew Formula 应下载对应平台的预编译 Release 资产，而不是声明 `depends_on "go" => :build`。这样 `brew install` 只下载和安装二进制包。

不同系统和架构需要分别提供 Bottle，或者使用 Homebrew 的 `on_macos`、`on_linux` 与架构条件选择对应资产。
