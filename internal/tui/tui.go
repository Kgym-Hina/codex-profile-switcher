package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"codex-profile-switcher/internal/switcher"
)

func Run(statuses []switcher.Status, apply func(int) error, in io.Reader, out io.Writer) error {
	if len(statuses) == 0 {
		return fmt.Errorf("未配置任何 profile")
	}
	reader := bufio.NewReader(in)
	selected := 0
	message := ""
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
	restoreTerminal := enableRawMode(in)
	defer restoreTerminal()
	for {
		render(out, statuses, selected, message)
		message = ""
		key, err := readKey(reader)
		if err != nil {
			return err
		}
		switch key {
		case "up":
			selected = (selected + len(statuses) - 1) % len(statuses)
		case "down":
			selected = (selected + 1) % len(statuses)
		case "enter":
			if !statuses[selected].Available {
				message = "该 profile 的认证文件不可用"
				continue
			}
			if err := apply(selected); err != nil {
				message = "切换失败: " + err.Error()
				continue
			}
			for i := range statuses {
				statuses[i].Active = i == selected
			}
			message = "已切换到 " + statuses[selected].Name
		case "q", "esc":
			return nil
		}
	}
}

func enableRawMode(in io.Reader) func() {
	file, ok := in.(*os.File)
	if !ok {
		return func() {}
	}
	command := exec.Command("stty", "-icanon", "-echo")
	command.Stdin = file
	if err := command.Run(); err != nil {
		return func() {}
	}
	return func() {
		restore := exec.Command("stty", "sane")
		restore.Stdin = file
		_ = restore.Run()
	}
}

func render(out io.Writer, statuses []switcher.Status, selected int, message string) {
	fmt.Fprint(out, "\x1b[H\x1b[2J")
	fmt.Fprintln(out, "Codex Profile Switcher")
	fmt.Fprintln(out, "↑/↓ 选择   Enter 切换   q/Esc 退出")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  PROFILE                 PROVIDER             AUTH                  状态")
	fmt.Fprintln(out, "  ───────────────────────────────────────────────────────────────────────")
	for i, status := range statuses {
		cursor := "  "
		if i == selected {
			cursor = "▸ "
		}
		availability := "可用"
		if !status.Available {
			availability = "缺少认证"
		}
		state := status.Reason
		if status.Active {
			state = "当前使用"
		}
		fmt.Fprintf(out, "%s%-22s %-20s %-21s %s\n", cursor, status.Name, status.Provider, availability, state)
	}
	if message != "" {
		fmt.Fprintf(out, "\n%s\n", message)
	}
}

func readKey(reader *bufio.Reader) (string, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return "", err
	}
	if b == 0x1b {
		b2, err := reader.ReadByte()
		if err != nil {
			return "esc", nil
		}
		if b2 != '[' {
			return "esc", nil
		}
		b3, err := reader.ReadByte()
		if err != nil {
			return "esc", nil
		}
		if b3 == 'A' {
			return "up", nil
		}
		if b3 == 'B' {
			return "down", nil
		}
		return "", nil
	}
	if b == '\r' || b == '\n' {
		return "enter", nil
	}
	return strings.ToLower(string(b)), nil
}
