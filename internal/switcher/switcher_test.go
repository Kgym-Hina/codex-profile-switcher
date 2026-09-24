package switcher

import (
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
