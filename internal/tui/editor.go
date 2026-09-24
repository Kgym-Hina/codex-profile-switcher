package tui

import (
	"fmt"
	"io"
	"strings"

	"codex-profile-switcher/internal/config"
	"codex-profile-switcher/internal/switcher"
)

func editForm(keys *keyReader, out io.Writer, initial config.Profile) (config.Profile, bool, error) {
	fmt.Fprint(out, "\x1b[H\x1b[2J\x1b[?25h")
	title := "新增 Profile"
	if initial.Name != "" {
		title = "编辑 Profile"
	}
	fmt.Fprintln(out, title)
	fmt.Fprintln(out, "按 Enter 确认每一项，Ctrl+U 清空当前项，Esc 取消")
	fmt.Fprintln(out)
	name, ok, err := keys.readLine(out, "名称: ", initial.Name)
	if err != nil || !ok {
		fmt.Fprint(out, "\x1b[?25l")
		return config.Profile{}, false, err
	}
	provider, ok, err := keys.readLine(out, "Provider: ", initial.Provider)
	if err != nil || !ok {
		fmt.Fprint(out, "\x1b[?25l")
		return config.Profile{}, false, err
	}
	authType := config.NormalizeAuthType(initial.AuthType, provider)
	authType, ok, err = keys.readLine(out, "认证类型 (official/api): ", authType)
	if err != nil || !ok {
		fmt.Fprint(out, "\x1b[?25l")
		return config.Profile{}, false, err
	}
	authType = strings.ToLower(strings.TrimSpace(authType))
	baseURL, ok, err := keys.readLine(out, "API Endpoint（official 可留空）: ", initial.BaseURL)
	if err != nil || !ok {
		fmt.Fprint(out, "\x1b[?25l")
		return config.Profile{}, false, err
	}
	model, ok, err := keys.readLine(out, "默认模型（可留空）: ", initial.Model)
	if err != nil || !ok {
		fmt.Fprint(out, "\x1b[?25l")
		return config.Profile{}, false, err
	}
	apiKey := ""
	if authType == "api" {
		apiKey, ok, err = keys.readLine(out, "API Key（留空保持现有）: ", "")
		if err != nil || !ok {
			fmt.Fprint(out, "\x1b[?25l")
			return config.Profile{}, false, err
		}
	}
	authFile := initial.AuthFile
	if authFile == "" {
		authFile = switcher.DefaultAuthFile(name)
	}
	authFile, ok, err = keys.readLine(out, "Auth 文件: ", authFile)
	fmt.Fprint(out, "\x1b[?25l")
	if err != nil || !ok {
		return config.Profile{}, false, err
	}
	return config.Profile{
		Name:     strings.TrimSpace(name),
		Provider: strings.TrimSpace(provider),
		AuthFile: strings.TrimSpace(authFile),
		AuthType: authType,
		BaseURL:  strings.TrimSpace(baseURL),
		Model:    strings.TrimSpace(model),
		APIKey:   strings.TrimSpace(apiKey),
	}, true, nil
}

func testModelForm(keys *keyReader, out io.Writer, initial string) (string, bool, error) {
	fmt.Fprint(out, "\x1b[H\x1b[2J\x1b[?25h")
	model, ok, err := keys.readLine(out, "测试模型: ", initial)
	fmt.Fprint(out, "\x1b[?25l")
	return strings.TrimSpace(model), ok, err
}

func confirmDelete(keys *keyReader, out io.Writer, name string) (bool, error) {
	fmt.Fprint(out, "\x1b[H\x1b[2J")
	fmt.Fprintf(out, "删除 profile %q？认证文件会保留。\n\n按 y 确认，其他键取消。\n", name)
	key, err := keys.readKey()
	if err != nil {
		return false, err
	}
	return key == "y", nil
}
