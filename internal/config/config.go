package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

//go:embed default_provider_presets.json
var defaultPresets []byte

type Profile struct {
	Name     string
	Provider string `json:"provider"`
	AuthFile string `json:"auth_file"`
}

type file struct {
	Presets map[string]Profile `json:"presets"`
}

func EnsureDefault(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("检查预设配置失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, fmt.Errorf("创建配置目录失败: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("创建默认预设配置失败: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(defaultPresets); err != nil {
		return false, fmt.Errorf("写入默认预设配置失败: %w", err)
	}
	return true, nil
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
