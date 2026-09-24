package tui

import (
	"fmt"
	"io"
	"unicode/utf8"

	"codex-profile-switcher/internal/switcher"
)

func render(out io.Writer, statuses []switcher.Status, selected int, message string) {
	fmt.Fprint(out, "\x1b[H\x1b[2J")
	fmt.Fprintln(out, "Codex Profile Switcher")
	fmt.Fprintln(out, "↑/↓ 选择   Enter 切换   t 测试模型   a 新增   e 编辑   d 删除   q/Esc 退出")
	fmt.Fprintln(out)
	if len(statuses) == 0 {
		fmt.Fprintln(out, "尚未配置 profile，按 a 新增。")
	} else {
		fmt.Fprintln(out, "  PROFILE                 PROVIDER             TYPE       AUTH                  状态")
		fmt.Fprintln(out, "  ─────────────────────────────────────────────────────────────────────────────────────")
		for i, status := range statuses {
			cursor := "  "
			if i == selected {
				cursor = "▸ "
			}
			availability := "可用"
			if !status.Available {
				availability = "缺少认证"
			}
			fmt.Fprintf(out, "%s%-22s %-20s %-10s %-21s %s\n",
				cursor,
				fit(status.Name, 22),
				fit(status.Provider, 20),
				fit(status.AuthType, 10),
				availability,
				status.Reason,
			)
		}
	}
	if message != "" {
		fmt.Fprintf(out, "\n%s\n", message)
	}
}

func fit(value string, width int) string {
	if utf8.RuneCountInString(value) <= width {
		return value
	}
	runes := []rune(value)
	return string(runes[:width-1]) + "…"
}
