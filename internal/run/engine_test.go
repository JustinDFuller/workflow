package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JustinDFuller/workflow/internal/check"
	"github.com/JustinDFuller/workflow/internal/workflow"
)

type fakeChecks struct {
	mu      sync.Mutex
	results map[string]check.Result
	calls   map[string]int
	queues  map[string][]check.Result
	delay   time.Duration
}

func newFake() *fakeChecks {
	return &fakeChecks{results: map[string]check.Result{}, calls: map[string]int{}, queues: map[string][]check.Result{}}
}

func (f *fakeChecks) set(name string, r check.Result) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results[name] = r
	delete(f.queues, name)
}

func (f *fakeChecks) queue(name string, rs ...check.Result) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queues[name] = rs
}

func (f *fakeChecks) count(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[name]
}

func (f *fakeChecks) Run(_ context.Context, c *workflow.Check, _ check.Env) check.Result {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[c.Name]++
	if q := f.queues[c.Name]; len(q) > 0 {
		f.queues[c.Name] = q[1:]
		return q[0]
	}
	if r, ok := f.results[c.Name]; ok {
		return r
	}
	return check.Result{Status: check.Pass}
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type harness struct {
	t      *testing.T
	engine *Engine
	fake   *fakeChecks
	clock  *clock
	id     string
}

const flowYAML = `name: flow
inputs:
  topic:
    default: letters
start: draft
checks:
  one:
    run: "true"
  two:
    run: "true"
  three:
    run: "true"
  gate:
    run: "true"
  review:
    run: "true"
    outcomes: [approved, changes_requested]
steps:
  draft:
    job: Write the draft.
    checks: [one, two, three]
    then: gated
  gated:
    checks: [gate]
    timeout: 1h
    then: reviewing
  reviewing:
    checks: [review]
    timeout: 14d
    needs_you: true
    on:
      approved: end
      changes_requested: revise
  revise:
    job: Fix what the review found.
    checks: [one]
    then: reviewing
`

func newHarness(t *testing.T, yamlText string) *harness {
	t.Helper()
	clk := &clock{t: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	store := &Store{Home: t.TempDir(), Now: clk.now}
	fake := newFake()
	engine := &Engine{Store: store, Runner: fake}
	w, err := workflow.Load([]byte(yamlText), "flow.yaml", "/wfdir")
	if err != nil {
		t.Fatal(err)
	}
	id, err := engine.Start(w, []byte(yamlText), "/work/tree", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, engine: engine, fake: fake, clock: clk, id: id}
}

func (h *harness) next() *Evaluation {
	h.t.Helper()
	ev, err := h.engine.Evaluate(context.Background(), h.id)
	if err != nil {
		h.t.Fatal(err)
	}
	return ev
}

func (h *harness) events(kind string) []Event {
	h.t.Helper()
	st, err := h.engine.Store.Load(h.id)
	if err != nil {
		h.t.Fatal(err)
	}
	var out []Event
	for _, e := range st.Events {
		if e.Type == kind {
			out = append(out, e)
		}
	}
	return out
}

func fail(msg string) check.Result    { return check.Result{Status: check.Fail, Message: msg} }
func pending(msg string) check.Result { return check.Result{Status: check.Pending, Message: msg} }
func broken(msg string) check.Result  { return check.Result{Status: check.Broken, Message: msg} }

func TestStartRecordsInputsAndEntersStartStep(t *testing.T) {
	h := newHarness(t, flowYAML)
	st, err := h.engine.Store.Load(h.id)
	if err != nil {
		t.Fatal(err)
	}
	if st.Step != "draft" || st.Visit != 1 || st.Inputs["topic"] != "letters" || st.Worktree != "/work/tree" {
		t.Fatalf("%+v", st)
	}
	if st.Workflow.Name != "flow" || st.Lifecycle != Active {
		t.Fatalf("%+v", st)
	}
}

func TestStartRefusalsAndReplacement(t *testing.T) {
	h := newHarness(t, flowYAML)
	w, _ := workflow.Load([]byte(flowYAML), "flow.yaml", "/wfdir")
	_, err := h.engine.Start(w, []byte(flowYAML), "/work/tree", nil)
	var active ActiveRunError
	if !asActive(err, &active) || active.ID != h.id || !strings.Contains(err.Error(), h.id) {
		t.Fatalf("want active run error naming %s, got %v", h.id, err)
	}
	if _, err := h.engine.Start(w, []byte(flowYAML), "/other/tree", nil); err != nil {
		t.Fatalf("other worktree: %v", err)
	}
	if _, err := h.engine.Start(w, []byte(flowYAML), "/x", map[string]string{"nope": "1"}); err == nil {
		t.Fatal("unknown input accepted")
	}
	h.fake.set("gate", check.Result{Status: check.Pass})
	h.fake.set("review", check.Result{Status: check.Pass, Outcome: "approved"})
	if ev := h.next(); ev.State.State() != StateEnded {
		t.Fatalf("not ended: %v", ev.State.State())
	}
	h.clock.advance(time.Minute)
	newID, err := h.engine.Start(w, []byte(flowYAML), "/work/tree", nil)
	if err != nil || newID == h.id {
		t.Fatalf("replacement: %v %s", err, newID)
	}
	if old, err := h.engine.Store.Load(h.id); err != nil || old.Lifecycle != Ended {
		t.Fatalf("old run not kept: %v", err)
	}
	latest, err := h.engine.Store.ForWorktree("/work/tree")
	if err != nil || latest.ID != newID {
		t.Fatalf("latest %v %v", latest, err)
	}
	if _, err := h.engine.Store.ForWorktree("/nowhere"); err == nil {
		t.Fatal("expected no run")
	}
	if _, err := h.engine.Store.Load("absent"); err == nil {
		t.Fatal("expected missing run error")
	}
}

func asActive(err error, target *ActiveRunError) bool {
	a, ok := err.(ActiveRunError)
	if ok {
		*target = a
	}
	return ok
}

func TestWorkWhenCheckFails(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("two", fail("3 tasks remain"))
	ev := h.next()
	r := Describe(ev.State, ev.Passed, h.clock.now())
	if r.State != StateWork || r.Reason != "3 tasks remain" || r.Job != "Write the draft." || r.NextCommand != "wf next" {
		t.Fatalf("%+v", r)
	}
	text := r.Text()
	for _, want := range []string{"Step: draft (visit 1 of 10)", "Job:\nWrite the draft.", "Not done yet: 3 tasks remain", "Next: wf next"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
	if ExitCode(r.State) != ExitWork {
		t.Fatal("exit")
	}
	if h.fake.count("three") != 0 {
		t.Fatal("third check ran after failure")
	}
}

func TestEarlierChecksStayVisible(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("three", pending("ci running"))
	ev := h.next()
	views := ev.State.CheckViews()
	if len(views) != 3 || !views[0].Ran || !views[1].Ran || !views[2].Ran || views[2].Result.Status != check.Pending {
		t.Fatalf("%+v", views)
	}
	if got := len(h.events(EventCheckResult)); got != 3 {
		t.Fatalf("result events: %d", got)
	}
	h.fake.set("two", fail("regressed"))
	ev = h.next()
	views = ev.State.CheckViews()
	if views[1].Result.Status != check.Fail || views[2].Ran {
		t.Fatalf("after regression: %+v", views)
	}
	if ev.State.State() != StateWork {
		t.Fatalf("state %v", ev.State.State())
	}
	h.fake.set("two", check.Result{Status: check.Pass})
	ev = h.next()
	if ev.State.State() != StatePending {
		t.Fatalf("state %v", ev.State.State())
	}
}

func TestStepPassesAndCascades(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("no review yet"))
	ev := h.next()
	if ev.State.Step != "reviewing" || ev.State.State() != StatePending {
		t.Fatalf("step %s state %s", ev.State.Step, ev.State.State())
	}
	if strings.Join(ev.Passed, ",") != "draft,gated" {
		t.Fatalf("passed %v", ev.Passed)
	}
	text := Describe(ev.State, ev.Passed, h.clock.now()).Text()
	if !strings.Contains(text, "Passed: draft") || !strings.Contains(text, "Passed: gated") || !strings.Contains(text, "This step waits for its checks") {
		t.Fatal(text)
	}
}

func TestPendingNeverBlocksAndNamesWait(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("no review yet"))
	start := time.Now()
	ev := h.next()
	if time.Since(start) > time.Second {
		t.Fatal("blocked")
	}
	r := Describe(ev.State, nil, h.clock.now())
	if r.NextCommand != "wf wait --timeout 14d" || !r.Background || r.ToolTimeout != 14*24*time.Hour {
		t.Fatalf("%+v", r)
	}
	if ExitCode(r.State) != ExitPending {
		t.Fatal("exit")
	}
	h.clock.advance(26 * time.Hour)
	ev = h.next()
	r = Describe(ev.State, nil, h.clock.now())
	if r.NextCommand != "wf wait --timeout 12d22h" {
		t.Fatalf("%q", r.NextCommand)
	}
	if !strings.Contains(r.Text(), "run this in the background") {
		t.Fatal(r.Text())
	}
}

func TestPollingRecordsOnce(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("no review yet"))
	for i := 0; i < 1000; i++ {
		h.next()
	}
	var reviewResults int
	for _, e := range h.events(EventCheckResult) {
		if e.Check == "review" {
			reviewResults++
		}
	}
	if reviewResults != 1 {
		t.Fatalf("review results recorded %d times", reviewResults)
	}
	h.fake.set("review", pending("one review requested"))
	h.next()
	reviewResults = 0
	for _, e := range h.events(EventCheckResult) {
		if e.Check == "review" {
			reviewResults++
		}
	}
	if reviewResults != 2 {
		t.Fatalf("changed message recorded %d times", reviewResults)
	}
}

