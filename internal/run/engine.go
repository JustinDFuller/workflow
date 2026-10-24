package run

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/JustinDFuller/workflow/internal/check"
	"github.com/JustinDFuller/workflow/internal/workflow"
)

type Engine struct {
	Store  *Store
	Runner check.Evaluator
}

func NewEngine(store *Store) *Engine {
	return &Engine{
		Store:  store,
		Runner: &check.Runner{Cache: &check.Cache{Dir: store.CacheDir()}},
	}
}

type ActiveRunError struct {
	ID string
}

func (e ActiveRunError) Error() string {
	return fmt.Sprintf("this worktree already has run %s that has not ended or stopped", e.ID)
}

func (e *Engine) Start(w *workflow.Workflow, yamlData []byte, worktree string, given map[string]string) (string, error) {
	inputs, err := w.ResolveInputs(given)
	if err != nil {
		return "", err
	}
	l, err := lock(filepath.Join(e.Store.Home, "start.lock"))
	if err != nil {
		return "", err
	}
	defer l.unlock()
	if prev, err := e.Store.ForWorktree(worktree); err == nil && prev.Lifecycle == Active {
		return "", ActiveRunError{ID: prev.ID}
	}
	id, err := newID()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(e.Store.runDir(id), 0o755); err != nil {
		return "", err
	}
	if err := e.Store.append(id, Event{
		Type: EventStarted, Workflow: w.Name, YAML: string(yamlData), File: w.File, Dir: w.Dir,
		Worktree: worktree, Inputs: inputs,
	}); err != nil {
		return "", err
	}
	if err := e.Store.append(id, Event{Type: EventStepEntered, Step: w.Start, Visit: 1}); err != nil {
		return "", err
	}
	return id, nil
}

func (e *Engine) record(st *RunState, ev Event) error {
	if ev.Time.IsZero() {
		ev.Time = e.Store.now().UTC()
	}
	if err := e.Store.append(st.ID, ev); err != nil {
		return err
	}
	st.apply(ev)
	return nil
}

func (e *Engine) stop(st *RunState, reason string) error {
	return e.record(st, Event{Type: EventStopped, Step: st.Step, Reason: reason})
}

func (e *Engine) env(st *RunState) check.Env {
	return check.Env{
		RunID:       st.ID,
		Step:        st.Step,
		Worktree:    st.Worktree,
		WorkflowDir: st.Workflow.Dir,
		Entered:     st.Entered,
		Inputs:      st.Inputs,
		Route:       st.Route,
	}
}

type Evaluation struct {
	State  *RunState
	Passed []string
}

func (e *Engine) Evaluate(ctx context.Context, id string) (*Evaluation, error) {
	l, err := e.Store.lockRun(id)
	if err != nil {
		return nil, err
	}
	defer l.unlock()
	st, err := e.Store.Load(id)
	if err != nil {
		return nil, err
	}
	before := len(st.Events)
	ev := &Evaluation{State: st}
	for st.Lifecycle == Active {
		if err := ctx.Err(); err != nil {
			return ev, err
		}
		advanced, err := e.evaluateStep(ctx, st)
		if err != nil {
			return ev, err
		}
		if !advanced {
			break
		}
	}
	for _, event := range st.Events[before:] {
		if event.Type == EventStepPassed {
			ev.Passed = append(ev.Passed, event.Step)
		}
	}
	return ev, nil
}

func (e *Engine) evaluateStep(ctx context.Context, st *RunState) (bool, error) {
	w := st.Workflow
	step := st.CurrentStep()
	res := check.EvaluateStep(ctx, w, step, e.Runner, e.env(st))
	for _, named := range res.Checks {
		if err := e.recordResult(st, named); err != nil {
			return false, err
		}
	}
	if res.Over {
		return true, e.advance(st, res)
	}
	last := res.Last()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if step.Timeout > 0 && e.Store.now().After(st.Entered.Add(step.Timeout)) {
		return false, e.stop(st, fmt.Sprintf("step %q timed out after %s; it was still %s: %s", step.Name, workflow.CompactDuration(step.Timeout), last.Status, last.Message))
	}
	if !step.HasJob() && last.Status == check.Fail {
		return false, e.stop(st, last.Message)
	}
	return false, nil
}

