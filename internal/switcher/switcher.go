package switcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"codex-profile-switcher/internal/config"
)

type Status struct {
	config.Profile
	AuthPath  string
	Available bool
	Active    bool
	Reason    string
}

var providerLine = regexp.MustCompile(`^(\s*model_provider\s*=\s*)(?:"[^"]*"|'[^']*')(\s*(?:#.*)?)$`)
var providerValue = regexp.MustCompile(`^\s*model_provider\s*=\s*(?:"([^"]*)"|'([^']*)')\s*(?:#.*)?$`)
var modelLine = regexp.MustCompile(`^\s*model\s*=\s*(?:"([^"]*)"|'([^']*)')\s*(?:#.*)?$`)

func Inspect(profiles []config.Profile, activeProfile, configPath, codexHome, home string) ([]Status, error) {
	configData, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		return nil, fmt.Errorf("读取 Codex config.toml 失败: %w", err)
	}
	currentProvider := readProvider(configData)
	currentAuth, authErr := os.ReadFile(filepath.Join(codexHome, "auth.json"))

	statuses := make([]Status, 0, len(profiles))
	for _, profile := range profiles {
		authPath := config.ResolveAuthPath(profile.AuthFile, configPath, home)
		status := Status{Profile: profile, AuthPath: authPath, Active: profile.Name == activeProfile}
		profileAuth, err := os.ReadFile(authPath)
		if err != nil || !ValidAuth(profileAuth) {
			status.Reason = "认证文件不存在"
			if err == nil {
				status.Reason = "认证文件为空或无效"
			}
			if status.Active {
				if err == nil {
					status.Reason = "当前使用，备份文件为空或无效"
				} else {
					status.Reason = "当前使用，备份文件缺失"
				}
			}
			statuses = append(statuses, status)
			continue
		}
		status.Available = true
		switch {
		case status.Active && currentProvider != profile.Provider:
			status.Reason = "当前使用，provider 未同步"
		case status.Active && authErr == nil && !bytes.Equal(currentAuth, profileAuth):
			status.Reason = "当前使用，有待备份变更"
		case status.Active:
			status.Reason = "当前使用"
		default:
			status.Reason = "可切换"
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func DetectActive(profiles []config.Profile, configPath, codexHome, home string) (string, error) {
	configData, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		return "", fmt.Errorf("读取 Codex config.toml 失败: %w", err)
	}
	currentAuth, err := os.ReadFile(filepath.Join(codexHome, "auth.json"))
	if err != nil {
		return "", fmt.Errorf("读取 Codex auth.json 失败: %w", err)
	}
	provider := readProvider(configData)
	exact := make([]string, 0, 1)
	providerMatches := make([]string, 0, 1)
	for _, profile := range profiles {
		if profile.Provider != provider {
			continue
		}
		providerMatches = append(providerMatches, profile.Name)
		profileAuth, err := os.ReadFile(config.ResolveAuthPath(profile.AuthFile, configPath, home))
		if err == nil && bytes.Equal(currentAuth, profileAuth) {
			exact = append(exact, profile.Name)
		}
	}
	if len(exact) > 0 {
		return exact[0], nil
	}
	if len(providerMatches) == 1 {
		return providerMatches[0], nil
	}
	return "", nil
}

func CurrentProvider(codexHome string) (string, error) {
	data, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		return "", fmt.Errorf("读取 Codex config.toml 失败: %w", err)
	}
	return readProvider(data), nil
}