func TestNewVisitIgnoresEarlierResults(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.queue("review", check.Result{Status: check.Pass, Outcome: "changes_requested", Values: map[string]string{"comments": "rename x"}})
	h.fake.queue("one", check.Result{Status: check.Pass}, fail("fix pending"))
	ev := h.next()
	if ev.State.Step != "revise" || ev.State.Visit != 1 {
		t.Fatalf("%s %d", ev.State.Step, ev.State.Visit)
	}
	h.fake.set("one", check.Result{Status: check.Pass})
	h.fake.set("review", pending("waiting again"))
	ev = h.next()
	if ev.State.Step != "reviewing" || ev.State.Visit != 2 {
		t.Fatalf("%s %d", ev.State.Step, ev.State.Visit)
	}
	views := ev.State.CheckViews()
	if len(views) != 1 || views[0].Result.Message != "waiting again" {
		t.Fatalf("%+v", views)
	}
}

func TestRouteValuesReachNextStep(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.queue("review", check.Result{Status: check.Pass, Outcome: "changes_requested", Values: map[string]string{"comments": "rename x"}})
	h.fake.queue("one", check.Result{Status: check.Pass}, fail("not yet"))
	ev := h.next()
	r := Describe(ev.State, ev.Passed, h.clock.now())
	if r.Step != "revise" || r.Route["comments"] != "rename x" || !strings.Contains(r.RouteFrom, "reviewing (changes_requested)") {
		t.Fatalf("%+v", r)
	}
	if text := r.Text(); !strings.Contains(text, "Values from reviewing (changes_requested):") || !strings.Contains(text, "rename x") {
		t.Fatal(text)
	}
}

