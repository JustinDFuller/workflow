package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	permission    = "Bash(wf:*)"
	slashFile     = "wf.md"
	settingsFile  = "settings.local.json"
	sessionMatch  = "startup|resume|clear|compact"
	slashTemplate = `---
description: Start a wf run in this worktree
argument-hint: <workflow-file> [--input name=value]
allowed-tools: Bash(wf:*)
---
Start a workflow run by running wf start $ARGUMENTS and do what it says.

After that, always follow the Next line that wf prints. Do not stop while the run has work or a wait left. Do not edit or read the wf state files.
`
)

type hookSpec struct {
	event   string
	matcher string
	command string
}

var hookSpecs = []hookSpec{
	{"Stop", "", "wf hook-stop"},
	{"PreToolUse", "*", "wf hook-tool"},
	{"SessionStart", sessionMatch, "wf hook-session"},
}

const statuslineCommand = "wf statusline"

const limits = `What the hooks do: they block a stop while the run has work left, deny the tools a step lists as denied, and deny edits and shell commands that touch the wf state directory.
What they do not do: they make it hard for an agent to forge a run, not impossible, because a shell command can hide a path. The log records every check result, so a person can audit what was checked and what it said.
`

func hasHook(entries []any, command string) bool {
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		hooks, _ := m["hooks"].([]any)
		for _, h := range hooks {
			if hm, ok := h.(map[string]any); ok && hm["command"] == command {
				return true
			}
		}
	}
	return false
}

func mergeSettings(settings map[string]any) (changes, kept []string) {
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	for _, spec := range hookSpecs {
		entries, _ := hooks[spec.event].([]any)
		if hasHook(entries, spec.command) {
			continue
		}
		entry := map[string]any{"hooks": []any{map[string]any{"type": "command", "command": spec.command}}}
		if spec.matcher != "" {
			entry["matcher"] = spec.matcher
		}
		hooks[spec.event] = append(entries, entry)
		changes = append(changes, fmt.Sprintf("added the %s hook (%s)", spec.event, spec.command))
	}
	if len(changes) > 0 {
		settings["hooks"] = hooks
	}

	switch current, _ := settings["statusLine"].(map[string]any); {
	case current == nil:
		settings["statusLine"] = map[string]any{"type": "command", "command": statuslineCommand}
		changes = append(changes, fmt.Sprintf("set the status line (%s)", statuslineCommand))
	case current["command"] != statuslineCommand:
		kept = append(kept, fmt.Sprintf("kept your existing status line (%v); run %s from it to show the run", current["command"], statuslineCommand))
	}

	perms, _ := settings["permissions"].(map[string]any)
	if perms == nil {
		perms = map[string]any{}
	}
	allow, _ := perms["allow"].([]any)
	found := false
	for _, a := range allow {
		if a == permission {
			found = true
		}
	}
	if !found {
		perms["allow"] = append(allow, permission)
		settings["permissions"] = perms
		changes = append(changes, fmt.Sprintf("allowed %s without a prompt", permission))
	}
	return changes, kept
}

func installClaude(args []string, _ *options, env Env) int {
	if len(args) != 1 || args[0] != "claude" {
		fmt.Fprintln(env.Stderr, "usage: wf install claude")
		return ExitUsage
	}
	cwd, err := env.cwd()
	if err != nil {
		return env.fail(err)
	}
	root, err := worktreeRoot(cwd)
	if err != nil {
		return env.fail(err)
	}
	dir := filepath.Join(root, ".claude")
	settingsPath := filepath.Join(dir, settingsFile)

	settings := map[string]any{}
	existing, err := os.ReadFile(settingsPath)
	switch {
	case err == nil:
		if err := json.Unmarshal(existing, &settings); err != nil {
			return env.fail(fmt.Errorf("%s is not valid JSON, so nothing was changed: %v", settingsPath, err))
		}
	case !errors.Is(err, os.ErrNotExist):
		return env.fail(err)
	}

	changes, kept := mergeSettings(settings)
	if len(changes) > 0 {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return env.fail(err)
		}
		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return env.fail(err)
		}
		if err := os.WriteFile(settingsPath, append(data, '\n'), 0o644); err != nil {
			return env.fail(err)
		}
	}

	slashPath := filepath.Join(dir, "commands", slashFile)
	switch current, err := os.ReadFile(slashPath); {
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(filepath.Dir(slashPath), 0o755); err != nil {
			return env.fail(err)
		}
		if err := os.WriteFile(slashPath, []byte(slashTemplate), 0o644); err != nil {
			return env.fail(err)
		}
		changes = append(changes, "added the /wf slash command (.claude/commands/wf.md)")
	case err != nil:
		return env.fail(err)
	case string(current) != slashTemplate:
		kept = append(kept, "kept your existing /wf slash command (.claude/commands/wf.md)")
	}

	if len(changes) == 0 {
		env.printf("Everything is already installed for Claude Code in %s.\n", root)
	} else {
		env.printf("Installed for Claude Code in %s:\n", root)
	}
	for _, c := range append(changes, kept...) {
		env.printf("  %s\n", c)
	}
	if _, err := exec.LookPath("wf"); err != nil {
		env.printf("Warning: wf is not on your PATH, so the hooks cannot run it. Put the wf binary on your PATH.\n")
	}
	env.printf("%s", limits)
	return 0
}