func Apply(current *config.Profile, target config.Profile, configPath, codexHome, home string) error {
	if info, err := os.Stat(codexHome); err != nil || !info.IsDir() {
		return fmt.Errorf("Codex 目录不存在: %s", codexHome)
	}
	configFile := filepath.Join(codexHome, "config.toml")
	authFile := filepath.Join(codexHome, "auth.json")
	originalConfig, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("读取 Codex config.toml 失败: %w", err)
	}
	originalAuth, err := os.ReadFile(authFile)
	if err != nil {
		return fmt.Errorf("读取当前认证文件失败: %w", err)
	}
	targetPath := config.ResolveAuthPath(target.AuthFile, configPath, home)
	targetAuth, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("缺少该 profile 对应的 auth 文件: %s", targetPath)
	}
	if !ValidAuth(targetAuth) {
		return fmt.Errorf("该 profile 的 auth 文件不是有效 JSON: %s", targetPath)
	}

	if current != nil {
		currentPath := config.ResolveAuthPath(current.AuthFile, configPath, home)
		if err := writeFileAtomic(currentPath, originalAuth, 0o600); err != nil {
			return fmt.Errorf("备份当前 profile 的认证文件失败: %w", err)
		}
		if samePath(currentPath, targetPath) {
			targetAuth = originalAuth
		}
	}

	updatedConfig := EnsureProviderData(originalConfig, target)
	if config.NormalizeAuthType(target.AuthType, target.Provider) == "official" {
		updatedConfig = removeProviderData(originalConfig)
	}
	if err := writeFileAtomic(configFile, updatedConfig, 0o600); err != nil {
		return fmt.Errorf("更新 config.toml 失败: %w", err)
	}
	if err := writeFileAtomic(authFile, targetAuth, 0o600); err != nil {
		if rollbackErr := writeFileAtomic(configFile, originalConfig, 0o600); rollbackErr != nil {
			return fmt.Errorf("替换认证文件失败: %v；回滚 config.toml 也失败: %v", err, rollbackErr)
		}
		return fmt.Errorf("替换认证文件失败: %w", err)
	}
	return nil
}

// RepairHistory updates the provider bucket used by Codex to list existing
// sessions. Only session metadata and the local thread index are changed;
// rollout conversation records are left untouched.
func RepairHistory(codexHome, fromProvider, toProvider string) error {
	if fromProvider == "" || toProvider == "" || fromProvider == toProvider {
		return nil
	}
	if err := repairRollouts(filepath.Join(codexHome, "sessions"), fromProvider, toProvider); err != nil {
		return err
	}
	if err := repairRollouts(filepath.Join(codexHome, "archived_sessions"), fromProvider, toProvider); err != nil {
		return err
	}
	return repairSQLiteIndexes(codexHome, fromProvider, toProvider)
}

// ValidAuth accepts JSON object snapshots with at least one field. An empty
// file or {} is not a usable Codex credential snapshot.
func ValidAuth(data []byte) bool {
	if len(bytes.TrimSpace(data)) == 0 || !json.Valid(data) {
		return false
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return false
	}
	if key, ok := object["OPENAI_API_KEY"]; ok {
		var value string
		if json.Unmarshal(key, &value) == nil && strings.TrimSpace(value) != "" {
			return true
		}
	}
	if tokens, ok := object["tokens"]; ok {
		var value map[string]json.RawMessage
		return json.Unmarshal(tokens, &value) == nil && len(value) > 0
	}
	return false
}

func CurrentModel(codexHome string) (string, error) {
	data, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		return "", fmt.Errorf("读取 Codex config.toml 失败: %w", err)
	}
	for start := 0; start <= len(data); {
		end := bytes.IndexByte(data[start:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += start
		}
		match := modelLine.FindSubmatch(data[start:end])
		if len(match) > 0 {
			if len(match[1]) > 0 {
				return string(match[1]), nil
			}
			return string(match[2]), nil
		}
		if end == len(data) {
			break
		}
		start = end + 1
	}
	return "", nil
}

