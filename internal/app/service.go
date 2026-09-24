package app

import (
	"fmt"
	"strings"

	"codex-profile-switcher/internal/config"
	"codex-profile-switcher/internal/switcher"
)

type Service struct {
	ConfigPath string
	CodexHome  string
	Home       string
}

func (s Service) Profiles() ([]config.Profile, error) {
	value, err := config.Load(s.ConfigPath)
	if err != nil {
		return nil, err
	}
	return value.Profiles(), nil
}

func (s Service) Statuses() ([]switcher.Status, error) {
	value, err := s.load()
	if err != nil {
		return nil, err
	}
	return switcher.Inspect(value.Profiles(), value.ActiveProfile, s.ConfigPath, s.CodexHome, s.Home)
}

func (s Service) Switch(name string) error {
	value, err := s.load()
	if err != nil {
		return err
	}
	target, ok := profileByName(value, name)
	if !ok {
		return fmt.Errorf("profile %q 不存在", name)
	}
	var current *config.Profile
	if active, ok := profileByName(value, value.ActiveProfile); ok {
		current = &active
	}
	previousActive := value.ActiveProfile
	value.ActiveProfile = target.Name
	if err := config.Save(s.ConfigPath, value); err != nil {
		return err
	}
	if err := switcher.Apply(current, target, s.ConfigPath, s.CodexHome, s.Home); err != nil {
		value.ActiveProfile = previousActive
		if rollbackErr := config.Save(s.ConfigPath, value); rollbackErr != nil {
			return fmt.Errorf("%v；恢复当前 profile 标记也失败: %v", err, rollbackErr)
		}
		return err
	}
	if current != nil {
		fromProviders := []string{current.Provider, switcher.SessionProvider(*current)}
		toProvider := switcher.SessionProvider(target)
		seen := make(map[string]bool)
		for _, fromProvider := range fromProviders {
			if fromProvider == "" || fromProvider == toProvider || seen[fromProvider] {
				continue
			}
			seen[fromProvider] = true
			if err := switcher.RepairHistory(s.CodexHome, fromProvider, toProvider); err != nil {
				return fmt.Errorf("修复历史会话归属失败: %w", err)
			}
		}
	}
	return nil
}

func (s Service) RestartPreference() (bool, bool, error) {
	value, err := config.Load(s.ConfigPath)
	if err != nil {
		return false, false, err
	}
	if value.RestartAfterSwitch == nil {
		return false, false, nil
	}
	return *value.RestartAfterSwitch, true, nil
}

func (s Service) SetRestartPreference(restart bool) error {
	value, err := s.load()
	if err != nil {
		return err
	}
	value.RestartAfterSwitch = &restart
	return config.Save(s.ConfigPath, value)
}

func (s Service) CurrentModel() (string, error) {
	return switcher.CurrentModel(s.CodexHome)
}

func (s Service) Test(name, model string) (string, error) {
	value, err := s.load()
	if err != nil {
		return "", err
	}
	profile, ok := profileByName(value, name)
	if !ok {
		return "", fmt.Errorf("profile %q 不存在", name)
	}
	authPath := config.ResolveAuthPath(profile.AuthFile, s.ConfigPath, s.Home)
	return switcher.Test(s.CodexHome, authPath, profile, model)
}

func (s Service) Add(profile config.Profile) error {
	value, err := s.load()
	if err != nil {
		return err
	}
	if err := value.SetProfile("", profile); err != nil {
		return err
	}
	apiKey := profile.APIKey
	profile, _ = profileByName(value, strings.TrimSpace(profile.Name))
	profile.APIKey = apiKey
	createdAuth, err := switcher.SeedAuth(profile, s.ConfigPath, s.CodexHome, s.Home)
	if err != nil {
		return err
	}
	if err := config.Save(s.ConfigPath, value); err != nil {
		if createdAuth {
			switcher.RemoveSeededAuth(profile, s.ConfigPath, s.Home)
		}
		return err
	}
	if err := switcher.EnsureProfileConfig(s.CodexHome, profile); err != nil {
		_ = value.DeleteProfile(profile.Name)
		if rollbackErr := config.Save(s.ConfigPath, value); rollbackErr != nil {
			return fmt.Errorf("%v；恢复 profile 配置也失败: %v", err, rollbackErr)
		}
		if createdAuth {
			switcher.RemoveSeededAuth(profile, s.ConfigPath, s.Home)
		}
		return err
	}
	return nil
}

