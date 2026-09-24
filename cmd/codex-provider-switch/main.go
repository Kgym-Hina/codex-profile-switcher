package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"codex-profile-switcher/internal/app"
	"codex-profile-switcher/internal/config"
	"codex-profile-switcher/internal/tui"
)

var version = "dev"

type options struct {
	configPath     string
	codexHome      string
	mode           string
	list           bool
	help           bool
	showVersion    bool
	explicitConfig bool
}

func main() {
	options, err := parseArgs(os.Args[1:])
	if err != nil {
		fatal(err)
	}
	if options.help {
		usage()
		return
	}
	if options.showVersion {
		fmt.Println(version)
		return
	}
	if !options.explicitConfig {
		created, err := config.EnsureDefault(options.configPath)
		if err != nil {
			fatal(err)
		}
		if created {
			fmt.Printf("已生成默认配置: %s\n", options.configPath)
		}
	}

	service := app.Service{
		ConfigPath: options.configPath,
		CodexHome:  options.codexHome,
		Home:       userHome(),
	}
	if options.list {
		profiles, err := service.Profiles()
		if err != nil {
			fatal(err)
		}
		for _, profile := range profiles {
			fmt.Println(profile.Name)
		}
		return
	}
	if options.mode != "" {
		if err := service.Switch(options.mode); err != nil {
			fatal(err)
		}
		fmt.Printf("切换完成\n  profile: %s\n", options.mode)
		return
	}

	actions := tui.Actions{
		Reload:               service.Statuses,
		Switch:               service.Switch,
		Add:                  service.Add,
		Edit:                 service.Edit,
		Delete:               service.Delete,
		Restart:              restartCodex,
		GetRestartPreference: service.RestartPreference,
		SetRestartPreference: service.SetRestartPreference,
	}
	if err := tui.Run(actions, os.Stdin, os.Stdout); err != nil && !errors.Is(err, io.EOF) {
		fatal(err)
	}
}

func restartCodex() error {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("osascript", "-e", `tell application "Codex" to quit`).Run()
		return exec.Command("open", "-a", "Codex").Start()
	case "windows":
		_ = exec.Command("taskkill", "/IM", "codex.exe", "/T", "/F").Run()
		path, err := exec.LookPath("codex.exe")
		if err != nil {
			return err
		}
		return exec.Command(path).Start()
	default:
		_ = exec.Command("pkill", "-x", "codex").Run()
		path, err := exec.LookPath("codex")
		if err != nil {
			return err
		}
		return exec.Command(path).Start()
	}
}

func parseArgs(args []string) (options, error) {
	result := options{
		configPath: defaultConfigPath(),
		codexHome:  filepath.Join(userHome(), ".codex"),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			result.help = true
		case "--version", "-v":
			result.showVersion = true
		case "--list":
			result.list = true
		case "--config":
			if i+1 >= len(args) {
				return options{}, fmt.Errorf("--config 需要一个文件路径")
			}
			i++
			result.explicitConfig = true
			result.configPath = args[i]
		case "--codex-home":
			if i+1 >= len(args) {
				return options{}, fmt.Errorf("--codex-home 需要一个目录路径")
			}
			i++
			result.codexHome = args[i]
		default:
			if len(args[i]) > 1 && args[i][0] == '-' {
				return options{}, fmt.Errorf("未知参数: %s", args[i])
			}
			if result.mode != "" {
				return options{}, fmt.Errorf("只允许指定一个 profile")
			}
			result.mode = args[i]
		}
	}
	return result, nil
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
	fmt.Println("用法: codex-provider-switch [profile] [--config PATH] [--codex-home PATH]")
	fmt.Println("无参数进入 TUI；支持切换以及新增、编辑、删除 profile。")
	fmt.Println("选项: --list 列出 profile，--version 显示版本，--help 显示帮助。")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
