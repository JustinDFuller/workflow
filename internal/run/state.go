package run

import (
	"time"

	"github.com/JustinDFuller/workflow/internal/check"
	"github.com/JustinDFuller/workflow/internal/workflow"
)

type Lifecycle string

const (
	Active  Lifecycle = "active"
	Stopped Lifecycle = "stopped"
	Ended   Lifecycle = "ended"
)

type State string

const (
	StateWork    State = "work"
	StatePending State = "pending"
	StateBroken  State = "broken"
	StateStopped State = "stopped"
	StateEnded   State = "ended"
)

type Artifact struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type CheckLast struct {
	Name string
	check.Result
}

type RunState struct {
	ID         string
	Workflow   *workflow.Workflow
	Worktree   string
	Inputs     map[string]string
	StartedAt  time.Time
	Lifecycle  Lifecycle
	StopReason string
	Step       string
	Visit      int
	Entered    time.Time
	Route      map[string]string
	RouteFrom  string
	Visits     map[string]int
	LastPassed map[string]time.Time
	Results    map[string]CheckLast
	Artifacts  []Artifact
	Events     []Event
	UpdatedAt  time.Time
}

func (r *RunState) apply(e Event) {
	r.Events = append(r.Events, e)
	r.UpdatedAt = e.Time
	switch e.Type {
	case EventStarted:
		r.StartedAt = e.Time
		r.Worktree = e.Worktree
		r.Inputs = e.Inputs
		r.Lifecycle = Active
	case EventStepEntered:
		r.Step = e.Step
		r.Visit = e.Visit
		r.Visits[e.Step] = e.Visit
		r.Entered = e.Time
		r.Route = e.Values
		r.RouteFrom = e.From
		r.Results = map[string]CheckLast{}
	case EventCheckResult:
		if e.Step == r.Step && e.Visit == r.Visit {
			r.Results[e.Check] = CheckLast{Name: e.Check, Result: check.Result{
				Status: e.Status, Message: e.Message, Outcome: e.Outcome, Values: e.Values,
			}}
		}
	case EventStepPassed:
		r.LastPassed[e.Step] = e.Time
	case EventArtifact:
		for i := range r.Artifacts {
			if r.Artifacts[i].Name == e.Name {
				r.Artifacts[i].Address = e.Address
				return
			}
		}
		r.Artifacts = append(r.Artifacts, Artifact{Name: e.Name, Address: e.Address})
	case EventStopped:
		r.Lifecycle = Stopped
		r.StopReason = e.Reason
	case EventEnded:
		r.Lifecycle = Ended
	}
}

func (r *RunState) CurrentStep() *workflow.Step {
	return r.Workflow.Steps[r.Step]
}

type CheckView struct {
	Name   string
	Ran    bool
	Result check.Result
}

func (r *RunState) CheckViews() []CheckView {
	if r.Lifecycle != Active {
		return nil
	}
	step := r.CurrentStep()
	views := make([]CheckView, 0, len(step.Checks))
	stopped := false
	for _, name := range step.Checks {
		last, ok := r.Results[name]
		if stopped || !ok {
			views = append(views, CheckView{Name: name})
			stopped = true
			continue
		}
		views = append(views, CheckView{Name: name, Ran: true, Result: last.Result})
		if !last.Passed() {
			stopped = true
		}
	}
	return views
}

func (r *RunState) State() State {
	switch r.Lifecycle {
	case Stopped:
		return StateStopped
	case Ended:
		return StateEnded
	}
	for _, v := range r.CheckViews() {
		if !v.Ran {
			break
		}
		switch v.Result.Status {
		case check.Pending:
			return StatePending
		case check.Broken:
			return StateBroken
		case check.Fail:
			return StateWork
		}
	}
	return StateWork
}

func (r *RunState) Reason() string {
	if r.Lifecycle == Stopped {
		return r.StopReason
	}
	for _, v := range r.CheckViews() {
		if v.Ran && !v.Result.Passed() {
			return v.Result.Message
		}
	}
	return ""
}

func (r *RunState) FailingCheck() (CheckView, bool) {
	for _, v := range r.CheckViews() {
		if v.Ran && !v.Result.Passed() {
			return v, true
		}
	}
	return CheckView{}, false
}

func (r *RunState) Deadline() (time.Time, bool) {
	step := r.CurrentStep()
	if step == nil || step.Timeout <= 0 {
		return time.Time{}, false
	}
	return r.Entered.Add(step.Timeout), true
}

func Replay(events []Event) (*RunState, error) {
	r := &RunState{
		Visits:     map[string]int{},
		LastPassed: map[string]time.Time{},
		Results:    map[string]CheckLast{},
	}
	for i, e := range events {
		if i == 0 {
			if e.Type != EventStarted {
				return nil, errFirstEvent
			}
			w, err := workflow.Load([]byte(e.YAML), e.File, e.Dir)
			if err != nil {
				return nil, err
			}
			r.Workflow = w
		}
		r.apply(e)
	}
	if r.Workflow == nil {
		return nil, errFirstEvent
	}
	if r.CurrentStep() == nil {
		return nil, errNoStep
	}
	return r, nil
}
