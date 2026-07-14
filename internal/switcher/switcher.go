package switcher

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"codex-profile-switcher/internal/config"
)

type Status struct {
	config.Profile
	AuthPath  string
	Available bool
	Active    bool
	Reason    string
}

var providerLine = regexp.MustCompile(`(?m)^(model_provider\s*=\s*)"([^"]*)"(\s*)$`)

func Inspect(profiles []config.Profile, configPath, codexHome, home string) ([]Status, error) {
	configFile := filepath.Join(codexHome, "config.toml")
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取 Codex config.toml 失败: %w", err)
	}
	match := providerLine.FindSubmatch(configData)
	currentProvider := ""
	if len(match) > 0 {
		currentProvider = string(match[2])
	}
	currentAuth, authErr := os.ReadFile(filepath.Join(codexHome, "auth.json"))

	statuses := make([]Status, 0, len(profiles))
	for _, profile := range profiles {
		authPath := config.ResolveAuthPath(profile.AuthFile, configPath, home)
		status := Status{Profile: profile, AuthPath: authPath}
		profileAuth, err := os.ReadFile(authPath)
		if err != nil {
			status.Reason = "认证文件不存在"
			statuses = append(statuses, status)
			continue
		}
		status.Available = true
		status.Active = authErr == nil && currentProvider == profile.Provider && bytes.Equal(currentAuth, profileAuth)
		if !status.Active {
			status.Reason = "可切换"
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func Apply(profile config.Profile, configPath, codexHome, home string) error {
	if info, err := os.Stat(codexHome); err != nil || !info.IsDir() {
		return fmt.Errorf("Codex 目录不存在: %s", codexHome)
	}
	configFile := filepath.Join(codexHome, "config.toml")
	if _, err := os.Stat(configFile); err != nil {
		return fmt.Errorf("未找到 Codex config.toml: %w", err)
	}
	sourceAuth := config.ResolveAuthPath(profile.AuthFile, configPath, home)
	if _, err := os.Stat(sourceAuth); err != nil {
		return fmt.Errorf("缺少该模式对应的 auth 文件: %s", sourceAuth)
	}
	if err := updateProvider(configFile, profile.Provider); err != nil {
		return err
	}
	return replaceAuth(sourceAuth, filepath.Join(codexHome, "auth.json"))
}

func updateProvider(path, provider string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 config.toml 失败: %w", err)
	}
	if !providerLine.Match(data) {
		return fmt.Errorf("config.toml 中未找到 model_provider = \"...\" 行")
	}
	replaced := providerLine.ReplaceAllString(string(data), `${1}"`+provider+`"${3}`)
	if err := os.WriteFile(path, []byte(replaced), 0o600); err != nil {
		return fmt.Errorf("更新 config.toml 失败: %w", err)
	}
	return nil
}

func replaceAuth(source, target string) error {
	sourceAbs, _ := filepath.Abs(source)
	targetAbs, _ := filepath.Abs(target)
	if sourceAbs == targetAbs {
		return nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("读取认证文件失败: %w", err)
	}
	info, err := os.Stat(target)
	mode := os.FileMode(0o600)
	if err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".auth.json.tmp-*")
	if err != nil {
		return fmt.Errorf("创建认证临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("写入认证文件失败: %w", err)
	}
	backup := target + ".bak"
	_ = os.Remove(backup)
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("备份认证文件失败: %w", err)
		}
	}
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("替换认证文件失败: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}
