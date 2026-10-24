package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const demoFlow = `name: demo
inputs:
  topic:
    default: notes
start: write
checks:
  has-file:
    run: 'test -f a.txt || { echo "a.txt is missing"; exit 1; }'
  approved:
    run: 'test -f approved.flag || { echo "waiting for approval"; exit 2; }'
steps:
  write:
    job: Create a.txt.
    checks: [has-file]
    then: review
  review:
    checks: [approved]
    timeout: 1h
    then: end
`

type sandbox struct {
	t    *testing.T
	repo string
	home string
	flow string
}

func newSandbox(t *testing.T) *sandbox {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "sub", "deeper"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.email=a@b.c", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	flow := filepath.Join(dir, "demo.yaml")
	if err := os.WriteFile(flow, []byte(demoFlow), 0o644); err != nil {
		t.Fatal(err)
	}
	return &sandbox{t: t, repo: repo, home: filepath.Join(dir, "home"), flow: flow}
}

func (s *sandbox) wf(cwd string, args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	env := Env{
		Stdout: &out,
		Stderr: &errOut,
		Cwd:    cwd,
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

func (s *sandbox) touch(name string) {
	s.t.Helper()
	if err := os.WriteFile(filepath.Join(s.repo, name), []byte("x"), 0o644); err != nil {
		s.t.Fatal(err)
	}
}

func TestRunLifecycleThroughCommands(t *testing.T) {
	s := newSandbox(t)

	code, out, errOut := s.wf(s.repo, "next")
	if code != ExitUsage || !strings.Contains(errOut, "there is no run") || !strings.Contains(errOut, "wf start") {
		t.Fatalf("no run: %d %q %q", code, out, errOut)
	}

	code, out, errOut = s.wf(s.repo, "start", s.flow)
	if code != 0 || !strings.Contains(out, "Step: write (visit 1 of 10)") || !strings.Contains(out, "a.txt is missing") || !strings.Contains(out, "Next: wf next") {
		t.Fatalf("start: %d %q %q", code, out, errOut)
	}

	code, _, errOut = s.wf(s.repo, "start", s.flow)
	if code != ExitUsage || !strings.Contains(errOut, "already has") {
		t.Fatalf("second start: %d %q", code, errOut)
	}

	sub := filepath.Join(s.repo, "sub", "deeper")
	code, out, _ = s.wf(sub, "next")
	if code != 0 || !strings.Contains(out, "Step: write") {
		t.Fatalf("subdirectory next: %d %q", code, out)
	}

	s.touch("a.txt")
	code, out, _ = s.wf(sub, "next")
	if code != 10 || !strings.Contains(out, "Step: review") || !strings.Contains(out, "wf wait --timeout") || !strings.Contains(out, "Bash tool timeout") {
		t.Fatalf("pending: %d %q", code, out)
	}

	start := time.Now()
	code, out, _ = s.wf(s.repo, "wait", "--timeout", "1s", "--interval", "1s")
	if code != 10 || time.Since(start) > 5*time.Second || !strings.Contains(out, "waiting for approval") {
		t.Fatalf("wait timeout: %d %q", code, out)
	}

	go func() {
		time.Sleep(300 * time.Millisecond)
		s.touch("approved.flag")
	}()
	code, out, _ = s.wf(s.repo, "wait", "--timeout", "10s", "--interval", "1s")
	if code != 13 || !strings.Contains(out, "State: ended") {
		t.Fatalf("wait resolve: %d %q", code, out)
	}

	code, out, _ = s.wf(s.repo, "next", "--json")
	if code != 13 {
		t.Fatalf("ended json code: %d", code)
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("%v %q", err, out)
	}
	for _, key := range []string{"run", "workflow", "step", "state", "job", "reason", "next_command"} {
		if _, ok := report[key]; !ok {
			t.Errorf("json lacks %q: %v", key, report)
		}
	}
	if report["state"] != "ended" {
		t.Fatalf("state %v", report["state"])
	}

	id, _ := report["run"].(string)
	code, out, _ = s.wf(os.TempDir(), "next", "--run", id)
	if code != 13 || !strings.Contains(out, "ended") {
		t.Fatalf("by run id: %d %q", code, out)
	}

	code, _, _ = s.wf(s.repo, "start", s.flow, "--input", "topic=other")
	if code != 0 {
		t.Fatalf("restart after end: %d", code)
	}
}

func TestStartRejectsBadInput(t *testing.T) {
	s := newSandbox(t)
	bad := filepath.Join(filepath.Dir(s.flow), "bad.yaml")
	if err := os.WriteFile(bad, []byte("name: bad\nstart: nowhere\nsteps: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := s.wf(s.repo, "start", bad)
	if code != ExitUsage || !strings.Contains(errOut, "no run was started") {
		t.Fatalf("invalid workflow: %d %q", code, errOut)
	}
	if entries, _ := os.ReadDir(filepath.Join(s.home, "runs")); len(entries) != 0 {
		t.Fatalf("run created: %v", entries)
	}
	for _, args := range [][]string{
		{"start"},
		{"start", s.flow, "--input", "novalue"},
		{"start", s.flow, "--input", "nope=1"},
		{"start", filepath.Join(filepath.Dir(s.flow), "absent.yaml")},
		{"next", "extra"},
		{"wait", "extra"},
		{"wait", "--timeout", "soon"},
		{"next", "--bogus"},
	} {
		if code, _, _ := s.wf(s.repo, args...); code != ExitUsage {
			t.Errorf("%v: code %d", args, code)
		}
	}
	outside := t.TempDir()
	if code, _, errOut := s.wf(outside, "start", s.flow); code != ExitUsage || !strings.Contains(errOut, "not inside a git worktree") {
		t.Fatalf("outside: %d %q", code, errOut)
	}
	if code, _, errOut := s.wf(outside, "next"); code != ExitUsage || !strings.Contains(errOut, "not inside a git worktree") {
		t.Fatalf("outside next: %d %q", code, errOut)
	}
	if code, _, _ := s.wf(s.repo, "next", "--run", "missing"); code != ExitUsage {
		t.Fatal("unknown run id accepted")
	}
}

func TestNoCommandSteersARun(t *testing.T) {
	forbidden := []string{"outcome", "step", "route", "approve", "retry", "cancel", "goto", "set", "skip"}
	for _, name := range Names() {
		for _, bad := range forbidden {
			if name == bad {
				t.Errorf("command %q can steer a run", name)
			}
		}
		for _, flagName := range Flags(name) {
			for _, bad := range forbidden {
				if flagName == bad {
					t.Errorf("command %q has steering flag %q", name, flagName)
				}
			}
		}
	}
	want := []string{"hook-session", "hook-stop", "hook-tool", "install", "next", "runs", "show", "start", "statusline", "validate", "wait"}
	if got := Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("commands %v, want %v; review each new command against the no-steering rule", got, want)
	}
	if got := Flags("nope"); got != nil {
		t.Fatalf("%v", got)
	}
}

func TestFlagTypes(t *testing.T) {
	var d duration
	if err := d.Set("2h"); err != nil || d.String() != "2h" {
		t.Fatalf("%v %q", err, d.String())
	}
	if err := d.Set("x"); err == nil {
		t.Fatal("bad duration accepted")
	}
	var m multi
	_ = m.Set("a=1")
	_ = m.Set("b=2")
	if m.String() != "a=1,b=2" {
		t.Fatal(m.String())
	}
	e := Env{}
	if _, err := e.cwd(); err != nil || e.context() == nil || e.now().IsZero() || e.env("PATH") == "" {
		t.Fatal("defaults")
	}
}