func TestGateFailureStopsRun(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("gate", fail("proposal not approved"))
	ev := h.next()
	r := Describe(ev.State, ev.Passed, h.clock.now())
	if r.State != StateStopped || r.StopReason != "proposal not approved" || ExitCode(r.State) != ExitStopped {
		t.Fatalf("%+v", r)
	}
	if text := r.Text(); !strings.Contains(text, "stopped") || !strings.Contains(text, "proposal not approved") {
		t.Fatal(text)
	}
}

func TestStoppedAndEndedRunNoChecks(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("gate", fail("nope"))
	h.next()
	before := h.fake.count("one") + h.fake.count("gate")
	ev := h.next()
	if ev.State.State() != StateStopped || h.fake.count("one")+h.fake.count("gate") != before {
		t.Fatal("stopped run evaluated checks")
	}

	h2 := newHarness(t, flowYAML)
	h2.fake.set("review", check.Result{Status: check.Pass, Outcome: "approved"})
	h2.next()
	calls := h2.fake.count("review")
	ev = h2.next()
	if ev.State.State() != StateEnded || h2.fake.count("review") != calls {
		t.Fatal("ended run evaluated checks")
	}
	if text := Describe(ev.State, nil, h2.clock.now()).Text(); !strings.Contains(text, "State: ended") {
		t.Fatal(text)
	}
	if ExitCode(StateEnded) != ExitEnded {
		t.Fatal("exit")
	}
}

