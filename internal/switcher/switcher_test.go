package switcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"codex-profile-switcher/internal/config"
)

func TestValidAuthRejectsEmptySnapshots(t *testing.T) {
	for _, data := range []string{"", "{}", `{"OPENAI_API_KEY":""}`, `{"auth_mode":"api"}`} {
		if ValidAuth([]byte(data)) {
			t.Errorf("ValidAuth(%q) = true, want false", data)
		}
	}
	if !ValidAuth([]byte(`{"OPENAI_API_KEY":"sk-test"}`)) {
		t.Fatal("API key auth should be valid")
	}
	if !ValidAuth([]byte(`{"tokens":{"access_token":"token"}}`)) {
		t.Fatal("official token auth should be valid")
	}
}

func TestOfficialProfileDetectionAndConfigCleanup(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "presets.json")
	profile := config.Profile{Name: "boom", Provider: "boom", AuthType: "official", AuthFile: "boom.json"}
	auth := []byte(`{"tokens":{"access_token":"test"}}`)
	writeTestFile(t, filepath.Join(root, "boom.json"), auth)
	writeTestFile(t, filepath.Join(codexHome, "auth.json"), auth)
	writeTestFile(t, filepath.Join(codexHome, "config.toml"), []byte("model_provider = \"boom\"\n[profiles.\"boom\"] # account\n# model_provider = \"keep comment\"\nmodel_provider = \"boom\" # stale\nmodel_provider_extra = \"keep\""))
	if err := Apply(nil, profile, configPath, codexHome, root); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if readProvider(got) != "" || strings.Contains(string(got), "model_provider = \"boom\" # stale") {
		t.Fatalf("official provider selector remains: %s", got)
	}
	if !strings.Contains(string(got), "# model_provider = \"keep comment\"") || !strings.Contains(string(got), "model_provider_extra = \"keep\"") {
		t.Fatalf("unrelated profile lines were removed: %s", got)
	}
	statuses, err := Inspect([]config.Profile{profile}, profile.Name, configPath, codexHome, root)
	if err != nil || len(statuses) != 1 || statuses[0].Reason != "当前使用" {
		t.Fatalf("official status = %+v, err = %v", statuses, err)
	}
	active, err := DetectActive([]config.Profile{profile}, configPath, codexHome, root)
	if err != nil || active != profile.Name {
		t.Fatalf("detected profile = %q, err = %v", active, err)
	}
}

func TestEnsureProfileConfigPreservesExistingProvider(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.toml")
	original := []byte("model = \"test\"\n[model_providers.akarin]\nbase_url = \"https://keep.example\"")
	writeTestFile(t, path, original)
	profile := config.Profile{Name: "api", Provider: "akarin", AuthType: "api", BaseURL: "https://new.example"}
	if err := EnsureProfileConfig(root, profile); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "https://keep.example") || strings.Contains(string(got), "https://new.example") || !strings.Contains(string(got), `[profiles."api"]`) {
		t.Fatalf("unexpected API config: %s", got)
	}
	newProfile := config.Profile{Name: "new", Provider: "new-provider", AuthType: "api", BaseURL: "https://new.example"}
	if err := EnsureProfileConfig(root, newProfile); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(path)
	for _, want := range []string{`[model_providers.new-provider]`, `wire_api = "responses"`, `base_url = "https://new.example"`} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("missing %q in config: %s", want, got)
		}
	}
}

func TestRepairHistoryUpdatesRolloutAndSingleSQLiteTable(t *testing.T) {
	sqlite, err := exec.LookPath("sqlite3")
	if err != nil {
		t.Skip("sqlite3 unavailable")
	}
	root := t.TempDir()
	rollout := filepath.Join(root, "sessions", "rollout.jsonl")
	writeTestFile(t, rollout, []byte("{\"type\":\"session_meta\",\"payload\":{\"model_provider\":\"boom\"}}\n{\"type\":\"message\"}\n"))
	db := filepath.Join(root, "state_5.sqlite")
	if output, err := exec.Command(sqlite, db, "CREATE TABLE threads (model_provider TEXT); INSERT INTO threads VALUES ('boom');").CombinedOutput(); err != nil {
		t.Fatalf("create SQLite fixture: %s: %v", output, err)
	}
	if err := RepairHistory(root, "boom", "openai"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rollout)
	if err != nil || !strings.Contains(string(data), `"model_provider":"openai"`) {
		t.Fatalf("rollout = %s, err = %v", data, err)
	}
	output, err := exec.Command(sqlite, db, "SELECT model_provider FROM threads;").CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "openai" {
		t.Fatalf("SQLite provider = %q, err = %v", output, err)
	}
}

func TestModelTestRejectsEmptyAuth(t *testing.T) {
	root := t.TempDir()
	authPath := filepath.Join(root, "empty.json")
	writeTestFile(t, filepath.Join(root, "config.toml"), []byte("model = \"test\"\n"))
	writeTestFile(t, authPath, []byte("{}"))
	if _, err := Test(root, authPath, config.Profile{Name: "empty", Provider: "openai", AuthType: "official"}, "test-model"); err == nil {
		t.Fatal("model test accepted empty auth")
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureProfileDataWritesOnce(t *testing.T) {
	profile := config.Profile{Name: "work", Provider: "third-party"}
	data := []byte("model_provider = \"openai\"\n")
	data = EnsureProfileData(data, profile)
	if !strings.Contains(string(data), `[profiles."work"]`) || !strings.Contains(string(data), `model_provider = "third-party"`) {
		t.Fatalf("profile section was not written: %s", data)
	}
	updated := EnsureProfileData(data, config.Profile{Name: "work", Provider: "new-provider"})
	if string(updated) != string(data) {
		t.Fatalf("existing profile section was rewritten: %s", updated)
	}
}

func TestProviderConfigRules(t *testing.T) {
	data := []byte("model_provider = \"old\"\n")
	api := config.Profile{Provider: "akarin", AuthType: "api", BaseURL: "https://i.zhs.moe:8443"}
	created := ensureProviderTableData(data, api)
	for _, want := range []string{
		`name = "akarin"`,
		`wire_api = "responses"`,
		`requires_openai_auth = true`,
		`base_url = "https://i.zhs.moe:8443"`,
	} {
		if !strings.Contains(string(created), want) {
			t.Errorf("new provider config missing %q: %s", want, created)
		}
	}
	existing := []byte("model_provider = \"old\"\n\n[model_providers.akarin]\nname = \"custom\"\nbase_url = \"https://keep.example\"\n")
	if string(ensureProviderTableData(existing, api)) != string(existing) {
		t.Fatal("existing provider configuration was rewritten")
	}
	if string(removeProviderData(data)) != "" {
		t.Fatal("official switch should remove the top-level model_provider")
	}
}

func TestSessionProviderUsesOpenAIForOfficialProfiles(t *testing.T) {
	if got := SessionProvider(config.Profile{Provider: "boom", AuthType: "official"}); got != "openai" {
		t.Fatalf("official session provider = %q, want openai", got)
	}
	if got := SessionProvider(config.Profile{Provider: "akarin", AuthType: "api"}); got != "akarin" {
		t.Fatalf("API session provider = %q, want akarin", got)
	}
}
