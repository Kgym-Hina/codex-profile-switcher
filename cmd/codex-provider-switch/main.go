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
	configPath, codexHome, list, help, mode, err := parseArgs(os.Args[1:])
	if err != nil {
		fatal(err)
	}
	if help {
		usage()
		return
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
		statuses, err := switcher.Inspect(profiles, configPath, codexHome, os.Getenv("HOME"))
		if err != nil {
			fatal(err)
		}
		if err := tui.Run(statuses, func(index int) error {
			return switcher.Apply(profiles[index], configPath, codexHome, os.Getenv("HOME"))
		}, os.Stdin, os.Stdout); err != nil {
			fatal(err)
		}
		return
	}
	for _, profile := range profiles {
		if profile.Name == mode {
			if err := switcher.Apply(profile, configPath, codexHome, os.Getenv("HOME")); err != nil {
				fatal(err)
			}
			fmt.Printf("切换完成\n  mode: %s\n  provider: %s\n", profile.Name, profile.Provider)
			return
		}
	}
	fatal(fmt.Errorf("预设模式不存在: %s", mode))
}

func parseArgs(args []string) (string, string, bool, bool, string, error) {
	defaultConfig := defaultConfigPath()
	defaultHome := filepath.Join(os.Getenv("HOME"), ".codex")
	configPath, codexHome := defaultConfig, defaultHome
	list, help := false, false
	mode := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			help = true
		case "--list":
			list = true
		case "--config":
			if i+1 >= len(args) {
				return "", "", false, false, "", fmt.Errorf("--config 需要一个文件路径")
			}
			i++
			configPath = args[i]
		case "--codex-home":
			if i+1 >= len(args) {
				return "", "", false, false, "", fmt.Errorf("--codex-home 需要一个目录路径")
			}
			i++
			codexHome = args[i]
		default:
			if len(args[i]) > 1 && args[i][0] == '-' {
				return "", "", false, false, "", fmt.Errorf("未知参数: %s", args[i])
			}
			if mode != "" {
				return "", "", false, false, "", fmt.Errorf("只允许指定一个 mode")
			}
			mode = args[i]
		}
	}
	return configPath, codexHome, list, help, mode, nil
}

func defaultConfigPath() string {
	relative := filepath.Join("config", "provider-presets.json")
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Dir(executable)
		for i := 0; i < 3; i++ {
			candidate := filepath.Join(dir, relative)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			dir = filepath.Dir(dir)
		}
	}
	return relative
}

func usage() {
	fmt.Println("用法: codex-provider-switch [mode] [--config PATH] [--codex-home PATH]")
	fmt.Println("无参数进入 TUI；支持 ↑/↓ 选择、Enter 切换、q/Esc 退出。")
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "错误:", err); os.Exit(1) }