func TestBrokenState(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("one", broken("gh: not logged in"))
	ev := h.next()
	r := Describe(ev.State, nil, h.clock.now())
	if r.State != StateBroken || ExitCode(r.State) != ExitBroken || !strings.Contains(r.Text(), "Do not work around it") {
		t.Fatalf("%+v\n%s", r, r.Text())
	}
}

func TestVisitLimitStopsRun(t *testing.T) {
	loop := `name: loop
start: a
checks:
  c:
    run: "true"
  d:
    run: "true"
    outcomes: [again, done]
steps:
  a:
    job: x
    checks: [c]
    then: b
  b:
    job: y
    checks: [d]
    on:
      again: a
      done: end
`
	h := newHarness(t, loop)
	h.fake.set("d", check.Result{Status: check.Pass, Outcome: "again"})
	ev := h.next()
	st := ev.State
	if st.State() != StateStopped {
		t.Fatalf("state %v", st.State())
	}
	if !strings.Contains(st.StopReason, `step "a"`) || !strings.Contains(st.StopReason, "limit of 10") || !strings.Contains(st.StopReason, "visit 11") {
		t.Fatal(st.StopReason)
	}
	if st.Visits["a"] != 10 {
		t.Fatalf("visits %v", st.Visits)
	}
}

func TestStepTimeouts(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("no review yet"))
	h.next()
	h.clock.advance(14*24*time.Hour + time.Minute)
	ev := h.next()
	if ev.State.State() != StateStopped || !strings.Contains(ev.State.StopReason, "timed out after 14d") || !strings.Contains(ev.State.StopReason, "reviewing") {
		t.Fatalf("%v %q", ev.State.State(), ev.State.StopReason)
	}

	h = newHarness(t, flowYAML)
	h.fake.set("review", pending("no review yet"))
	h.next()
	h.clock.advance(15 * 24 * time.Hour)
	h.fake.set("review", check.Result{Status: check.Pass, Outcome: "approved"})
	ev = h.next()
	if ev.State.State() != StateEnded {
		t.Fatalf("late approval: %v %q", ev.State.State(), ev.State.StopReason)
	}
}

func TestStopTargetAndMissingRoute(t *testing.T) {
	text := `name: s
start: a
checks:
  c:
    run: "true"
steps:
  a:
    job: x
    checks: [c]
    then: stop
`
	h := newHarness(t, text)
	if ev := h.next(); ev.State.State() != StateStopped || !strings.Contains(ev.State.StopReason, "routed the run to stop") {
		t.Fatalf("%q", ev.State.StopReason)
	}
	h = newHarness(t, flowYAML)
	h.fake.set("review", check.Result{Status: check.Pass, Outcome: "surprise"})
	if ev := h.next(); ev.State.State() != StateStopped || !strings.Contains(ev.State.StopReason, `no route for outcome "surprise"`) {
		t.Fatalf("%q", ev.State.StopReason)
	}
}

func TestArtifactsRecordedAndReplaced(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("one", check.Result{Status: check.Pass, Artifacts: []check.Artifact{{Name: "pr", Address: "https://example.test/pull/1"}}})
	h.fake.set("two", fail("later"))
	ev := h.next()
	if len(ev.State.Artifacts) != 1 || ev.State.Artifacts[0].Address != "https://example.test/pull/1" {
		t.Fatalf("%+v", ev.State.Artifacts)
	}
	h.next()
	if got := len(h.events(EventArtifact)); got != 1 {
		t.Fatalf("artifact events %d", got)
	}
	h.fake.set("one", check.Result{Status: check.Pass, Artifacts: []check.Artifact{{Name: "pr", Address: "https://example.test/pull/2"}, {Name: "change", Address: "openspec/changes/x"}}})
	ev = h.next()
	if len(ev.State.Artifacts) != 2 || ev.State.Artifacts[0].Address != "https://example.test/pull/2" {
		t.Fatalf("%+v", ev.State.Artifacts)
	}
}

