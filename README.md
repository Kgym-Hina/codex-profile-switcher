# Codex Profile Switcher

使用 Go 编写的 Codex profile 切换工具。它会同步更新 Codex 配置中的 `model_provider` 和 `auth.json`。

## 当前功能

### Profile 管理

- 无参数运行时进入交互式 TUI。
- 使用上下箭头选择 profile，按 Enter 切换 provider 和账号。
- 在 TUI 内新增、编辑、删除 profile。
- 编辑时可以修改 profile 名称、provider 和认证文件路径等全部 profile 字段。
- 删除 profile 只删除配置，不删除认证文件；当前使用中的 profile 需要先切换后才能删除。
- 自动显示 provider、认证文件是否可用、当前使用状态和待同步状态。
- 配置文件不存在时自动创建默认配置；profile 的认证快照不存在时创建为空文件并显示为不可用，不会伪造或复制 API key。
- 支持官方账号认证（`tokens` 等 Codex token 数据）和第三方 API key 认证。
- 新增或编辑 profile 时可以设置认证类型、API endpoint 和默认模型；API key 只写入该 profile 的认证文件，不写入 presets 配置。

### Codex 文件同步

- 切换前把当前 `~/.codex/auth.json` 的最新内容写回原 profile 认证文件，避免 token 刷新丢失。
- 将目标 profile 的认证文件写入当前 `auth.json`。
- 第三方 API profile 切换时，将目标 provider 写入 `config.toml` 的 `model_provider`。
- 新增第三方 API profile 时，才创建对应的 `[profiles."名称"]` 和 `[model_providers.<provider>]` 配置；provider 已存在时保留原有 `name`、`wire_api`、`requires_openai_auth` 和 `base_url`。
- 官方 profile 不创建 provider block，切换时移除 `model_provider`，只更新当前 `auth.json`。
- 加载或切换已有 profile 不会把它们重新写入 `config.toml`。
- 配置、认证文件和 profile 配置均使用原子写入。
- 切换失败时恢复 profile 标记和 `config.toml`；认证写入失败时恢复原配置。
- 支持绝对路径、`~/` 路径和相对于 profile 配置文件的认证路径。

### 历史会话修复

- 切换 profile 后自动修复历史会话的 provider 归属标记。
- 更新 `sessions/` 和 `archived_sessions/` 中 rollout JSONL 第一条 `session_meta` 记录的 `model_provider`。
- 如果系统提供 `sqlite3`，同时更新 Codex SQLite 线程索引中的 `model_provider`。
- 只修改索引和元数据，不修改对话正文。

### 重启控制

- TUI 切换完成后询问是否立即重启 Codex。
- 支持本次选择“重启”或“不重启”。
- 支持记住“重启”或“不重启”的偏好，后续切换自动执行。
- macOS 使用 Codex 应用重启，Windows 使用 `codex.exe`，其他系统使用 `codex` 命令。

### 命令行和发布

- 支持 `--list`、`--version`、`--help`、显式 profile、`--config` 和 `--codex-home`。
- 支持 Go 源码运行、当前平台构建和 macOS arm64/amd64 构建。
- 提供 Homebrew Cask 安装、升级和卸载方式。
- 推送 `v*` tag 后，GitHub Actions 自动构建 macOS 安装包、生成 GitHub Release、校验和并更新 Homebrew Cask。

## 使用

在项目根目录执行：

```bash
go run ./cmd/codex-provider-switch
```

TUI 快捷键：

- `↑` / `↓`：选择 profile
- `Enter`：切换到选中的 profile
- `t`：选择模型并发送一次 `hello` 测试请求
- `a`：新增 profile；可选择官方认证或第三方 API，并填写 endpoint、模型和 API key
- `e`：编辑选中的 profile
- `d`：删除选中的 profile（保留认证文件）
- `q` 或 `Esc`：退出

切换完成后，TUI 会询问是否重启 Codex：`r` 重启、`n` 不重启，`a`/`d` 分别记住重启或不重启。历史会话修复只更新 `session_meta` 的 `model_provider` 和线程索引，不修改对话正文。

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
      "auth_file": "~/.codex/auth.official.json",
      "auth_type": "official",
      "model": "gpt-5"
    }
  }
}
```

`active_profile` 由程序自动维护，用于可靠记录当前 profile。旧版配置首次运行时会根据 `model_provider` 和 `auth.json` 自动识别并补充该字段。

`provider` 会自动写入 `config.toml` 的 `model_provider`，`auth_file` 是对应的认证文件路径。`auth_type` 可设为 `official` 或 `api`；第三方 profile 可以额外设置 `base_url` 和 `model`。相对路径以配置文件所在目录为基准，也支持绝对路径和 `~/`。切换时，程序先把当前 `auth.json` 保存到原 profile，再载入目标 profile 的认证文件，因此 Codex 运行期间发生的 token 更新不会丢失。

程序会在配置文件所在目录创建缺失的父目录和 profile 认证目录。缺失的 profile 认证文件会创建为空文件，空文件不会被标记为可用，也不能用于切换或模型测试。API profile 在 TUI 中填写 API key 后，程序会把它写入对应认证文件的 `OPENAI_API_KEY` 字段。

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

## Homebrew 安装（macOS）

项目通过 Homebrew Cask 提供 macOS 安装包，支持 Apple Silicon（arm64）和 Intel（x86_64）。首次安装时添加 Tap，然后安装 Cask：

```bash
brew tap kgym-hina/codex-profile-switcher https://github.com/Kgym-Hina/codex-profile-switcher.git
brew install --cask codex-provider-switcher
```

安装完成后可以直接运行：

```bash
codex-provider-switch
```

升级到最新版本：

```bash
brew update
brew upgrade --cask codex-provider-switcher
```

卸载程序（不会删除 `~/.codex` 中的认证文件和 profile 配置）：

```bash
brew uninstall --cask codex-provider-switcher
```

如果只想移除 Tap：

```bash
brew untap kgym-hina/codex-profile-switcher
```

### 发布 Homebrew 版本

推送形如 `v0.2.0` 的 Git tag 后，GitHub Actions 会自动：

1. 构建 macOS Apple Silicon 和 Intel 版本
2. 创建 GitHub Release 并上传归档包与 SHA256 校验文件
3. 更新 `Casks/codex-provider-switcher.rb` 并提交回 `main`

发布新版本：

```bash
git tag v0.2.0
git push origin v0.2.0
```
