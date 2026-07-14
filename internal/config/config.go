package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Profile struct {
	Name     string
	Provider string `json:"provider"`
	AuthFile string `json:"auth_file"`
}

type file struct {
	Presets map[string]Profile `json:"presets"`
}

func Load(path string) ([]Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取预设配置失败: %w", err)
	}
	var parsed file
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("解析预设配置失败: %w", err)
	}
	profiles := make([]Profile, 0, len(parsed.Presets))
	for name, profile := range parsed.Presets {
		profile.Name = name
		if profile.Provider == "" || profile.AuthFile == "" {
			return nil, fmt.Errorf("预设 %q 缺少 provider 或 auth_file", name)
		}
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	return profiles, nil
}

func ResolveAuthPath(raw, configPath, home string) string {
	if raw == "~" {
		return home
	}
	if len(raw) > 2 && raw[:2] == "~/" {
		return filepath.Join(home, raw[2:])
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Join(filepath.Dir(configPath), raw)
}