// Test sends one harmless hello prompt through the installed Codex CLI using
// a temporary CODEX_HOME containing the selected profile.
func Test(codexHome, authPath string, profile config.Profile, model string) (string, error) {
	model = strings.TrimSpace(model)
	if model == "" || strings.ContainsAny(model, "\r\n") {
		return "", fmt.Errorf("模型不能为空且不能包含换行符")
	}
	configData, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		return "", fmt.Errorf("读取 Codex config.toml 失败: %w", err)
	}
	authData, err := os.ReadFile(authPath)
	if err != nil || !ValidAuth(authData) {
		return "", fmt.Errorf("认证文件为空或无效: %s", authPath)
	}
	testHome, err := os.MkdirTemp("", "codex-profile-test-")
	if err != nil {
		return "", fmt.Errorf("创建测试目录失败: %w", err)
	}
	defer os.RemoveAll(testHome)
	testConfig := EnsureProviderData(configData, profile)
	if config.NormalizeAuthType(profile.AuthType, profile.Provider) == "official" {
		testConfig = removeProviderData(configData)
	}
	if err := writeFileAtomic(filepath.Join(testHome, "config.toml"), testConfig, 0o600); err != nil {
		return "", fmt.Errorf("准备测试配置失败: %w", err)
	}
	if err := writeFileAtomic(filepath.Join(testHome, "auth.json"), authData, 0o600); err != nil {
		return "", fmt.Errorf("准备测试认证失败: %w", err)
	}
	codex, err := exec.LookPath("codex")
	if err != nil {
		codex, err = exec.LookPath("codex++")
	}
	if err != nil {
		return "", fmt.Errorf("找不到 codex 客户端")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, codex, "exec", "--ephemeral", "--skip-git-repo-check", "--color", "never", "--sandbox", "read-only", "--model", model, "hello")
	cmd.Dir = codexHome
	cmd.Env = append(os.Environ(), "CODEX_HOME="+testHome)
	output, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(output))
	if ctx.Err() != nil {
		return result, fmt.Errorf("模型测试超时")
	}
	if err != nil {
		if result != "" {
			return result, fmt.Errorf("模型测试失败: %w", err)
		}
		return "", fmt.Errorf("模型测试失败: %w", err)
	}
	return result, nil
}

func repairRollouts(root, fromProvider, toProvider string) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".jsonl" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := bytes.SplitAfter(data, []byte("\n"))
		if len(lines) == 0 {
			return nil
		}
		line := bytes.TrimSuffix(lines[0], []byte("\n"))
		var record map[string]interface{}
		if json.Unmarshal(line, &record) != nil || record["type"] != "session_meta" {
			return nil
		}
		payload, ok := record["payload"].(map[string]interface{})
		if !ok || payload["model_provider"] != fromProvider {
			return nil
		}
		payload["model_provider"] = toProvider
		updated, err := json.Marshal(record)
		if err != nil {
			return err
		}
		lines[0] = append(updated, '\n')
		if !bytes.HasSuffix(data, []byte("\n")) {
			lines[0] = bytes.TrimSuffix(lines[0], []byte("\n"))
		}
		mode := os.FileMode(0o600)
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
		return writeFileAtomic(path, bytes.Join(lines, nil), mode)
	})
}

func repairSQLiteIndexes(codexHome, fromProvider, toProvider string) error {
	sqlite, err := exec.LookPath("sqlite3")
	if err != nil {
		return nil
	}
	paths := []string{filepath.Join(codexHome, "state_5.sqlite"), filepath.Join(codexHome, "state.db")}
	if matches, globErr := filepath.Glob(filepath.Join(codexHome, "sqlite", "*.db")); globErr == nil {
		paths = append(paths, matches...)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		from := strings.ReplaceAll(fromProvider, "'", "''")
		to := strings.ReplaceAll(toProvider, "'", "''")
		query := fmt.Sprintf("UPDATE threads SET model_provider='%s' WHERE model_provider='%s'; UPDATE local_thread_catalog SET model_provider='%s' WHERE model_provider='%s';", to, from, to, from)
		cmd := exec.Command(sqlite, path, query)
		if output, err := cmd.CombinedOutput(); err != nil && len(output) > 0 && !bytes.Contains(output, []byte("no such table")) {
			return fmt.Errorf("修复 Codex 会话索引失败: %w", err)
		}
	}
	return nil
}

func SeedAuth(profile config.Profile, configPath, codexHome, home string) (bool, error) {
	path := config.ResolveAuthPath(profile.AuthFile, configPath, home)
	if data, err := os.ReadFile(path); err == nil {
		if ValidAuth(data) {
			return false, nil
		}
		if profile.APIKey == "" {
			return false, nil
		}
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("检查认证文件失败: %w", err)
	}
	data := []byte{}
	if profile.APIKey != "" {
		encoded, err := json.Marshal(map[string]string{"OPENAI_API_KEY": profile.APIKey})
		if err != nil {
			return false, fmt.Errorf("生成 API key 认证文件失败: %w", err)
		}
		data = encoded
	}
	if err := writeFileAtomic(path, data, 0o600); err != nil {
		return false, fmt.Errorf("创建 profile 认证文件失败: %w", err)
	}
	return true, nil
}