func TestStateRebuiltFromLogInNewProcess(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.queue("review", check.Result{Status: check.Pass, Outcome: "changes_requested", Values: map[string]string{"comments": "c"}})
	h.fake.queue("one", check.Result{Status: check.Pass}, fail("x"))
	h.next()
	fresh := &Store{Home: h.engine.Store.Home}
	st, err := fresh.Load(h.id)
	if err != nil {
		t.Fatal(err)
	}
	if st.Step != "revise" || st.Visit != 1 || st.Route["comments"] != "c" || st.Visits["draft"] != 1 || st.Visits["reviewing"] != 1 {
		t.Fatalf("%+v", st)
	}
	all, err := fresh.All()
	if err != nil || len(all) != 1 {
		t.Fatalf("%v %v", all, err)
	}
}

func TestConcurrentEvaluationsAdvanceOnce(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.delay = 50 * time.Millisecond
	h.fake.set("review", pending("waiting"))
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := h.engine.Evaluate(context.Background(), h.id); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	passed := map[string]int{}
	for _, e := range h.events(EventStepPassed) {
		passed[e.Step]++
	}
	if passed["draft"] != 1 || passed["gated"] != 1 {
		t.Fatalf("passed %v", passed)
	}
	entered := 0
	for _, e := range h.events(EventStepEntered) {
		if e.Step == "reviewing" {
			entered++
		}
	}
	if entered != 1 {
		t.Fatalf("reviewing entered %d times", entered)
	}
}

func TestSlowStepWording(t *testing.T) {
	text := strings.Replace(flowYAML, "    job: Write the draft.\n", "    job: Write the draft.\n    slow: true\n", 1)
	h := newHarness(t, text)
	h.fake.set("one", fail("not yet"))
	ev := h.next()
	r := Describe(ev.State, nil, h.clock.now())
	if !r.Background || r.NextCommand != "wf next" || !strings.Contains(r.Text(), "run this in the background: wf next") {
		t.Fatalf("%+v\n%s", r, r.Text())
	}
}

func TestPendingWithoutDeadlineUsesFallback(t *testing.T) {
	text := `name: p
start: a
checks:
  c:
    run: "true"
steps:
  a:
    job: x
    checks: [c]
    then: end
`
	h := newHarness(t, text)
	h.fake.set("c", pending("later"))
	ev := h.next()
	if r := Describe(ev.State, nil, h.clock.now()); r.NextCommand != "wf wait --timeout 1h" {
		t.Fatalf("%q", r.NextCommand)
	}
}

func TestWaitReturnsAtOnceWhenNotPending(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("one", fail("work to do"))
	start := time.Now()
	ev, err := h.engine.Wait(context.Background(), h.id, WaitOptions{Timeout: time.Minute, Interval: time.Minute})
	if err != nil || ev.State.State() != StateWork || time.Since(start) > time.Second {
		t.Fatalf("%v %v", err, ev)
	}
}

func TestWaitResolves(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("waiting"))
	go func() {
		time.Sleep(150 * time.Millisecond)
		h.fake.set("review", check.Result{Status: check.Pass, Outcome: "approved"})
	}()
	ev, err := h.engine.Wait(context.Background(), h.id, WaitOptions{Timeout: 5 * time.Second, Interval: 20 * time.Millisecond})
	if err != nil || ev.State.State() != StateEnded {
		t.Fatalf("%v %v", err, ev.State.State())
	}
	if strings.Join(ev.Passed, ",") != "draft,gated,reviewing" {
		t.Fatalf("passed accumulated %v", ev.Passed)
	}
}

func TestWaitCommandTimeoutReturnsPending(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("waiting"))
	start := time.Now()
	ev, err := h.engine.Wait(context.Background(), h.id, WaitOptions{Timeout: 150 * time.Millisecond, Interval: 40 * time.Millisecond})
	if err != nil || ev.State.State() != StatePending {
		t.Fatalf("%v %v", err, ev)
	}
	if d := time.Since(start); d < 100*time.Millisecond || d > 3*time.Second {
		t.Fatalf("took %v", d)
	}
	if r := Describe(ev.State, nil, h.clock.now()); r.NextCommand == "" {
		t.Fatal("no wait command")
	}
}