func (e *Engine) recordResult(st *RunState, named check.Named) error {
	prev, seen := st.Results[named.Name]
	if !seen || prev.Status != named.Status || prev.Message != named.Message || prev.Outcome != named.Outcome {
		if err := e.record(st, Event{
			Type: EventCheckResult, Step: st.Step, Visit: st.Visit, Check: named.Name,
			Status: named.Status, Message: named.Message, Outcome: named.Outcome, Values: named.Values,
		}); err != nil {
			return err
		}
	}
	for _, a := range named.Artifacts {
		if known, ok := artifactAddress(st, a.Name); ok && known == a.Address {
			continue
		}
		if err := e.record(st, Event{Type: EventArtifact, Step: st.Step, Name: a.Name, Address: a.Address}); err != nil {
			return err
		}
	}
	return nil
}

func artifactAddress(st *RunState, name string) (string, bool) {
	for _, a := range st.Artifacts {
		if a.Name == name {
			return a.Address, true
		}
	}
	return "", false
}

func (e *Engine) advance(st *RunState, res check.StepResult) error {
	w := st.Workflow
	step := st.CurrentStep()
	final := res.Last()
	outcome := final.Outcome
	target, ok := w.Target(step, outcome)
	if !ok {
		return e.stop(st, fmt.Sprintf("step %q has no route for outcome %q", step.Name, outcome))
	}
	if err := e.record(st, Event{Type: EventStepPassed, Step: step.Name, Visit: st.Visit, Outcome: outcome, Target: target}); err != nil {
		return err
	}
	switch target {
	case workflow.TargetEnd:
		return e.record(st, Event{Type: EventEnded, Step: step.Name})
	case workflow.TargetStop:
		return e.stop(st, fmt.Sprintf("step %q routed the run to stop", step.Name))
	}
	next := w.Steps[target]
	visit := st.Visits[target] + 1
	if visit > next.VisitLimit() {
		return e.stop(st, fmt.Sprintf("step %q would be entered for visit %d, over its limit of %d", target, visit, next.VisitLimit()))
	}
	from := step.Name
	if outcome != "" {
		from = fmt.Sprintf("%s (%s)", step.Name, outcome)
	}
	return e.record(st, Event{Type: EventStepEntered, Step: target, Visit: visit, From: from, Values: final.Values})
}

func (e *Engine) heartbeatPath(id string) string {
	return filepath.Join(e.Store.runDir(id), "waiter")
}

func (e *Engine) beat(id string, interval time.Duration) {
	path := e.heartbeatPath(id)
	tmp := path + ".tmp"
	data := fmt.Sprintf("%d %d\n", e.Store.now().UnixNano(), interval.Nanoseconds())
	if err := os.WriteFile(tmp, []byte(data), 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func (e *Engine) WaiterLive(id string) bool {
	data, err := os.ReadFile(e.heartbeatPath(id))
	if err != nil {
		return false
	}
	var at, interval int64
	if _, err := fmt.Sscanf(string(data), "%d %d", &at, &interval); err != nil {
		return false
	}
	age := e.Store.now().Sub(time.Unix(0, at))
	return age <= 2*time.Duration(interval)
}

type WaitOptions struct {
	Timeout  time.Duration
	Interval time.Duration
}

const (
	DefaultWaitTimeout  = 90 * time.Second
	DefaultWaitInterval = 30 * time.Second
)

func (e *Engine) Wait(ctx context.Context, id string, opt WaitOptions) (*Evaluation, error) {
	if opt.Timeout <= 0 {
		opt.Timeout = DefaultWaitTimeout
	}
	if opt.Interval <= 0 {
		opt.Interval = DefaultWaitInterval
	}
	defer os.Remove(e.heartbeatPath(id))
	deadline := time.Now().Add(opt.Timeout)
	var passed []string
	for {
		e.beat(id, opt.Interval)
		ev, err := e.Evaluate(ctx, id)
		if err != nil {
			return ev, err
		}
		passed = append(passed, ev.Passed...)
		ev.Passed = passed
		if ev.State.State() != StatePending {
			return ev, nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return ev, nil
		}
		pause := opt.Interval
		if remaining < pause {
			pause = remaining
		}
		select {
		case <-ctx.Done():
			return ev, ctx.Err()
		case <-time.After(pause):
		}
	}
}
