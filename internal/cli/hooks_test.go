package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (s *sandbox) wfIn(cwd, stdin string, args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	env := Env{
		Stdout: &out, Stderr: &errOut, Stdin: strings.NewReader(stdin), Cwd: cwd,
		Getenv: func(k string) string {
			if k == "WF_HOME" {
				return s.home
			}
			return os.Getenv(k)
		},
		Ctx: context.Background(),
	}
	code := Run(args, env)
	return code, out.String(), errOut.String()
}

func hookJSON(fields map[string]any) string {
	data, _ := json.Marshal(fields)
	return string(data)
}

func TestHooksAllowAndPrintNothingWithoutARun(t *testing.T) {
	s := newSandbox(t)
	tool := hookJSON(map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": "cat " + s.home + "/runs/x"}})
	for _, args := range [][]string{{"hook-stop"}, {"hook-tool"}, {"hook-session"}, {"statusline"}} {
		code, out, errOut := s.wfIn(s.repo, tool, args...)
		if code != 0 || out != "" || errOut != "" {
			t.Errorf("%v: %d %q %q", args, code, out, errOut)
		}
	}
	for _, cwd := range []string{os.TempDir(), "/nonexistent-dir-for-wf"} {
		code, out, _ := s.wfIn(cwd, "not json at all", "hook-stop")
		if code != 0 || out != "" {
			t.Errorf("outside a worktree %q: %d %q", cwd, code, out)
		}
	}
}

func TestHooksFailOpenOnUnreadableState(t *testing.T) {
	s := newSandbox(t)
	s.wf(s.repo, "start", s.flow)
	logs, _ := filepath.Glob(filepath.Join(s.home, "runs", "*", "events.jsonl"))
	if len(logs) != 1 {
		t.Fatalf("%v", logs)
	}
	if err := os.WriteFile(logs[0], []byte("not json\n{\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"hook-stop"}, {"hook-tool"}, {"hook-session"}, {"statusline"}} {
		code, out, _ := s.wfIn(s.repo, hookJSON(map[string]any{"tool_name": "Bash"}), args...)
		if code != 0 || out != "" {
			t.Errorf("%v: %d %q", args, code, out)
		}
	}
}

func TestStopHookThroughTheCommand(t *testing.T) {
	s := newSandbox(t)
	s.wf(s.repo, "start", s.flow)
	input := hookJSON(map[string]any{"cwd": s.repo, "hook_event_name": "Stop", "stop_hook_active": false})
	code, out, _ := s.wfIn(os.TempDir(), input, "hook-stop")
	if code != 0 {
		t.Fatal(code)
	}
	var decision map[string]string
	if err := json.Unmarshal([]byte(out), &decision); err != nil || decision["decision"] != "block" {
		t.Fatalf("%v %q", err, out)
	}
	for _, want := range []string{"Step: write", "Create a.txt.", "Next: wf next"} {
		if !strings.Contains(decision["reason"], want) {
			t.Errorf("reason lacks %q: %s", want, decision["reason"])
		}
	}

	s.touch("a.txt")
	s.wf(s.repo, "next")
	_, out, _ = s.wfIn(s.repo, input, "hook-stop")
	if !strings.Contains(out, "wf wait --timeout") || !strings.Contains(out, "block") {
		t.Fatalf("pending without a waiter should block: %q", out)
	}

	s.touch("approved.flag")
	s.wf(s.repo, "next")
	if _, out, _ := s.wfIn(s.repo, input, "hook-stop"); out != "" {
		t.Fatalf("ended run blocked the stop: %q", out)
	}
}

func TestStopHookGuardThroughTheCommand(t *testing.T) {
	s := newSandbox(t)
	s.wf(s.repo, "start", s.flow)
	input := hookJSON(map[string]any{"cwd": s.repo})
	for i := 0; i < 5; i++ {
		if _, out, _ := s.wfIn(s.repo, input, "hook-stop"); !strings.Contains(out, "block") {
			t.Fatalf("stop %d allowed", i+1)
		}
	}
	if _, out, _ := s.wfIn(s.repo, input, "hook-stop"); out != "" {
		t.Fatalf("the guard did not release: %q", out)
	}
	_, out, _ := s.wf(s.repo, "show")
	if !strings.Contains(out, "the stop hook let the agent stop") {
		t.Fatalf("the board does not show the release:\n%s", out)
	}
}

func TestToolHookThroughTheCommand(t *testing.T) {
	s := newSandbox(t)
	s.wf(s.repo, "start", s.flow)
	cases := []struct {
		name  string
		input map[string]any
		deny  bool
	}{
		{"edit in state", map[string]any{"tool_name": "Edit", "tool_input": map[string]any{"file_path": filepath.Join(s.home, "runs", "x", "events.jsonl")}}, true},
		{"notebook in state", map[string]any{"tool_name": "NotebookEdit", "tool_input": map[string]any{"notebook_path": filepath.Join(s.home, "n.ipynb")}}, true},
		{"shell naming state", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": "echo hi >> " + s.home + "/runs/x/events.jsonl"}}, true},
		{"edit in repo", map[string]any{"tool_name": "Edit", "tool_input": map[string]any{"file_path": filepath.Join(s.repo, "a.txt")}}, false},
		{"plain shell", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": "go test ./..."}}, false},
	}
	for _, c := range cases {
		c.input["cwd"] = s.repo
		code, out, _ := s.wfIn(s.repo, hookJSON(c.input), "hook-tool")
		if code != 0 {
			t.Errorf("%s: exit %d", c.name, code)
		}
		if c.deny {
			var got struct {
				Out struct {
					Event    string `json:"hookEventName"`
					Decision string `json:"permissionDecision"`
					Reason   string `json:"permissionDecisionReason"`
				} `json:"hookSpecificOutput"`
			}
			if err := json.Unmarshal([]byte(out), &got); err != nil || got.Out.Decision != "deny" || got.Out.Event != "PreToolUse" || got.Out.Reason == "" {
				t.Errorf("%s: %v %q", c.name, err, out)
			}
		} else if out != "" {
			t.Errorf("%s: %q", c.name, out)
		}
	}
}

func TestToolHookDeniesStepDeniedTool(t *testing.T) {
	s := newSandbox(t)
	flow := strings.Replace(demoFlow, "    checks: [has-file]\n", "    checks: [has-file]\n    deny_tools: [Edit]\n", 1)
	if err := os.WriteFile(s.flow, []byte(flow), 0o644); err != nil {
		t.Fatal(err)
	}
	s.wf(s.repo, "start", s.flow)
	input := hookJSON(map[string]any{"cwd": s.repo, "tool_name": "Edit", "tool_input": map[string]any{"file_path": filepath.Join(s.repo, "a.txt")}})
	_, out, _ := s.wfIn(s.repo, input, "hook-tool")
	if !strings.Contains(out, "deny") || !strings.Contains(out, "write step") {
		t.Fatalf("%q", out)
	}
}

func TestSessionHookAndStatusLineThroughTheCommands(t *testing.T) {
	s := newSandbox(t)
	s.wf(s.repo, "start", s.flow)
	input := hookJSON(map[string]any{"workspace": map[string]any{"current_dir": s.repo}})
	code, out, _ := s.wfIn(os.TempDir(), input, "hook-session")
	if code != 0 || !strings.Contains(out, "Step: write") || !strings.Contains(out, "Create a.txt.") || !strings.Contains(out, "Next: wf next") {
		t.Fatalf("%d %q", code, out)
	}
	code, out, _ = s.wfIn(os.TempDir(), input, "statusline")
	if code != 0 || out != "demo > write (1/10) . work\n" {
		t.Fatalf("%d %q", code, out)
	}
	s.touch("a.txt")
	s.wf(s.repo, "next")
	_, out, _ = s.wfIn(s.repo, "", "statusline")
	if out != "demo > review (1/10) . pending\n" {
		t.Fatalf("%q", out)
	}
	s.touch("approved.flag")
	s.wf(s.repo, "next")
	if _, out, _ := s.wfIn(s.repo, input, "hook-session"); out != "" {
		t.Fatalf("session context for an ended run: %q", out)
	}
}

func TestHookReadsNothingFromATerminal(t *testing.T) {
	dev, err := os.Open("/dev/null")
	if err != nil {
		t.Skip(err)
	}
	defer dev.Close()
	var out bytes.Buffer
	env := Env{Stdout: &out, Stderr: &out, Stdin: dev, Cwd: t.TempDir()}
	if got := readHookInput(env); got != (hookInput{}) {
		t.Fatalf("%+v", got)
	}
	if got := readHookInput(Env{}); got != (hookInput{}) {
		t.Fatalf("%+v", got)
	}
}
