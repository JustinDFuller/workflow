package run

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JustinDFuller/workflow/internal/check"
)

const denyFlow = `name: guarded
start: edit
checks:
  ready:
    run: "true"
  review:
    run: "true"
steps:
  edit:
    job: Make the change.
    checks: [ready]
    deny_tools: [WebFetch]
    max_visits: 3
    then: reviewing
  reviewing:
    checks: [review]
    timeout: 1h
    needs_you: true
    then: end
`

func pass() check.Result { return check.Result{Status: check.Pass} }

func (h *harness) load() *RunState {
	h.t.Helper()
	st, err := h.engine.Store.Load(h.id)
	if err != nil {
		h.t.Fatal(err)
	}
	return st
}

func TestStopHookBlocksWhileWorkRemains(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	d := h.engine.DecideStop(h.load())
	if !d.Block {
		t.Fatal("work remained but the stop was allowed")
	}
	for _, want := range []string{"Step: edit", "Make the change.", "Next: wf next", "not ready"} {
		if !strings.Contains(d.Reason, want) {
			t.Errorf("reason lacks %q:\n%s", want, d.Reason)
		}
	}
}

func TestStopHookWithPendingAndWaiter(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("review", pending("no review yet"))
	h.next()
	d := h.engine.DecideStop(h.load())
	if !d.Block || !strings.Contains(d.Reason, "run this in the background") || !strings.Contains(d.Reason, "wf wait --timeout") {
		t.Fatalf("no waiter should block with the wait command: %+v", d)
	}
	h.engine.beat(h.id, 30*time.Second)
	if d := h.engine.DecideStop(h.load()); d.Block {
		t.Fatalf("live waiter should allow the stop: %+v", d)
	}
	h.clock.advance(2 * time.Minute)
	if d := h.engine.DecideStop(h.load()); !d.Block {
		t.Fatal("a killed waiter should not count")
	}
}

func TestStopHookAllowsWhenOverOrBroken(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", broken("gh is not logged in"))
	h.next()
	if d := h.engine.DecideStop(h.load()); d.Block {
		t.Fatalf("broken should allow the stop: %+v", d)
	}

	h = newHarness(t, denyFlow)
	h.next()
	if d := h.engine.DecideStop(h.load()); d.Block {
		t.Fatalf("ended should allow the stop: %+v", d)
	}

	h = newHarness(t, denyFlow)
	h.fake.set("review", fail("never"))
	h.next()
	h.clock.advance(2 * time.Hour)
	h.next()
	st := h.load()
	if st.State() != StateStopped {
		t.Fatalf("state %s", st.State())
	}
	if d := h.engine.DecideStop(st); d.Block {
		t.Fatalf("stopped should allow the stop: %+v", d)
	}
}

func TestStopGuardReleasesAfterFiveBlocksAndRecords(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	for i := 1; i <= StopGuardLimit; i++ {
		if d := h.engine.DecideStop(h.load()); !d.Block {
			t.Fatalf("block %d was allowed", i)
		}
	}
	d := h.engine.DecideStop(h.load())
	if d.Block {
		t.Fatal("the sixth stop should be released by the guard")
	}
	released := h.events(EventGuardReleased)
	if len(released) != 1 || !strings.Contains(released[0].Reason, "5 times") || released[0].Step != "edit" {
		t.Fatalf("%+v", released)
	}
	if d := h.engine.DecideStop(h.load()); !d.Block {
		t.Fatal("after a release the count starts again")
	}
	text := Show(h.load(), h.clock.now()).Text()
	if !strings.Contains(text, "the stop hook let the agent stop") {
		t.Fatalf("history does not show the release:\n%s", text)
	}
}

func TestStopGuardResetsOnNewEvent(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	for i := 0; i < StopGuardLimit-1; i++ {
		h.engine.DecideStop(h.load())
	}
	h.fake.set("ready", fail("still not ready"))
	h.next()
	for i := 0; i < StopGuardLimit; i++ {
		if d := h.engine.DecideStop(h.load()); !d.Block {
			t.Fatalf("block %d after progress was allowed", i+1)
		}
	}
	if len(h.events(EventGuardReleased)) != 0 {
		t.Fatal("released although an event was recorded between the blocks")
	}
}

