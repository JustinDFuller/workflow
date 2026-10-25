package run

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/JustinDFuller/workflow/internal/workflow"
)

const FallbackWait = time.Hour

type CheckLine struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Report struct {
	Run           string            `json:"run"`
	Workflow      string            `json:"workflow"`
	Worktree      string            `json:"worktree"`
	Step          string            `json:"step"`
	Visit         int               `json:"visit"`
	MaxVisits     int               `json:"max_visits"`
	State         State             `json:"state"`
	Job           string            `json:"job"`
	RouteFrom     string            `json:"route_from,omitempty"`
	Route         map[string]string `json:"route,omitempty"`
	Reason        string            `json:"reason"`
	Checks        []CheckLine       `json:"checks,omitempty"`
	Passed        []string          `json:"passed,omitempty"`
	NextCommand   string            `json:"next_command"`
	Background    bool              `json:"background"`
	ToolTimeout   time.Duration     `json:"-"`
	ToolTimeoutMs int64             `json:"tool_timeout_ms,omitempty"`
	StopReason    string            `json:"stop_reason,omitempty"`
}

func NextCommand(st *RunState, now time.Time) (command string, background bool, toolTimeout time.Duration) {
	switch st.State() {
	case StateStopped, StateEnded:
		return "", false, 0
	case StatePending:
		left := FallbackWait
		if deadline, ok := st.Deadline(); ok {
			left = deadline.Sub(now)
			if left < time.Second {
				left = time.Second
			}
		}
		return "wf wait --timeout " + workflow.CompactDuration(left), true, left
	}
	step := st.CurrentStep()
	return "wf next", step != nil && step.Slow, 0
}

func Describe(st *RunState, passed []string, now time.Time) Report {
	r := Report{
		Run: st.ID, Workflow: st.Workflow.Name, Worktree: st.Worktree,
		Step: st.Step, Visit: st.Visit, State: st.State(), Reason: st.Reason(),
		Passed: passed, StopReason: st.StopReason,
	}
	if step := st.CurrentStep(); step != nil {
		r.MaxVisits = step.VisitLimit()
		if st.Lifecycle == Active {
			r.Job = step.Job
			r.Route = st.Route
			r.RouteFrom = st.RouteFrom
		}
	}
	for _, v := range st.CheckViews() {
		line := CheckLine{Name: v.Name, Status: "not run yet"}
		if v.Ran {
			line.Status = string(v.Result.Status)
			line.Message = v.Result.Message
		}
		r.Checks = append(r.Checks, line)
	}
	r.NextCommand, r.Background, r.ToolTimeout = NextCommand(st, now)
	r.ToolTimeoutMs = r.ToolTimeout.Milliseconds()
	return r
}

func (r Report) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Run %s (%s) in %s\n", r.Run, r.Workflow, r.Worktree)
	for _, name := range r.Passed {
		fmt.Fprintf(&b, "Passed: %s\n", name)
	}
	switch r.State {
	case StateEnded:
		fmt.Fprintf(&b, "State: ended\nThe workflow is finished. There is nothing more to do.\n")
		return b.String()
	case StateStopped:
		fmt.Fprintf(&b, "State: stopped\nThe run stopped at step %s and cannot continue.\nReason: %s\n", r.Step, r.StopReason)
		fmt.Fprintf(&b, "Do not work around this. Tell the person and wait for them to start a new run.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "Step: %s (visit %d of %d)\n", r.Step, r.Visit, r.MaxVisits)
	fmt.Fprintf(&b, "State: %s\n", r.State)
	if r.Job != "" {
		fmt.Fprintf(&b, "Job:\n%s\n", strings.TrimRight(r.Job, "\n"))
	} else {
		fmt.Fprintf(&b, "Job: none. This step waits for its checks.\n")
	}
	if len(r.Route) > 0 {
		fmt.Fprintf(&b, "Values from %s:\n", r.RouteFrom)
		keys := make([]string, 0, len(r.Route))
		for k := range r.Route {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "%s:\n%s\n", k, strings.TrimRight(r.Route[k], "\n"))
		}
	}
	switch r.State {
	case StateBroken:
		fmt.Fprintf(&b, "A check could not run: %s\n", r.Reason)
		fmt.Fprintf(&b, "Do not work around it. If it is not something you caused, tell the person.\n")
	case StatePending:
		fmt.Fprintf(&b, "Waiting on: %s\n", r.Reason)
	default:
		fmt.Fprintf(&b, "Not done yet: %s\n", r.Reason)
	}
	fmt.Fprintf(&b, "Next: %s\n", r.nextLine())
	return b.String()
}

func (r Report) nextLine() string {
	if !r.Background {
		return r.NextCommand
	}
	if r.ToolTimeout > 0 {
		return fmt.Sprintf("run this in the background with the Bash tool timeout set to %d milliseconds: %s", r.ToolTimeout.Milliseconds()+60000, r.NextCommand)
	}
	return "run this in the background: " + r.NextCommand
}

const (
	ExitWork    = 0
	ExitUsage   = 1
	ExitPending = 10
	ExitBroken  = 11
	ExitStopped = 12
	ExitEnded   = 13
)

func ExitCode(s State) int {
	switch s {
	case StatePending:
		return ExitPending
	case StateBroken:
		return ExitBroken
	case StateStopped:
		return ExitStopped
	case StateEnded:
		return ExitEnded
	}
	return ExitWork
}
