# Codex Profile Switcher

使用 Go 编写的 Codex profile 切换工具。它会同步更新 Codex 配置中的 `model_provider` 和 `auth.json`。

## 功能

- 无参数运行时进入 TUI
- 使用上下箭头选择 profile，按 Enter 切换账号
- 显示 provider、认证文件可用性和当前使用状态
- 支持 `--list`、显式 profile、`--config` 和 `--codex-home`
- 使用临时文件替换认证文件，避免直接截断现有文件

## 使用

在项目根目录执行：

```bash
go run ./cmd/codex-provider-switch
```

TUI 快捷键：

- `↑` / `↓`：选择 profile
- `Enter`：切换到选中的 profile
- `q` 或 `Esc`：退出

也可以使用命令行模式：

```bash
go run ./cmd/codex-provider-switch --list
go run ./cmd/codex-provider-switch official
go run ./cmd/codex-provider-switch thirdparty-a --codex-home ~/.codex
```

## 配置

默认配置文件是 `~/.config/codex-provider-switcher/provider-presets.json`。首次运行时，程序会从内置模板自动生成该文件，不依赖项目目录或当前运行目录：

```json
{
  "presets": {
    "official": {
      "provider": "openai",
      "auth_file": "~/.codex/auth.official.json"
    }
  }
}
```

`provider` 会写入 `config.toml` 的 `model_provider`，`auth_file` 是对应的认证文件路径。相对路径以配置文件所在目录为基准，也支持绝对路径和 `~/`。

修改用户配置文件后重新启动程序即可生效。也可以通过 `--config PATH` 临时指定其他配置文件。

## 构建

构建当前平台版本：

```bash
go build -o codex-provider-switch ./cmd/codex-provider-switch
```

构建 macOS Apple Silicon（arm64）版本：

```bash
GOOS=darwin GOARCH=arm64 go build -o codex-provider-switch-darwin-arm64 ./cmd/codex-provider-switch
```

运行编译后的版本：

```bash
./codex-provider-switch-darwin-arm64
```

## 开发检查

```bash
go vet ./...
go build ./...
```

## Homebrew Cask

推送形如 `v0.1.0` 的 Git tag 后，GitHub Actions 会自动：

1. 构建 macOS Apple Silicon 和 Intel 版本
2. 创建 GitHub Release 并上传归档包与 SHA256 校验文件
3. 更新 `Casks/codex-provider-switcher.rb` 并提交回 `main`

首次发布完成后，可以通过个人 Tap 安装：

```bash
brew tap kgym-hina/codex-profile-switcher https://github.com/Kgym-Hina/codex-profile-switcher.git
brew install --cask codex-provider-switcher
```

发布新版本：

```bash
git tag v0.1.0
git push origin v0.1.0
```