func TestStopGuardIgnoresUnreadableCounter(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	if err := os.WriteFile(h.engine.guardPath(h.id), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if d := h.engine.DecideStop(h.load()); !d.Block {
		t.Fatal("an unreadable counter should count as zero")
	}
}

func TestStopHookDoesNotWaitForTheRunLock(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	l, err := h.engine.Store.lockRun(context.Background(), h.id)
	if err != nil {
		t.Fatal(err)
	}
	defer l.unlock()
	done := make(chan StopDecision, 1)
	go func() { done <- h.engine.DecideStop(h.load()) }()
	select {
	case d := <-done:
		if !d.Block {
			t.Fatal("expected a block")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the stop hook waited for the run lock")
	}
}

func TestToolDeniedByStepNamesStep(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	st := h.load()
	reason, deny := CheckTool(st, h.engine.Store.Home, ToolCall{Name: "WebFetch"})
	if !deny || !strings.Contains(reason, "edit") || !strings.Contains(reason, "WebFetch") {
		t.Fatalf("%q %v", reason, deny)
	}
	if _, deny := CheckTool(st, h.engine.Store.Home, ToolCall{Name: "Read", Path: "/x"}); deny {
		t.Fatal("an allowed tool was denied")
	}
}

func TestToolDenyListEndsWithTheRun(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.next()
	if _, deny := CheckTool(h.load(), h.engine.Store.Home, ToolCall{Name: "WebFetch"}); deny {
		t.Fatal("a finished run still denies tools")
	}
}

func TestToolDeniesStateDirectoryEdits(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	st := h.load()
	home := h.engine.Store.Home
	for _, call := range []ToolCall{
		{Name: "Edit", Path: filepath.Join(home, "runs", h.id, "events.jsonl")},
		{Name: "Write", Path: filepath.Join(home, "cache", "x")},
		{Name: "MultiEdit", Path: filepath.Join(home, "new", "file")},
		{Name: "NotebookEdit", Path: filepath.Join(home, "runs", "n.ipynb")},
		{Name: "Edit", Path: "runs/" + h.id + "/events.jsonl", Cwd: home},
		{Name: "Edit", Path: filepath.Join(home, "sub", "..", "runs", "x")},
	} {
		reason, deny := CheckTool(st, home, call)
		if !deny || !strings.Contains(reason, "workflow state") {
			t.Errorf("%+v: %q %v", call, reason, deny)
		}
	}
	for _, call := range []ToolCall{
		{Name: "Edit", Path: filepath.Join(filepath.Dir(home), "elsewhere.txt")},
		{Name: "Edit", Path: home + "-other/file"},
		{Name: "Read", Path: filepath.Join(home, "runs", h.id, "events.jsonl")},
		{Name: "Edit"},
	} {
		if reason, deny := CheckTool(st, home, call); deny {
			t.Errorf("%+v was denied: %q", call, reason)
		}
	}
}

func TestToolDeniesStateDirectoryThroughSymlink(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	st := h.load()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(h.engine.Store.Home, link); err != nil {
		t.Fatal(err)
	}
	if _, deny := CheckTool(st, h.engine.Store.Home, ToolCall{Name: "Write", Path: filepath.Join(link, "runs", "x")}); !deny {
		t.Fatal("a symlink into the state directory was allowed")
	}
}

func TestToolDeniesShellCommandsNamingStateDirectory(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	st := h.load()
	home := h.engine.Store.Home
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{
		"cat " + home + "/runs/x/events.jsonl",
		"echo '{}' >> " + filepath.Join(home, "runs", h.id, "events.jsonl"),
		"rm -rf $WF_HOME",
		"rm -rf ${WF_HOME}/runs",
	} {
		if _, deny := CheckTool(st, home, ToolCall{Name: "Bash", Command: command}); !deny {
			t.Errorf("%q was allowed", command)
		}
	}
	for _, command := range []string{"rm -rf ~/.wf/runs", "rm -rf $HOME/.wf/runs", "rm -rf ${HOME}/.wf/runs", "rm -rf ~//.wf/runs", "rm -rf $HOME//.wf//runs"} {
		if _, deny := CheckTool(st, filepath.Join(userHome, ".wf"), ToolCall{Name: "Bash", Command: command}); !deny {
			t.Errorf("%q was allowed", command)
		}
	}
	for _, command := range []string{"go test ./...", "wf next", "git status", ""} {
		if reason, deny := CheckTool(st, home, ToolCall{Name: "Bash", Command: command}); deny {
			t.Errorf("%q was denied: %q", command, reason)
		}
	}
	if _, deny := CheckTool(st, home, ToolCall{Name: "Read", Command: "cat " + home}); deny {
		t.Error("only shell commands are matched by text")
	}
}

func TestStatusLine(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	if got := StatusLine(h.load()); got != "guarded > edit (1/3) . work" {
		t.Fatalf("%q", got)
	}
	h.fake.set("ready", pass())
	h.fake.set("review", pending("none"))
	h.next()
	if got := StatusLine(h.load()); got != "guarded > reviewing (1/10) . pending . NEEDS YOU" {
		t.Fatalf("%q", got)
	}
	h.fake.set("review", pass())
	h.next()
	if got := StatusLine(h.load()); got != "guarded . ended" {
		t.Fatalf("%q", got)
	}

	h = newHarness(t, denyFlow)
	h.fake.set("review", fail("never"))
	h.next()
	h.clock.advance(2 * time.Hour)
	h.next()
	if got := StatusLine(h.load()); got != "guarded > reviewing . stopped . NEEDS YOU" {
		t.Fatalf("%q", got)
	}

	h = newHarness(t, denyFlow)
	h.fake.set("ready", broken("gh missing"))
	h.next()
	if got := StatusLine(h.load()); got != "guarded > edit (1/3) . broken . NEEDS YOU" {
		t.Fatalf("%q", got)
	}
}

func TestSessionContext(t *testing.T) {
	h := newHarness(t, denyFlow)
	h.fake.set("ready", fail("not ready"))
	h.next()
	text := SessionContext(h.load(), h.clock.now())
	for _, want := range []string{"Step: edit", "Make the change.", "Next: wf next"} {
		if !strings.Contains(text, want) {
			t.Errorf("session context lacks %q:\n%s", want, text)
		}
	}
	h.fake.set("ready", pass())
	h.fake.set("review", pass())
	h.next()
	if got := SessionContext(h.load(), h.clock.now()); got != "" {
		t.Fatalf("an ended run added context: %q", got)
	}
}
