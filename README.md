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

默认配置文件是 `config/provider-presets.json`：

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

## 构建

构建当前平台版本：

```bash
go build -o codex-provider-switch ./cmd/codex-provider-switch
```

构建 macOS Apple Silicon（arm64）版本：

```bash
GOOS=darwin GOARCH=arm64 go build -o codex-provider-switch-darwin-arm64 ./cmd/codex-provider-switch
```

运行编译后的版本时，请从项目根目录执行，以便自动找到 `config/provider-presets.json`：

```bash
./codex-provider-switch-darwin-arm64
```

## 开发检查

```bash
go vet ./...
go build ./...
```
