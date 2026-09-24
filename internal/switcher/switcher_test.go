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

func TestEnsureProfileDataWritesAndUpdatesProfile(t *testing.T) {
	profile := config.Profile{Name: "work", Provider: "third-party"}
	data := []byte("model_provider = \"openai\"\n")
	data = EnsureProfileData(data, profile)
	if !strings.Contains(string(data), `[profiles."work"]`) || !strings.Contains(string(data), `model_provider = "third-party"`) {
		t.Fatalf("profile section was not written: %s", data)
	}
	updated := EnsureProfileData(append(data, []byte("\n[profiles.\"other\"]\nmodel_provider = \"openai\"\n")...), config.Profile{Name: "work", Provider: "new-provider"})
	if strings.Count(string(updated), `[profiles."work"]`) != 1 || !strings.Contains(string(updated), `model_provider = "new-provider"`) {
		t.Fatalf("profile section was not updated: %s", updated)
	}
}
