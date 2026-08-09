package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed default_provider_presets.json
var defaultPresets []byte

type Profile struct {
	Name     string `json:"-"`
	Provider string `json:"provider"`
	AuthFile string `json:"auth_file"`
}

type Config struct {
	ActiveProfile string             `json:"active_profile,omitempty"`
	Presets       map[string]Profile `json:"presets"`
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
	if _, err := file.Write(defaultPresets); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return false, fmt.Errorf("写入默认预设配置失败: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return false, fmt.Errorf("保存默认预设配置失败: %w", err)
	}
	return true, nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取预设配置失败: %w", err)
	}
	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("解析预设配置失败: %w", err)
	}
	if parsed.Presets == nil {
		parsed.Presets = make(map[string]Profile)
	}
	for name, profile := range parsed.Presets {
		profile.Name = name
		if err := ValidateProfile(profile); err != nil {
			return nil, fmt.Errorf("预设 %q 无效: %w", name, err)
		}
		parsed.Presets[name] = profile
	}
	if parsed.ActiveProfile != "" {
		if _, ok := parsed.Presets[parsed.ActiveProfile]; !ok {
			parsed.ActiveProfile = ""
		}
	}
	return &parsed, nil
}

func Save(path string, value *Config) error {
	if value == nil {
		return fmt.Errorf("配置不能为空")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化预设配置失败: %w", err)
	}
	data = append(data, '\n')
	if err := atomicWrite(path, data, 0o600); err != nil {
		return fmt.Errorf("保存预设配置失败: %w", err)
	}
	return nil
}

func (c *Config) Profiles() []Profile {
	profiles := make([]Profile, 0, len(c.Presets))
	for name, profile := range c.Presets {
		profile.Name = name
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	return profiles
}

func (c *Config) SetProfile(oldName string, profile Profile) error {
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Provider = strings.TrimSpace(profile.Provider)
	profile.AuthFile = strings.TrimSpace(profile.AuthFile)
	if err := ValidateProfile(profile); err != nil {
		return err
	}
	if oldName != profile.Name {
		if _, exists := c.Presets[profile.Name]; exists {
			return fmt.Errorf("profile %q 已存在", profile.Name)
		}
	}
	if oldName != "" {
		if _, exists := c.Presets[oldName]; !exists {
			return fmt.Errorf("profile %q 不存在", oldName)
		}
		delete(c.Presets, oldName)
		if c.ActiveProfile == oldName {
			c.ActiveProfile = profile.Name
		}
	}
	c.Presets[profile.Name] = Profile{Provider: profile.Provider, AuthFile: profile.AuthFile}
	return nil
}

func (c *Config) DeleteProfile(name string) error {
	if name == c.ActiveProfile {
		return fmt.Errorf("当前使用中的 profile 不能删除，请先切换")
	}
	if _, exists := c.Presets[name]; !exists {
		return fmt.Errorf("profile %q 不存在", name)
	}
	delete(c.Presets, name)
	return nil
}

func ValidateProfile(profile Profile) error {
	if strings.TrimSpace(profile.Name) == "" {
		return fmt.Errorf("名称不能为空")
	}
	if strings.ContainsAny(profile.Name, "\r\n") {
		return fmt.Errorf("名称不能包含换行符")
	}
	if strings.TrimSpace(profile.Provider) == "" {
		return fmt.Errorf("provider 不能为空")
	}
	if strings.ContainsAny(profile.Provider, "\r\n") {
		return fmt.Errorf("provider 不能包含换行符")
	}
	if strings.TrimSpace(profile.AuthFile) == "" {
		return fmt.Errorf("auth 文件路径不能为空")
	}
	if strings.ContainsAny(profile.AuthFile, "\r\n") {
		return fmt.Errorf("auth 文件路径不能包含换行符")
	}
	return nil
}

func ResolveAuthPath(raw, configPath, home string) string {
	if raw == "~" {
		return home
	}
	if strings.HasPrefix(raw, "~/") {
		return filepath.Join(home, raw[2:])
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Join(filepath.Dir(configPath), raw)
}

func atomicWrite(path string, data []byte, fallbackMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".provider-presets.tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(fallbackMode); err != nil {
		_ = tmp.Close()
		return err
	}
	_, err = tmp.Write(data)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
