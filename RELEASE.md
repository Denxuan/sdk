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

## GitHub 自动发布

仓库已包含 [发布工作流](.github/workflows/release.yml)。在 GitHub 的 **Releases** 页面
发布一个 Release 后，GitHub Actions 会自动测试、交叉编译，并将全部压缩包、
`checksums.txt` 和 Homebrew Formula 上传到刚创建的 Release。

推荐先推送标签，再在网页中选择该标签创建 Release：

```bash
git tag v0.0.2
git push origin v0.0.2
```

也可以在 GitHub 的 **Actions → Release SDK → Run workflow** 中输入一个已经推送的标签，
为已有 Release 重新生成并覆盖附件。这也适用于工作流加入前已经发布的版本。

首次使用时，请确认仓库的 **Settings → Actions → General → Workflow permissions** 允许工作流
读写仓库内容；工作流已声明 `contents: write`，用来创建 Release 和上传附件。

## Homebrew Formula

创建一个单独的 Tap 仓库，例如 `Denxuan/homebrew-sdk`，并将每次 Release 自动生成的
`sdk.rb` 放到该仓库的 `Formula/sdk.rb`。这个 Formula 会按系统和 CPU 架构下载对应的
预编译包，且**没有** `depends_on "go" => :build`，所以不会下载或编译 Go。

发布工作流会把 `sdk.rb` 作为 Release 附件上传；也可以本地生成：

```bash
./scripts/package-release.sh v0.0.2
./scripts/generate-homebrew-formula.sh v0.0.2
```

随后提交到 Tap：

```bash
git clone git@github.com:Denxuan/homebrew-sdk.git
cp /path/to/sdk/dist/v0.0.2/sdk.rb homebrew-sdk/Formula/sdk.rb
cd homebrew-sdk
git add Formula/sdk.rb
git commit -m "sdk 0.0.2"
git push
```

用户安装和升级命令为：

```bash
brew tap Denxuan/sdk
brew install sdk
brew upgrade sdk
```

将 `Denxuan` 替换为实际的 GitHub 用户名或组织名。可通过 `brew audit --new --formula sdk`
和 `brew install --build-from-source ./Formula/sdk.rb` 在 Tap 仓库内验证 Formula；这里的
`--build-from-source` 只是 Homebrew 的验证参数，Formula 本身不会编译 Go。