// EnsureProviderData updates only the active provider selector. Provider
// tables are created separately when a new API profile is added.
func EnsureProviderData(data []byte, profile config.Profile) []byte {
	return updateProviderData(data, profile.Provider)
}

func ensureProviderTableData(data []byte, profile config.Profile) []byte {
	if profile.BaseURL == "" || profile.Provider == "openai" || providerTableExists(data, profile.Provider) {
		return data
	}
	name := strconv.Quote(profile.Provider)
	baseURL := strconv.Quote(profile.BaseURL)
	block := fmt.Sprintf("\n[model_providers.%s]\nname = %s\nwire_api = \"responses\"\nrequires_openai_auth = true\nbase_url = %s\n", profile.Provider, name, baseURL)
	return append(data, []byte(block)...)
}

func EnsureProfileData(data []byte, profile config.Profile) []byte {
	section := fmt.Sprintf("[profiles.%s]", strconv.Quote(profile.Name))
	if bytes.Contains(data, []byte(section)) {
		return data
	}
	block := fmt.Sprintf("\n%s\nmodel_provider = %s\n", section, strconv.Quote(profile.Provider))
	return append(data, []byte(block)...)
}

func EnsureProfileConfig(codexHome string, profile config.Profile) error {
	if config.NormalizeAuthType(profile.AuthType, profile.Provider) == "official" {
		return nil
	}
	path := filepath.Join(codexHome, "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 Codex config.toml 失败: %w", err)
	}
	updated := EnsureProfileData(data, profile)
	if profile.BaseURL != "" {
		updated = ensureProviderTableData(updated, profile)
	}
	if bytes.Equal(data, updated) {
		return nil
	}
	return writeFileAtomic(path, updated, 0o600)
}

func removeProviderData(data []byte) []byte {
	start, end := topLevelProviderLine(data)
	if start < 0 {
		return data
	}
	if end < len(data) && data[end] == '\n' {
		end++
	}
	return append(append([]byte{}, data[:start]...), data[end:]...)
}

func providerTableExists(data []byte, provider string) bool {
	quoted := regexp.QuoteMeta(provider)
	pattern := regexp.MustCompile(`(?m)^\s*\[model_providers\.` + quoted + `\]\s*$`)
	return pattern.Match(data)
}

func RemoveSeededAuth(profile config.Profile, configPath, home string) {
	_ = os.Remove(config.ResolveAuthPath(profile.AuthFile, configPath, home))
}

func readProvider(data []byte) string {
	start, end := topLevelProviderLine(data)
	if start < 0 {
		return ""
	}
	match := providerValue.FindSubmatch(data[start:end])
	if len(match) == 0 {
		return ""
	}
	if len(match[1]) > 0 {
		return string(match[1])
	}
	return string(match[2])
}

func updateProviderData(data []byte, provider string) []byte {
	value := strconv.Quote(provider)
	lineStart, lineEnd := topLevelProviderLine(data)
	if lineStart < 0 {
		prefix := []byte("model_provider = " + value + "\n")
		return append(prefix, data...)
	}
	match := providerLine.FindSubmatchIndex(data[lineStart:lineEnd])
	if len(match) == 0 {
		prefix := []byte("model_provider = " + value + "\n")
		return append(prefix, data...)
	}
	for i := range match {
		match[i] += lineStart
	}
	var updated bytes.Buffer
	updated.Grow(len(data) + len(provider))
	updated.Write(data[:match[0]])
	updated.Write(data[match[2]:match[3]])
	updated.WriteString(value)
	updated.Write(data[match[4]:match[5]])
	updated.Write(data[match[1]:])
	return updated.Bytes()
}

func topLevelProviderLine(data []byte) (int, int) {
	for start := 0; start <= len(data); {
		end := bytes.IndexByte(data[start:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += start
		}
		line := data[start:end]
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			break
		}
		if providerLine.Match(line) {
			return start, end
		}
		if end == len(data) {
			break
		}
		start = end + 1
	}
	return -1, -1
}

func writeFileAtomic(path string, data []byte, fallbackMode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".codex-profile.tmp-*")
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

func DefaultAuthFile(name string) string {
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r > 127 {
			return r
		}
		return '-'
	}, strings.TrimSpace(name))
	return "~/.codex/auth." + safe + ".json"
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && leftAbs == rightAbs
}
