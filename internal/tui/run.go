package tui

import (
	"fmt"
	"io"

	"codex-profile-switcher/internal/config"
	"codex-profile-switcher/internal/switcher"
)

type Actions struct {
	Reload func() ([]switcher.Status, error)
	Switch func(string) error
	Add    func(config.Profile) error
	Edit   func(string, config.Profile) error
	Delete func(string) error
}

func Run(actions Actions, in io.Reader, out io.Writer) error {
	statuses, err := actions.Reload()
	if err != nil {
		return err
	}
	keys := newKeyReader(in)
	selected := 0
	message := ""
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
	restoreTerminal := enableRawMode(in)
	defer restoreTerminal()

	for {
		render(out, statuses, selected, message)
		message = ""
		key, err := keys.readKey()
		if err != nil {
			return err
		}
		switch key {
		case "up":
			if len(statuses) > 0 {
				selected = (selected + len(statuses) - 1) % len(statuses)
			}
		case "down":
			if len(statuses) > 0 {
				selected = (selected + 1) % len(statuses)
			}
		case "enter":
			if len(statuses) == 0 {
				message = "请先新增 profile"
				continue
			}
			if !statuses[selected].Available {
				message = "该 profile 的认证文件不可用"
				continue
			}
			if err := actions.Switch(statuses[selected].Name); err != nil {
				message = "切换失败: " + err.Error()
				continue
			}
			message = "已切换到 " + statuses[selected].Name
			statuses, selected, err = reload(actions, statuses[selected].Name, selected)
			if err != nil {
				return err
			}
		case "a":
			profile, ok, err := editForm(keys, out, config.Profile{})
			if err != nil {
				return err
			}
			if !ok {
				message = "已取消新增"
				continue
			}
			if err := actions.Add(profile); err != nil {
				message = "新增失败: " + err.Error()
				continue
			}
			message = "已新增 " + profile.Name
			statuses, selected, err = reload(actions, profile.Name, selected)
			if err != nil {
				return err
			}
		case "e":
			if len(statuses) == 0 {
				message = "没有可编辑的 profile"
				continue
			}
			oldName := statuses[selected].Name
			profile, ok, err := editForm(keys, out, statuses[selected].Profile)
			if err != nil {
				return err
			}
			if !ok {
				message = "已取消编辑"
				continue
			}
			if err := actions.Edit(oldName, profile); err != nil {
				message = "编辑失败: " + err.Error()
				continue
			}
			message = "已更新 " + profile.Name
			statuses, selected, err = reload(actions, profile.Name, selected)
			if err != nil {
				return err
			}
		case "d":
			if len(statuses) == 0 {
				message = "没有可删除的 profile"
				continue
			}
			name := statuses[selected].Name
			confirmed, err := confirmDelete(keys, out, name)
			if err != nil {
				return err
			}
			if !confirmed {
				message = "已取消删除"
				continue
			}
			if err := actions.Delete(name); err != nil {
				message = "删除失败: " + err.Error()
				continue
			}
			message = "已删除 " + name + "，认证文件已保留"
			statuses, selected, err = reload(actions, "", selected)
			if err != nil {
				return err
			}
		case "q", "esc":
			return nil
		}
	}
}

func reload(actions Actions, preferred string, selected int) ([]switcher.Status, int, error) {
	statuses, err := actions.Reload()
	if err != nil {
		return nil, 0, err
	}
	for i := range statuses {
		if statuses[i].Name == preferred {
			return statuses, i, nil
		}
	}
	if len(statuses) == 0 {
		return statuses, 0, nil
	}
	if selected >= len(statuses) {
		selected = len(statuses) - 1
	}
	return statuses, selected, nil
}
