package cli

import (
	"encoding/json"
	"io"
	"os"

	"github.com/JustinDFuller/workflow/internal/run"
)

type hookInput struct {
	Cwd       string `json:"cwd"`
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
		Command      string `json:"command"`
	} `json:"tool_input"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
}

func readHookInput(env Env) hookInput {
	var in hookInput
	if env.Stdin == nil {
		return in
	}
	if f, ok := env.Stdin.(*os.File); ok {
		if info, err := f.Stat(); err != nil || info.Mode()&os.ModeCharDevice != 0 {
			return in
		}
	}
	data, err := io.ReadAll(env.Stdin)
	if err != nil {
		return in
	}
	_ = json.Unmarshal(data, &in)
	return in
}

func (in hookInput) dir(env Env) string {
	switch {
	case in.Cwd != "":
		return in.Cwd
	case in.Workspace.CurrentDir != "":
		return in.Workspace.CurrentDir
	}
	cwd, _ := env.cwd()
	return cwd
}

type hookContext struct {
	store  *run.Store
	engine *run.Engine
	state  *run.RunState
	input  hookInput
}

func loadHookContext(env Env) (hookContext, bool) {
	in := readHookInput(env)
	store, err := env.store()
	if err != nil {
		return hookContext{}, false
	}
	root, err := worktreeRoot(in.dir(env))
	if err != nil {
		return hookContext{}, false
	}
	st, err := store.ForWorktree(root)
	if err != nil {
		return hookContext{}, false
	}
	return hookContext{store: store, engine: run.NewEngine(store), state: st, input: in}, true
}

func failOpen(env Env, body func(hookContext)) int {
	defer func() { _ = recover() }()
	if ctx, ok := loadHookContext(env); ok {
		body(ctx)
	}
	return 0
}

func (e Env) emitJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	e.printf("%s\n", data)
}

func hookStop(_ []string, _ *options, env Env) int {
	return failOpen(env, func(h hookContext) {
		decision := h.engine.DecideStop(h.state)
		if decision.Block {
			env.emitJSON(map[string]string{"decision": "block", "reason": decision.Reason})
		}
	})
}

func hookTool(_ []string, _ *options, env Env) int {
	return failOpen(env, func(h hookContext) {
		path := h.input.ToolInput.FilePath
		if path == "" {
			path = h.input.ToolInput.NotebookPath
		}
		call := run.ToolCall{Name: h.input.ToolName, Path: path, Command: h.input.ToolInput.Command, Cwd: h.input.dir(env)}
		reason, deny := run.CheckTool(h.state, h.store.Home, call)
		if !deny {
			return
		}
		env.emitJSON(map[string]any{"hookSpecificOutput": map[string]string{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": reason,
		}})
	})
}

func hookSession(_ []string, _ *options, env Env) int {
	return failOpen(env, func(h hookContext) {
		if text := run.SessionContext(h.state, env.now()); text != "" {
			env.printf("%s", text)
		}
	})
}

func statusline(_ []string, _ *options, env Env) int {
	return failOpen(env, func(h hookContext) {
		env.printf("%s\n", run.StatusLine(h.state))
	})
}
