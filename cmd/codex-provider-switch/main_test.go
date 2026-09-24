package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestParseAppPIDsOnlyMatchesMainExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("macOS process paths use Unix separators")
	}
	appPath := "/Applications/ChatGPT.app"
	processes := []byte(`101 /Applications/ChatGPT.app/Contents/MacOS/ChatGPT
102 /Applications/ChatGPT.app/Contents/Frameworks/Codex Framework.framework/Helpers/Codex (Renderer)
103 /opt/homebrew/bin/codex
104 /Applications/Other.app/Contents/MacOS/Other
105 /Applications/ChatGPT.app/Contents/MacOS/ChatGPT
`)
	if got := parseAppPIDs(processes, appPath); !reflect.DeepEqual(got, []int{101, 105}) {
		t.Fatalf("app PIDs = %v, want [101 105]", got)
	}
}

func TestRestartCodexMacKillsBeforeLaunchingFreshInstance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	root := t.TempDir()
	appPath := filepath.Join(root, "ChatGPT.app")
	state := filepath.Join(root, "state")
	log := filepath.Join(root, "commands")
	if err := os.WriteFile(state, []byte("running"), 0o600); err != nil {
		t.Fatal(err)
	}
	scripts := map[string]string{
		"osascript": "printf '%s/\\n' \"$TEST_APP_PATH\"\n",
		"ps":        "if [ \"$(cat \"$TEST_STATE\")\" = running ]; then printf '101 %s/Contents/MacOS/ChatGPT\\n' \"$TEST_APP_PATH\"; fi\nprintf '202 /opt/homebrew/bin/codex\\n'\n",
		"kill":      "printf 'kill %s\\n' \"$*\" >> \"$TEST_LOG\"\nprintf dead > \"$TEST_STATE\"\n",
		"open":      "printf 'open %s\\n' \"$*\" >> \"$TEST_LOG\"\nprintf running > \"$TEST_STATE\"\n",
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(root, name), []byte("#!/bin/sh\n"+body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_APP_PATH", appPath)
	t.Setenv("TEST_STATE", state)
	t.Setenv("TEST_LOG", log)
	if err := restartCodexMac(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	want := "kill -9 101\nopen -n -a " + appPath + "\n"
	if string(data) != want {
		t.Fatalf("commands = %q, want %q", strings.TrimSpace(string(data)), strings.TrimSpace(want))
	}
}
