package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

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
		Test:                 service.Test,
		CurrentModel:         service.CurrentModel,
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
		return restartCodexMac()
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

func restartCodexMac() error {
	output, err := exec.Command("osascript", "-e", `POSIX path of (path to application id "com.openai.codex")`).Output()
	if err != nil {
		return fmt.Errorf("定位 Codex 应用失败: %w", err)
	}
	appPath := filepath.Clean(strings.TrimSpace(string(output)))
	if appPath == "." || filepath.Ext(appPath) != ".app" {
		return fmt.Errorf("Codex 应用路径无效: %q", appPath)
	}
	pids, err := codexAppPIDs(appPath)
	if err != nil {
		return err
	}
	for _, pid := range pids {
		if err := exec.Command("kill", "-9", strconv.Itoa(pid)).Run(); err != nil {
			active, checkErr := codexAppPIDs(appPath)
			if checkErr != nil {
				return checkErr
			}
			if containsPID(active, pid) {
				return fmt.Errorf("强制退出 Codex 应用进程 %d 失败: %w", pid, err)
			}
		}
	}
	if err := waitForCodexApp(appPath, false, 15*time.Second); err != nil {
		return err
	}
	if err := exec.Command("open", "-n", "-a", appPath).Run(); err != nil {
		return fmt.Errorf("启动 Codex 应用失败: %w", err)
	}
	return waitForCodexApp(appPath, true, 15*time.Second)
}

func codexAppPIDs(appPath string) ([]int, error) {
	output, err := exec.Command("ps", "-axo", "pid=,comm=").Output()
	if err != nil {
		return nil, fmt.Errorf("检查 Codex 应用进程失败: %w", err)
	}
	return parseAppPIDs(output, appPath), nil
}

// A direct executable under Contents/MacOS is the app's main process.
// Codex CLI and Electron helpers under Frameworks are excluded.
func parseAppPIDs(output []byte, appPath string) []int {
	mainDir := filepath.Join(appPath, "Contents", "MacOS")
	var pids []int
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		separator := strings.IndexAny(line, " \t")
		if separator < 0 {
			continue
		}
		pid, err := strconv.Atoi(line[:separator])
		if err != nil {
			continue
		}
		command := strings.TrimSpace(line[separator:])
		if filepath.Dir(command) == mainDir {
			pids = append(pids, pid)
		}
	}
	return pids
}

func containsPID(pids []int, want int) bool {
	for _, pid := range pids {
		if pid == want {
			return true
		}
	}
	return false
}

func waitForCodexApp(appPath string, running bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pids, err := codexAppPIDs(appPath)
		if err != nil {
			return err
		}
		if (len(pids) > 0) == running {
			return nil
		}
		if time.Now().After(deadline) {
			if running {
				return fmt.Errorf("等待 Codex 应用启动超时")
			}
			return fmt.Errorf("等待 Codex 应用退出超时")
		}
		time.Sleep(250 * time.Millisecond)
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
