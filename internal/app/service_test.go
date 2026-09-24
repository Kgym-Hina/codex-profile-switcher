package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-profile-switcher/internal/config"
)

func TestAddAndEditAPIProfileCreatesMissingCodexConfig(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	writeFixture(t, filepath.Join(codexHome, "config.toml"), []byte("model = \"test\"\n"))
	writeFixture(t, filepath.Join(codexHome, "auth.json"), []byte(`{"OPENAI_API_KEY":"current"}`))
	configPath := filepath.Join(root, "presets.json")
	if err := config.Save(configPath, &config.Config{Presets: map[string]config.Profile{}}); err != nil {
		t.Fatal(err)
	}
	service := Service{ConfigPath: configPath, CodexHome: codexHome, Home: root}
	profile := config.Profile{Name: "api", Provider: "akarin", AuthType: "api", BaseURL: "https://first.example", AuthFile: "api.json", APIKey: "test-key"}
	if err := service.Add(profile); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`[profiles."api"]`, `[model_providers.akarin]`, `base_url = "https://first.example"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %q after Add: %s", want, data)
		}
	}
	profile.BaseURL = "https://changed.example"
	if err := service.Edit("api", profile); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if strings.Contains(string(data), "https://changed.example") || !strings.Contains(string(data), "https://first.example") {
		t.Fatalf("Edit rewrote existing provider: %s", data)
	}
	writeFixture(t, filepath.Join(codexHome, "config.toml"), []byte("model = \"test\"\n"))
	if err := service.Edit("api", profile); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if !strings.Contains(string(data), "https://changed.example") {
		t.Fatalf("Edit did not restore missing provider: %s", data)
	}
}

func TestSwitchMigratesLegacyOfficialProvider(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	configPath := filepath.Join(root, "presets.json")
	old := config.Profile{Provider: "boom", AuthType: "official", AuthFile: "old.json"}
	target := config.Profile{Provider: "openai", AuthType: "official", AuthFile: "new.json"}
	if err := config.Save(configPath, &config.Config{ActiveProfile: "old", Presets: map[string]config.Profile{"old": old, "new": target}}); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(root, "old.json"), []byte(`{"tokens":{"access_token":"old"}}`))
	writeFixture(t, filepath.Join(root, "new.json"), []byte(`{"tokens":{"access_token":"new"}}`))
	writeFixture(t, filepath.Join(codexHome, "auth.json"), []byte(`{"tokens":{"access_token":"old"}}`))
	writeFixture(t, filepath.Join(codexHome, "config.toml"), []byte("model_provider = \"boom\"\n[profiles.\"new\"]\nmodel_provider = \"boom\""))
	rollout := filepath.Join(codexHome, "sessions", "rollout.jsonl")
	writeFixture(t, rollout, []byte("{\"type\":\"session_meta\",\"payload\":{\"model_provider\":\"boom\"}}\n"))
	service := Service{ConfigPath: configPath, CodexHome: codexHome, Home: root}
	if err := service.Switch("new"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(rollout)
	if !strings.Contains(string(data), `"model_provider":"openai"`) {
		t.Fatalf("legacy rollout not migrated: %s", data)
	}
	data, _ = os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if strings.Contains(string(data), `model_provider = "boom"`) {
		t.Fatalf("legacy provider remains in config: %s", data)
	}
}

func TestAddOfficialProfileDoesNotCreateProviderConfig(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	configPath := filepath.Join(root, "presets.json")
	writeFixture(t, filepath.Join(codexHome, "config.toml"), []byte("model = \"test\"\n"))
	if err := config.Save(configPath, &config.Config{Presets: map[string]config.Profile{}}); err != nil {
		t.Fatal(err)
	}
	service := Service{ConfigPath: configPath, CodexHome: codexHome, Home: root}
	if err := service.Add(config.Profile{Name: "boom", Provider: "boom", AuthType: "official", AuthFile: "boom.json"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil || string(data) != "model = \"test\"\n" {
		t.Fatalf("official Add changed Codex config: %s, err = %v", data, err)
	}
}

func writeFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
