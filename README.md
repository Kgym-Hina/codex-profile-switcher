# Codex Profile Switcher

使用 Go 编写的 Codex profile 切换工具。它会同步更新 Codex 配置中的 `model_provider` 和 `auth.json`。

## 功能

- 无参数运行时进入 TUI
- 使用上下箭头选择 profile，按 Enter 切换账号
- 在 TUI 内新增、编辑和删除 profile
- 显示 provider、认证文件可用性和当前使用状态
- 支持 `--list`、显式 profile、`--config` 和 `--codex-home`
- 切换前自动把当前 `auth.json` 回写到原 profile 的认证文件
- 原子更新配置和认证文件，切换失败时自动回滚

## 使用

在项目根目录执行：

```bash
go run ./cmd/codex-provider-switch
```

TUI 快捷键：

- `↑` / `↓`：选择 profile
- `Enter`：切换到选中的 profile
- `a`：新增 profile；认证文件不存在时会复制当前 `auth.json`
- `e`：编辑选中的 profile
- `d`：删除选中的 profile（保留认证文件）
- `q` 或 `Esc`：退出

也可以使用命令行模式：

```bash
go run ./cmd/codex-provider-switch --list
go run ./cmd/codex-provider-switch --version
go run ./cmd/codex-provider-switch official
go run ./cmd/codex-provider-switch thirdparty-a --codex-home ~/.codex
```

## 配置

默认配置文件是 `~/.config/codex-provider-switcher/provider-presets.json`。首次运行时，程序会从内置模板自动生成该文件，不依赖项目目录或当前运行目录：

```json
{
  "active_profile": "official",
  "presets": {
    "official": {
      "provider": "openai",
      "auth_file": "~/.codex/auth.official.json"
    }
  }
}
```

`active_profile` 由程序自动维护，用于可靠记录当前 profile。旧版配置首次运行时会根据 `model_provider` 和 `auth.json` 自动识别并补充该字段。

`provider` 会自动写入 `config.toml` 的 `model_provider`，`auth_file` 是对应的认证文件路径。相对路径以配置文件所在目录为基准，也支持绝对路径和 `~/`。切换时，程序先把当前 `auth.json` 保存到原 profile，再载入目标 profile 的认证文件，因此 Codex 运行期间发生的 token 更新不会丢失。

当前使用中的 profile 不能直接删除，需要先切换到其他 profile。删除操作只移除 profile 配置，不删除认证文件。

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

推送形如 `v0.2.0` 的 Git tag 后，GitHub Actions 会自动：

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
git tag v0.2.0
git push origin v0.2.0
```