func TestWaitEndsOnBroken(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("waiting"))
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.fake.set("review", broken("gh failed"))
	}()
	ev, err := h.engine.Wait(context.Background(), h.id, WaitOptions{Timeout: 5 * time.Second, Interval: 20 * time.Millisecond})
	if err != nil || ev.State.State() != StateBroken || ev.State.Reason() != "gh failed" {
		t.Fatalf("%v %v", err, ev.State.State())
	}
}

func TestWaitDefaultsAndCancel(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("waiting"))
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	if _, err := h.engine.Wait(ctx, h.id, WaitOptions{}); err == nil {
		t.Fatal("expected cancellation")
	}
	if DefaultWaitTimeout != 90*time.Second || DefaultWaitInterval != 30*time.Second {
		t.Fatal("defaults changed")
	}
	if h.engine.WaiterLive(h.id) {
		t.Fatal("heartbeat left behind after cancel")
	}
}

func TestHeartbeatLiveAndRemoved(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.fake.set("review", pending("waiting"))
	var live atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = h.engine.Wait(context.Background(), h.id, WaitOptions{Timeout: 400 * time.Millisecond, Interval: 50 * time.Millisecond})
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !live.Load() {
		if h.engine.WaiterLive(h.id) {
			live.Store(true)
		}
		time.Sleep(10 * time.Millisecond)
	}
	<-done
	if !live.Load() {
		t.Fatal("waiter never live")
	}
	if h.engine.WaiterLive(h.id) {
		t.Fatal("heartbeat remains after wait ended")
	}
}

func TestTwoWaitersKeepTheirOwnHeartbeats(t *testing.T) {
	h := newHarness(t, flowYAML)
	other := filepath.Join(h.engine.Store.runDir(h.id), "waiter-1")
	if err := os.WriteFile(other, []byte(fmt.Sprintf("%d %d", h.clock.now().UnixNano(), int64(30*time.Second))), 0o644); err != nil {
		t.Fatal(err)
	}
	h.engine.beat(h.id, 30*time.Second)
	os.Remove(h.engine.heartbeatPath(h.id))
	if !h.engine.WaiterLive(h.id) {
		t.Fatal("removing one waiter's heartbeat hid the other waiter")
	}
}

