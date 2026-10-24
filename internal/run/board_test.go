package run

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/JustinDFuller/workflow/internal/check"
	"github.com/JustinDFuller/workflow/internal/workflow"
)

func (h *harness) state() *RunState {
	h.t.Helper()
	st, err := h.engine.Store.Load(h.id)
	if err != nil {
		h.t.Fatal(err)
	}
	return st
}

func startMore(t *testing.T, h *harness, worktree string) string {
	t.Helper()
	w, err := workflow.Load([]byte(flowYAML), "flow.yaml", "/wfdir")
	if err != nil {
		t.Fatal(err)
	}
	h.clock.advance(time.Second)
	id, err := h.engine.Start(w, []byte(flowYAML), worktree, nil)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestRunsBoardListsSeveralAndHidesEnded(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("two", fail("3 tasks remain"))
	h.next()
	second := startMore(t, h, "/work/second")
	third := startMore(t, h, "/work/third")
	h.fake.set("two", check.Result{Status: check.Pass})
	h.fake.set("review", check.Result{Status: check.Pass, Outcome: "approved"})
	if _, err := h.engine.Evaluate(t.Context(), third); err != nil {
		t.Fatal(err)
	}
	states, err := h.engine.Store.All()
	if err != nil || len(states) != 3 {
		t.Fatalf("%v %d", err, len(states))
	}
	visible := SummarizeAll(states, false, h.clock.now())
	if len(visible) != 2 {
		t.Fatalf("ended run listed: %+v", visible)
	}
	all := SummarizeAll(states, true, h.clock.now())
	if len(all) != 3 {
		t.Fatalf("%+v", all)
	}
	text := RunsText(all)
	for _, want := range []string{h.id, second, third, "/work/tree", "/work/second", "ENDED", "WORK"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
	if RunsText(nil) == "" || !strings.Contains(RunsText(nil), "wf start") {
		t.Fatal("empty board gives no hint")
	}
}

func TestNeedsYouAndWaitingOn(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("no review submitted since 10:14"))
	h.next()
	h.clock.advance(5 * time.Minute)
	s := Summarize(h.state(), h.clock.now())
	if !s.NeedsYou || s.State != StatePending || s.Detail != "no review submitted since 10:14" {
		t.Fatalf("%+v", s)
	}
	text := RunsText([]RunSummary{s})
	if !strings.Contains(text, "NEEDS YOU") || !strings.Contains(text, "waiting on no review submitted since 10:14") || !strings.Contains(text, "5m") {
		t.Fatal(text)
	}

	ci := newHarness(t, flowYAML)
	ci.fake.set("two", pending("2 checks running"))
	ci.next()
	s = Summarize(ci.state(), ci.clock.now())
	if s.NeedsYou || s.State != StatePending || s.Detail != "2 checks running" {
		t.Fatalf("%+v", s)
	}
	if text := RunsText([]RunSummary{s}); strings.Contains(text, "NEEDS YOU") || !strings.Contains(text, "does not need you") {
		t.Fatal(text)
	}

	st := newHarness(t, flowYAML)
	st.fake.set("gate", fail("proposal not approved"))
	st.next()
	s = Summarize(st.state(), st.clock.now())
	if !s.NeedsYou || s.State != StateStopped || !strings.Contains(RunsText([]RunSummary{s}), "stopped    proposal not approved") {
		t.Fatalf("%+v", s)
	}

	br := newHarness(t, flowYAML)
	br.fake.set("one", broken("gh: not logged in"))
	br.next()
	if s := Summarize(br.state(), br.clock.now()); !s.NeedsYou || !strings.Contains(RunsText([]RunSummary{s}), "broken     gh: not logged in") {
		t.Fatalf("%+v", s)
	}

	wk := newHarness(t, flowYAML)
	wk.fake.set("one", fail("working"))
	wk.next()
	if s := Summarize(wk.state(), wk.clock.now()); s.NeedsYou || !strings.Contains(RunsText([]RunSummary{s}), "working on working") {
		t.Fatalf("%+v", s)
	}
}

func TestRunsBoardShowsOnlyTheFirstLineOfAFindingList(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("one", fail("the verify check failed at abc123:\n- a.go: first finding\n- b.go: second finding"))
	h.next()
	s := Summarize(h.state(), h.clock.now())
	if !strings.Contains(s.Detail, "second finding") {
		t.Fatalf("the full message is lost from the summary: %q", s.Detail)
	}
	text := RunsText([]RunSummary{s})
	if !strings.Contains(text, "working on the verify check failed at abc123: ...\n") || strings.Contains(text, "first finding") {
		t.Fatalf("the board should show one line:\n%s", text)
	}
}

func TestShowPositionDoneAndLeft(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("no review yet"))
	h.next()
	h.clock.advance(90 * time.Minute)
	d := Show(h.state(), h.clock.now())
	got := map[string]string{}
	for _, s := range d.Steps {
		got[s.Name] = s.Status
	}
	want := map[string]string{"draft": "passed", "gated": "passed", "reviewing": "current", "revise": "not reached"}
	for name, status := range want {
		if got[name] != status {
			t.Errorf("%s is %q, want %q", name, got[name], status)
		}
	}
	if d.Steps[0].Name != "draft" || d.Steps[3].Name != "revise" {
		t.Fatalf("not in definition order: %+v", d.Steps)
	}
	text := d.Text()
	for _, want := range []string{"[x] draft", "[x] gated", "[>] reviewing", "[ ] revise", "passed 1h30m ago", "not reached", "current", "NEEDS YOU", "PENDING"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
}

func TestShowLoopCounts(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.queue("review",
		check.Result{Status: check.Pass, Outcome: "changes_requested"},
		check.Result{Status: check.Pass, Outcome: "changes_requested"},
		pending("again"),
	)
	h.next()
	d := Show(h.state(), h.clock.now())
	for _, s := range d.Steps {
		if s.Name == "reviewing" && (s.Visits != 3 || s.MaxVisits != 10) {
			t.Fatalf("%+v", s)
		}
	}
	if !strings.Contains(d.Text(), "3 of 10") {
		t.Fatal(d.Text())
	}
}

const sixChecks = `name: six
start: only
checks:
  c1:
    run: "true"
  c2:
    run: "true"
  c3:
    run: "true"
  c4:
    run: "true"
  c5:
    run: "true"
  c6:
    run: "true"
steps:
  only:
    job: Do the thing.
    checks: [c1, c2, c3, c4, c5, c6]
    timeout: 2d
    then: waiting
  waiting:
    checks: [c1]
    timeout: 1h
    then: end
`

func TestShowCurrentStepDetail(t *testing.T) {
	h := newHarness(t, sixChecks)
	h.fake.set("c4", pending("CI is running"))
	h.next()
	h.clock.advance(3 * time.Hour)
	d := Show(h.state(), h.clock.now())
	var statuses []string
	for _, c := range d.Checks {
		statuses = append(statuses, c.Status)
	}
	if strings.Join(statuses, ",") != "pass,pass,pass,pending,not run yet,not run yet" {
		t.Fatalf("%v", statuses)
	}
	if d.Checks[3].Message != "CI is running" || d.Job != "Do the thing." || d.DeadlineSeconds == nil {
		t.Fatalf("%+v", d)
	}
	text := d.Text()
	for _, want := range []string{"Job:\n  Do the thing.", "Deadline: 1d21h left", "pending      c4: CI is running", "not run yet  c5"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
	h.clock.advance(3 * 24 * time.Hour)
	if text := Show(h.state(), h.clock.now()).Text(); !strings.Contains(text, "Deadline: passed") {
		t.Fatal(text)
	}
}

func TestShowStepWithoutJobAndMultilineMessage(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("first line\nsecond line"))
	h.next()
	text := Show(h.state(), h.clock.now()).Text()
	if !strings.Contains(text, "Job: none. This step waits for its checks.") || !strings.Contains(text, "first line ...") || strings.Contains(text, "second line") {
		t.Fatal(text)
	}
}

func TestShowArtifacts(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("two", fail("later"))
	if text := Show(h.state(), h.clock.now()).Text(); !strings.Contains(text, "none yet") {
		t.Fatal(text)
	}
	h.fake.set("one", check.Result{Status: check.Pass, Artifacts: []check.Artifact{
		{Name: "change", Address: "openspec/changes/add-x"},
		{Name: "proposal_pr", Address: "https://example.test/pull/1"},
	}})
	h.next()
	d := Show(h.state(), h.clock.now())
	if len(d.Artifacts) != 2 {
		t.Fatalf("%+v", d.Artifacts)
	}
	text := d.Text()
	if !strings.Contains(text, "change         openspec/changes/add-x") || !strings.Contains(text, "proposal_pr    https://example.test/pull/1") {
		t.Fatal(text)
	}
}

func TestShowHistoryIsPlainAndNewestFirst(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", check.Result{Status: check.Pass, Outcome: "approved"})
	h.fake.set("two", pending("2 checks running"))
	h.next()
	h.clock.advance(time.Minute)
	h.fake.set("two", check.Result{Status: check.Pass})
	h.fake.set("three", fail("not pushed"))
	h.next()
	h.clock.advance(time.Minute)
	h.fake.set("three", check.Result{Status: check.Pass})
	h.next()
	d := Show(h.state(), h.clock.now())
	if len(d.History) == 0 || d.History[0].Text != "run ended" {
		t.Fatalf("%+v", d.History)
	}
	joined := ""
	for _, line := range d.History {
		joined += line.Text + "\n"
	}
	for _, want := range []string{"draft passed, moved to gated", "reviewing passed, the workflow is finished", "two is pending: 2 checks running", "three failed: not pushed", "review reported approved", "run ended"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in\n%s", want, joined)
		}
	}
	for _, raw := range []string{"step_passed", "check_result", "step_entered"} {
		if strings.Contains(joined, raw) {
			t.Errorf("history exposes internal name %q", raw)
		}
	}
	for i := 1; i < len(d.History); i++ {
		if d.History[i].Time.After(d.History[i-1].Time) {
			t.Fatalf("not newest first: %+v", d.History)
		}
	}
	if len(d.History) > HistoryLimit {
		t.Fatal("history not limited")
	}
	text := d.Text()
	if !strings.Contains(text, "The workflow is finished.") || strings.Contains(text, "Now:") {
		t.Fatal(text)
	}
}

