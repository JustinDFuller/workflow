package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBoardCommands(t *testing.T) {
	s := newSandbox(t)

	code, out, _ := s.wf(s.repo)
	if code != 0 || !strings.Contains(out, "No runs to show") {
		t.Fatalf("bare empty: %d %q", code, out)
	}

	s.wf(s.repo, "start", s.flow)
	code, out, _ = s.wf(s.repo)
	if code != 0 || !strings.Contains(out, "demo") || !strings.Contains(out, "WORK") {
		t.Fatalf("bare: %d %q", code, out)
	}
	_, same, _ := s.wf(s.repo, "runs")
	if same != out {
		t.Fatalf("bare and runs differ:\n%s\n%s", out, same)
	}

	code, out, _ = s.wf(s.repo, "show")
	if code != 0 || !strings.Contains(out, "[>] write") || !strings.Contains(out, "[ ] review") || !strings.Contains(out, "Create a.txt.") || !strings.Contains(out, "a.txt is missing") {
		t.Fatalf("show: %d %q", code, out)
	}

	s.touch("a.txt")
	s.wf(s.repo, "next")
	_, out, _ = s.wf(s.repo, "runs", "--json")
	var list []map[string]any
	if err := json.Unmarshal([]byte(out), &list); err != nil || len(list) != 1 {
		t.Fatalf("%v %q", err, out)
	}
	for _, key := range []string{"run", "workflow", "worktree", "step", "state", "needs_you", "artifacts", "in_step_seconds"} {
		if _, ok := list[0][key]; !ok {
			t.Errorf("runs json lacks %q", key)
		}
	}
	if list[0]["state"] != "pending" || list[0]["step"] != "review" {
		t.Fatalf("%v", list[0])
	}
	id, _ := list[0]["run"].(string)

	code, out, _ = s.wf(s.repo, "show", "--json", "--run", id)
	var detail map[string]any
	if err := json.Unmarshal([]byte(out), &detail); err != nil || code != 0 {
		t.Fatalf("%v %q", err, out)
	}
	for _, key := range []string{"steps", "checks", "artifacts", "history", "state", "step"} {
		if _, ok := detail[key]; !ok {
			t.Errorf("show json lacks %q", key)
		}
	}

	s.touch("approved.flag")
	s.wf(s.repo, "next")
	_, out, _ = s.wf(s.repo, "runs")
	if strings.Contains(out, id) {
		t.Fatalf("ended run listed:\n%s", out)
	}
	_, out, _ = s.wf(s.repo, "runs", "--all")
	if !strings.Contains(out, id) || !strings.Contains(out, "ENDED") {
		t.Fatalf("--all:\n%s", out)
	}

	for _, args := range [][]string{{"runs", "extra"}, {"show", "extra"}, {"show", "--run", "missing"}, {"runs", "--interval", "x"}} {
		if code, _, _ := s.wf(s.repo, args...); code != ExitUsage {
			t.Errorf("%v: %d", args, code)
		}
	}
	if code, _, _ := s.wf(os.TempDir(), "show"); code != ExitUsage {
		t.Fatal("show outside a worktree")
	}
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestWatchRedrawsOnChange(t *testing.T) {
	for command, marks := range map[string][2]string{"runs": {"step       write", "step       review"}, "show": {"[>] write", "[>] review"}} {
		s := newSandbox(t)
		s.wf(s.repo, "start", s.flow)
		ctx, cancel := context.WithCancel(context.Background())
		var out, errOut syncBuffer
		env := Env{
			Stdout: &out, Stderr: &errOut, Cwd: s.repo, Ctx: ctx,
			Getenv: func(k string) string {
				if k == "WF_HOME" {
					return s.home
				}
				return os.Getenv(k)
			},
		}
		done := make(chan int)
		go func() { done <- Run([]string{command, "--watch", "--interval", "1s"}, env) }()

		waitFor(t, &out, marks[0])
		s.touch("a.txt")
		s.wf(s.repo, "next")
		waitFor(t, &out, marks[1])
		cancel()
		select {
		case code := <-done:
			if code != 0 {
				t.Fatalf("%s watch exit %d %s", command, code, errOut.String())
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s watch did not stop", command)
		}
		if strings.Count(out.String(), clearScreen) < 2 {
			t.Fatalf("%s did not redraw: %q", command, out.String())
		}
	}
}

func waitFor(t *testing.T, out *syncBuffer, want string) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		text := out.String()
		if i := strings.LastIndex(text, clearScreen); i >= 0 && strings.Contains(text[i:], want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("never saw %q in the latest redraw: %q", want, out.String())
}
