package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readSettings(t *testing.T, repo string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, ".claude", "settings.local.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestInstallFirstSecondAndExisting(t *testing.T) {
	s := newSandbox(t)
	code, out, errOut := s.wf(s.repo, "install", "claude")
	if code != 0 {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
	for _, want := range []string{
		"Installed for Claude Code", "added the Stop hook (wf hook-stop)", "added the PreToolUse hook (wf hook-tool)",
		"added the SessionStart hook (wf hook-session)", "set the status line (wf statusline)", "allowed Bash(wf:*)", "added the /wf slash command",
		"hard for an agent to forge a run, not impossible", "records every check result",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("first install lacks %q:\n%s", want, out)
		}
	}
	settings := readSettings(t, s.repo)
	hooks := settings["hooks"].(map[string]any)
	for _, event := range []string{"Stop", "PreToolUse", "SessionStart"} {
		if len(hooks[event].([]any)) != 1 {
			t.Errorf("%s: %v", event, hooks[event])
		}
	}
	if settings["statusLine"].(map[string]any)["command"] != "wf statusline" {
		t.Errorf("%v", settings["statusLine"])
	}
	slash, err := os.ReadFile(filepath.Join(s.repo, ".claude", "commands", "wf.md"))
	if err != nil || !strings.Contains(string(slash), "wf start $ARGUMENTS") {
		t.Fatalf("%v %q", err, slash)
	}

	before, _ := os.ReadFile(filepath.Join(s.repo, ".claude", "settings.local.json"))
	code, out, _ = s.wf(s.repo, "install", "claude")
	after, _ := os.ReadFile(filepath.Join(s.repo, ".claude", "settings.local.json"))
	if code != 0 || !strings.Contains(out, "Everything is already installed") || strings.Contains(out, "added") || string(before) != string(after) {
		t.Fatalf("second install changed things: %d\n%s", code, out)
	}
	if !strings.Contains(out, "not impossible") {
		t.Fatalf("second install omits the limits:\n%s", out)
	}
}

func TestInstallKeepsUnrelatedSettings(t *testing.T) {
	s := newSandbox(t)
	dir := filepath.Join(s.repo, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{
  "model": "opus",
  "env": {"A": "1"},
  "permissions": {"allow": ["Bash(go test:*)"], "deny": ["Bash(rm:*)"]},
  "hooks": {"Stop": [{"hooks": [{"type": "command", "command": "other-tool stop"}]}], "PostToolUse": [{"hooks": [{"type": "command", "command": "fmt"}]}]},
  "statusLine": {"type": "command", "command": "my-status"}
}`
	if err := os.WriteFile(filepath.Join(dir, "settings.local.json"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "commands"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "commands", "wf.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, _ := s.wf(s.repo, "install", "claude")
	if code != 0 || !strings.Contains(out, "kept your existing status line (my-status)") || !strings.Contains(out, "kept your existing /wf slash command") {
		t.Fatalf("%d\n%s", code, out)
	}
	settings := readSettings(t, s.repo)
	if settings["model"] != "opus" || settings["env"].(map[string]any)["A"] != "1" {
		t.Errorf("unrelated keys lost: %v", settings)
	}
	perms := settings["permissions"].(map[string]any)
	if allow := perms["allow"].([]any); len(allow) != 2 || allow[0] != "Bash(go test:*)" || allow[1] != "Bash(wf:*)" {
		t.Errorf("%v", allow)
	}
	if deny := perms["deny"].([]any); len(deny) != 1 {
		t.Errorf("%v", deny)
	}
	hooks := settings["hooks"].(map[string]any)
	if stop := hooks["Stop"].([]any); len(stop) != 2 {
		t.Errorf("existing stop hook not kept: %v", stop)
	}
	if len(hooks["PostToolUse"].([]any)) != 1 {
		t.Errorf("unrelated hook lost")
	}
	if settings["statusLine"].(map[string]any)["command"] != "my-status" {
		t.Errorf("status line replaced: %v", settings["statusLine"])
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "commands", "wf.md")); string(got) != "mine" {
		t.Errorf("slash command replaced: %q", got)
	}

	code, out, _ = s.wf(s.repo, "install", "claude")
	if code != 0 || !strings.Contains(out, "Everything is already installed") {
		t.Fatalf("%d\n%s", code, out)
	}
}

func TestInstallRefusesSettingsOfTheWrongShape(t *testing.T) {
	for _, body := range []string{
		"null",
		`{"hooks":[]}`,
		`{"hooks":{"Stop":{}}}`,
		`{"statusLine":"x"}`,
		`{"permissions":[]}`,
		`{"permissions":{"allow":"Bash"}}`,
	} {
		s := newSandbox(t)
		dir := filepath.Join(s.repo, ".claude")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "settings.local.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		code, _, errOut := s.wf(s.repo, "install", "claude")
		if code != ExitUsage || !strings.Contains(errOut, "nothing was changed") {
			t.Errorf("%s: %d %q", body, code, errOut)
		}
		if got, _ := os.ReadFile(path); string(got) != body {
			t.Errorf("%s: the settings file was changed: %q", body, got)
		}
	}
}

func TestInstallErrors(t *testing.T) {
	s := newSandbox(t)
	for _, args := range [][]string{{"install"}, {"install", "codex"}, {"install", "claude", "extra"}} {
		if code, _, errOut := s.wf(s.repo, args...); code != ExitUsage || !strings.Contains(errOut, "usage: wf install claude") {
			t.Errorf("%v: %d %q", args, code, errOut)
		}
	}
	if code, _, _ := s.wf(os.TempDir(), "install", "claude"); code != ExitUsage {
		t.Error("install outside a worktree")
	}

	dir := filepath.Join(s.repo, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "settings.local.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := s.wf(s.repo, "install", "claude")
	if code != ExitUsage || !strings.Contains(errOut, "not valid JSON") {
		t.Fatalf("%d %q", code, errOut)
	}
	if got, _ := os.ReadFile(path); string(got) != "{not json" {
		t.Fatalf("an invalid settings file was changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "commands", "wf.md")); err == nil {
		t.Fatal("the slash command was written although the settings were invalid")
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := s.wf(s.repo, "install", "claude"); code != ExitUsage {
		t.Fatal("an unreadable settings path was accepted")
	}
}