func TestShowStoppedAndOtherSentences(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("gate", fail("proposal not approved"))
	h.next()
	text := Show(h.state(), h.clock.now()).Text()
	if !strings.Contains(text, "Stopped at gated: proposal not approved") || !strings.Contains(text, "run stopped: proposal not approved") {
		t.Fatal(text)
	}
	for _, e := range []Event{
		{Type: EventStepPassed, Step: "a", Target: workflow.TargetStop},
		{Type: EventCheckResult, Check: "c", Status: check.Broken, Message: "boom"},
		{Type: EventCheckResult, Check: "c", Status: check.Pass},
		{Type: EventArtifact, Name: "n", Address: "p"},
	} {
		if s, ok := sentence(e); !ok || s == "" {
			t.Errorf("no sentence for %+v", e)
		}
	}
	if _, ok := sentence(Event{Type: EventStepEntered}); ok {
		t.Error("step_entered should not produce a line")
	}
}

func TestBoardTakesNoLock(t *testing.T) {
	h := newHarness(t, flowYAML)
	l, err := h.engine.Store.lockRun(context.Background(), h.id)
	if err != nil {
		t.Fatal(err)
	}
	defer l.unlock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		states, err := h.engine.Store.All()
		if err != nil || len(states) != 1 {
			t.Error(err)
			return
		}
		_ = SummarizeAll(states, true, h.clock.now())
		_ = Show(states[0], h.clock.now()).Text()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("board blocked on the run lock")
	}
}
