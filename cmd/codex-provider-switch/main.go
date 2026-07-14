package main

import (
	"fmt"
	"os"
	"path/filepath"

	"codex-profile-switcher/internal/config"
	"codex-profile-switcher/internal/switcher"
	"codex-profile-switcher/internal/tui"
)

var version = "dev"

func main() {
	configPath, codexHome, list, help, mode, explicitConfig, err := parseArgs(os.Args[1:])
	if err != nil {
		fatal(err)
	}
	if help {
		usage()
		return
	}
	if !explicitConfig {
		created, err := config.EnsureDefault(configPath)
		if err != nil {
			fatal(err)
		}
		if created {
			fmt.Printf("已生成默认配置: %s\n", configPath)
		}
	}
	profiles, err := config.Load(configPath)
	if err != nil {
		fatal(err)
	}
	if list {
		for _, p := range profiles {
			fmt.Println(p.Name)
		}
		return
	}

	if mode == "" {
		home := userHome()
		statuses, err := switcher.Inspect(profiles, configPath, codexHome, home)
		if err != nil {
			fatal(err)
		}
		if err := tui.Run(statuses, func(index int) error {
			return switcher.Apply(profiles[index], configPath, codexHome, home)
		}, os.Stdin, os.Stdout); err != nil {
			fatal(err)
		}
		return
	}
	for _, profile := range profiles {
		if profile.Name == mode {
			if err := switcher.Apply(profile, configPath, codexHome, userHome()); err != nil {
				fatal(err)
			}
			fmt.Printf("切换完成\n  mode: %s\n  provider: %s\n", profile.Name, profile.Provider)
			return
		}
	}
	fatal(fmt.Errorf("预设模式不存在: %s", mode))
}

func parseArgs(args []string) (string, string, bool, bool, string, bool, error) {
	defaultConfig := defaultConfigPath()
	defaultHome := filepath.Join(userHome(), ".codex")
	configPath, codexHome := defaultConfig, defaultHome
	list, help := false, false
	mode := ""
	explicitConfig := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			help = true
		case "--list":
			list = true
		case "--config":
			if i+1 >= len(args) {
				return "", "", false, false, "", false, fmt.Errorf("--config 需要一个文件路径")
			}
			i++
			explicitConfig = true
			configPath = args[i]
		case "--codex-home":
			if i+1 >= len(args) {
				return "", "", false, false, "", false, fmt.Errorf("--codex-home 需要一个目录路径")
			}
			i++
			codexHome = args[i]
		default:
			if len(args[i]) > 1 && args[i][0] == '-' {
				return "", "", false, false, "", false, fmt.Errorf("未知参数: %s", args[i])
			}
			if mode != "" {
				return "", "", false, false, "", false, fmt.Errorf("只允许指定一个 mode")
			}
			mode = args[i]
		}
	}
	return configPath, codexHome, list, help, mode, explicitConfig, nil
}

func defaultConfigPath() string {
	return filepath.Join(userHome(), ".config", "codex-provider-switcher", "provider-presets.json")
}

func userHome() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return os.Getenv("HOME")
}

func usage() {
	fmt.Println("用法: codex-provider-switch [mode] [--config PATH] [--codex-home PATH]")
	fmt.Println("无参数进入 TUI；支持 ↑/↓ 选择、Enter 切换、q/Esc 退出。")
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "错误:", err); os.Exit(1) }