func TestWaitReturnsOnTimeWhileAnotherCommandHoldsTheRun(t *testing.T) {
	h := newHarness(t, flowYAML)
	held, err := h.engine.Store.lockRun(context.Background(), h.id)
	if err != nil {
		t.Fatal(err)
	}
	defer held.unlock()
	start := time.Now()
	ev, err := h.engine.Wait(context.Background(), h.id, WaitOptions{Timeout: 200 * time.Millisecond, Interval: 50 * time.Millisecond})
	if err != nil || ev == nil || ev.State == nil {
		t.Fatalf("ev=%v err=%v", ev, err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("wait outlived its timeout")
	}
}

type blockingRunner struct{}

func (blockingRunner) Run(ctx context.Context, _ *workflow.Check, _ check.Env) check.Result {
	<-ctx.Done()
	return check.Result{Status: check.Broken, Message: "check was cancelled"}
}

func TestWaitCutsOffACheckThatOutlivesTheTimeout(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.engine.Runner = blockingRunner{}
	start := time.Now()
	ev, err := h.engine.Wait(context.Background(), h.id, WaitOptions{Timeout: 200 * time.Millisecond, Interval: 50 * time.Millisecond})
	if err != nil || ev == nil || ev.State == nil {
		t.Fatalf("ev=%v err=%v", ev, err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("wait outlived its timeout")
	}
	if got := len(h.events(EventCheckResult)); got != 0 {
		t.Fatalf("a cut-off check recorded %d results", got)
	}
}

func TestDefaultHomeIsAbsolute(t *testing.T) {
	got, err := DefaultHome(func(string) string { return "relative/state" })
	if err != nil || !filepath.IsAbs(got) {
		t.Fatalf("%q %v", got, err)
	}
}

func TestStateDirectoryGuardIgnoresCaseOnMac(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("only macOS volumes fold case")
	}
	dir := t.TempDir()
	if !inside(filepath.Join(strings.ToUpper(dir), "runs", "x"), dir) {
		t.Fatal("a differently cased path escaped the guard")
	}
	if !mentions("echo x > "+strings.ToUpper(dir)+"/runs/x", dir) {
		t.Fatal("a differently cased command escaped the guard")
	}
}

func TestLockGivesUpWhenCancelled(t *testing.T) {
	h := newHarness(t, flowYAML)
	held, err := h.engine.Store.lockRun(context.Background(), h.id)
	if err != nil {
		t.Fatal(err)
	}
	defer held.unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := h.engine.Store.lockRun(ctx, h.id); err == nil {
		t.Fatal("expected the lock attempt to fail")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("lock attempt ignored cancellation")
	}
}

func TestKilledWaiterStopsCountingAsLive(t *testing.T) {
	h := newHarness(t, flowYAML)
	h.engine.beat(h.id, 30*time.Second)
	if !h.engine.WaiterLive(h.id) {
		t.Fatal("fresh heartbeat not live")
	}
	h.clock.advance(59 * time.Second)
	if !h.engine.WaiterLive(h.id) {
		t.Fatal("within twice the interval should be live")
	}
	h.clock.advance(2 * time.Second)
	if h.engine.WaiterLive(h.id) {
		t.Fatal("stale heartbeat still live")
	}
}

func TestEnvReachesChecks(t *testing.T) {
	var seen check.Env
	h := newHarness(t, flowYAML)
	h.engine.Runner = envRecorder{&seen}
	h.next()
	if seen.RunID != h.id || seen.Worktree != "/work/tree" || seen.WorkflowDir != "/wfdir" || seen.Inputs["topic"] != "letters" || seen.Step == "" || seen.Entered.IsZero() {
		t.Fatalf("%+v", seen)
	}
}

type envRecorder struct{ seen *check.Env }

func (e envRecorder) Run(_ context.Context, _ *workflow.Check, env check.Env) check.Result {
	*e.seen = env
	return check.Result{Status: check.Fail, Message: "x"}
}

func TestDefaultHome(t *testing.T) {
	got, err := DefaultHome(func(k string) string {
		if k == HomeEnv {
			return "/custom"
		}
		return ""
	})
	if err != nil || got != "/custom" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = DefaultHome(func(string) string { return "" })
	if err != nil || !strings.HasSuffix(got, "/.wf") {
		t.Fatalf("%q %v", got, err)
	}
}

func TestReplayRejectsBadLogs(t *testing.T) {
	if _, err := Replay(nil); err == nil {
		t.Fatal("empty log accepted")
	}
	if _, err := Replay([]Event{{Type: EventEnded}}); err == nil {
		t.Fatal("log without started event accepted")
	}
	if _, err := Replay([]Event{{Type: EventStarted, YAML: "name: [", File: "f"}}); err == nil {
		t.Fatal("bad workflow accepted")
	}
}

func TestHalfStartedRunIsIgnored(t *testing.T) {
	h := newHarness(t, flowYAML)
	dir := h.engine.Store.runDir("deadbeef")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := h.engine.Store.append("deadbeef", Event{Type: EventStarted, Workflow: "demo", YAML: flowYAML, File: "f", Dir: dir, Worktree: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.engine.Store.Load("deadbeef"); !errors.Is(err, errNoStep) {
		t.Fatalf("err = %v", err)
	}
	runs, err := h.engine.Store.All()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range runs {
		if r.ID == "deadbeef" {
			t.Fatal("half-started run listed")
		}
	}
}

type cancellingRunner struct{ cancel func() }

func (c cancellingRunner) Run(_ context.Context, _ *workflow.Check, _ check.Env) check.Result {
	c.cancel()
	return check.Result{Status: check.Broken, Message: "check was cancelled"}
}

func TestCancellationIsNotRecordedAsABrokenCheck(t *testing.T) {
	h := newHarness(t, denyFlow)
	before := len(h.events(EventCheckResult))
	ctx, cancel := context.WithCancel(context.Background())
	h.engine.Runner = cancellingRunner{cancel: cancel}
	if _, err := h.engine.Evaluate(ctx, h.id); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if got := len(h.events(EventCheckResult)); got != before {
		t.Fatalf("a cancelled evaluation recorded %d check results", got-before)
	}
}