func (s Service) Edit(oldName string, profile config.Profile) error {
	value, err := s.load()
	if err != nil {
		return err
	}
	oldProfile, ok := profileByName(value, oldName)
	if !ok {
		return fmt.Errorf("profile %q 不存在", oldName)
	}
	wasActive := value.ActiveProfile == oldName
	if err := value.SetProfile(oldName, profile); err != nil {
		return err
	}
	newName := profile.Name
	if wasActive {
		newName = value.ActiveProfile
	}
	apiKey := profile.APIKey
	profile, _ = profileByName(value, strings.TrimSpace(newName))
	profile.APIKey = apiKey
	createdAuth := false
	if wasActive {
		createdAuth, err = switcher.SeedAuth(profile, s.ConfigPath, s.CodexHome, s.Home)
		if err != nil {
			return err
		}
	}
	if err := config.Save(s.ConfigPath, value); err != nil {
		if createdAuth {
			switcher.RemoveSeededAuth(profile, s.ConfigPath, s.Home)
		}
		return err
	}
	if err := switcher.EnsureProfileConfig(s.CodexHome, profile); err != nil {
		_ = value.SetProfile(profile.Name, oldProfile)
		value.ActiveProfile = oldName
		if rollbackErr := config.Save(s.ConfigPath, value); rollbackErr != nil {
			return fmt.Errorf("%v；恢复 profile 配置也失败: %v", err, rollbackErr)
		}
		if createdAuth {
			switcher.RemoveSeededAuth(profile, s.ConfigPath, s.Home)
		}
		return err
	}
	if !wasActive {
		return nil
	}
	if err := switcher.Apply(&oldProfile, profile, s.ConfigPath, s.CodexHome, s.Home); err != nil {
		_ = value.SetProfile(profile.Name, oldProfile)
		value.ActiveProfile = oldName
		if rollbackErr := config.Save(s.ConfigPath, value); rollbackErr != nil {
			return fmt.Errorf("%v；恢复 profile 配置也失败: %v", err, rollbackErr)
		}
		if createdAuth {
			switcher.RemoveSeededAuth(profile, s.ConfigPath, s.Home)
		}
		return err
	}
	return nil
}

func (s Service) Delete(name string) error {
	value, err := s.load()
	if err != nil {
		return err
	}
	if err := value.DeleteProfile(name); err != nil {
		return err
	}
	return config.Save(s.ConfigPath, value)
}

func (s Service) load() (*config.Config, error) {
	value, err := config.Load(s.ConfigPath)
	if err != nil {
		return nil, err
	}
	if len(value.Presets) == 0 {
		return value, nil
	}
	// A profile is usable as soon as it is configured. If its snapshot was
	// removed, recreate it from the live auth state before inspecting or
	// switching profiles.
	for _, profile := range value.Profiles() {
		if _, err := switcher.SeedAuth(profile, s.ConfigPath, s.CodexHome, s.Home); err != nil {
			return nil, err
		}
	}
	if active, ok := profileByName(value, value.ActiveProfile); ok {
		provider, err := switcher.CurrentProvider(s.CodexHome)
		if err != nil {
			return nil, err
		}
		if provider == switcher.SessionProvider(active) || (active.AuthType == "official" && provider == active.Provider) {
			return value, nil
		}
	}
	active, err := switcher.DetectActive(value.Profiles(), s.ConfigPath, s.CodexHome, s.Home)
	if err != nil {
		return nil, err
	}
	if active == value.ActiveProfile {
		return value, nil
	}
	value.ActiveProfile = active
	if err := config.Save(s.ConfigPath, value); err != nil {
		return nil, err
	}
	return value, nil
}

func profileByName(value *config.Config, name string) (config.Profile, bool) {
	profile, ok := value.Presets[name]
	if ok {
		profile.Name = name
	}
	return profile, ok
}
