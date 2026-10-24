package run

import (
	"fmt"
	"strings"
	"time"

	"github.com/JustinDFuller/workflow/internal/check"
	"github.com/JustinDFuller/workflow/internal/workflow"
)

const HistoryLimit = 12

type RunSummary struct {
	Run           string     `json:"run"`
	Workflow      string     `json:"workflow"`
	Worktree      string     `json:"worktree"`
	Step          string     `json:"step"`
	State         State      `json:"state"`
	NeedsYou      bool       `json:"needs_you"`
	Detail        string     `json:"detail,omitempty"`
	Artifacts     []Artifact `json:"artifacts"`
	InStepSeconds int64      `json:"in_step_seconds"`
	StartedAt     time.Time  `json:"started_at"`
}

func NeedsYou(st *RunState) bool {
	switch st.State() {
	case StateStopped, StateBroken:
		return true
	case StatePending:
		step := st.CurrentStep()
		return step != nil && step.NeedsYou
	}
	return false
}

func Summarize(st *RunState, now time.Time) RunSummary {
	s := RunSummary{
		Run: st.ID, Workflow: st.Workflow.Name, Worktree: st.Worktree,
		Step: st.Step, State: st.State(), NeedsYou: NeedsYou(st),
		Detail: st.Reason(), Artifacts: st.Artifacts, StartedAt: st.StartedAt,
	}
	if s.Artifacts == nil {
		s.Artifacts = []Artifact{}
	}
	if st.Lifecycle == Active {
		s.InStepSeconds = int64(now.Sub(st.Entered).Seconds())
	}
	return s
}

func SummarizeAll(states []*RunState, includeEnded bool, now time.Time) []RunSummary {
	out := []RunSummary{}
	for _, st := range states {
		if st.Lifecycle == Ended && !includeEnded {
			continue
		}
		out = append(out, Summarize(st, now))
	}
	return out
}

func ago(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	return workflow.CompactDuration(d.Truncate(time.Second))
}

func RunsText(list []RunSummary) string {
	if len(list) == 0 {
		return "No runs to show. Start one with: wf start <workflow-file>\n"
	}
	var b strings.Builder
	for i, s := range list {
		if i > 0 {
			b.WriteString("\n")
		}
		who := "does not need you"
		if s.NeedsYou {
			who = "NEEDS YOU"
		}
		fmt.Fprintf(&b, "%s  %s  %s  %s\n", s.Run, s.Workflow, strings.ToUpper(string(s.State)), who)
		fmt.Fprintf(&b, "  worktree   %s\n", s.Worktree)
		if s.State == StateEnded {
			fmt.Fprintf(&b, "  step       %s (finished)\n", s.Step)
		} else if s.State == StateStopped {
			fmt.Fprintf(&b, "  step       %s (stopped)\n", s.Step)
		} else {
			fmt.Fprintf(&b, "  step       %s (%s in this step)\n", s.Step, ago(time.Duration(s.InStepSeconds)*time.Second))
		}
		switch s.State {
		case StatePending:
			fmt.Fprintf(&b, "  waiting on %s\n", s.Detail)
		case StateStopped:
			fmt.Fprintf(&b, "  stopped    %s\n", s.Detail)
		case StateBroken:
			fmt.Fprintf(&b, "  broken     %s\n", s.Detail)
		case StateWork:
			if s.Detail != "" {
				fmt.Fprintf(&b, "  working on %s\n", s.Detail)
			}
		}
		if len(s.Artifacts) > 0 {
			parts := make([]string, 0, len(s.Artifacts))
			for _, a := range s.Artifacts {
				parts = append(parts, a.Name+"="+a.Address)
			}
			fmt.Fprintf(&b, "  artifacts  %s\n", strings.Join(parts, "  "))
		}
	}
	return b.String()
}

type StepLine struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Visits    int    `json:"visits"`
	MaxVisits int    `json:"max_visits"`
	PassedAgo int64  `json:"passed_seconds_ago,omitempty"`
}

type HistoryLine struct {
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}

type Detail struct {
	Run             string        `json:"run"`
	Workflow        string        `json:"workflow"`
	Worktree        string        `json:"worktree"`
	State           State         `json:"state"`
	NeedsYou        bool          `json:"needs_you"`
	Step            string        `json:"step"`
	StartedSeconds  int64         `json:"started_seconds_ago"`
	InStepSeconds   int64         `json:"in_step_seconds"`
	Steps           []StepLine    `json:"steps"`
	Job             string        `json:"job"`
	DeadlineSeconds *int64        `json:"deadline_seconds_left,omitempty"`
	Checks          []CheckLine   `json:"checks"`
	StopReason      string        `json:"stop_reason,omitempty"`
	Artifacts       []Artifact    `json:"artifacts"`
	History         []HistoryLine `json:"history"`
	now             time.Time
}

func Show(st *RunState, now time.Time) Detail {
	d := Detail{
		Run: st.ID, Workflow: st.Workflow.Name, Worktree: st.Worktree,
		State: st.State(), NeedsYou: NeedsYou(st), Step: st.Step,
		StartedSeconds: int64(now.Sub(st.StartedAt).Seconds()),
		Artifacts:      st.Artifacts, StopReason: st.StopReason, now: now,
		Checks: []CheckLine{}, History: []HistoryLine{},
	}
	if d.Artifacts == nil {
		d.Artifacts = []Artifact{}
	}
	for _, name := range st.Workflow.StepOrder {
		step := st.Workflow.Steps[name]
		line := StepLine{Name: name, Status: "not reached", Visits: st.Visits[name], MaxVisits: step.VisitLimit()}
		if passed, ok := st.LastPassed[name]; ok {
			line.Status = "passed"
			line.PassedAgo = int64(now.Sub(passed).Seconds())
		}
		if st.Lifecycle == Active && name == st.Step {
			line.Status = "current"
		}
		d.Steps = append(d.Steps, line)
	}
	if st.Lifecycle == Active {
		d.InStepSeconds = int64(now.Sub(st.Entered).Seconds())
		if step := st.CurrentStep(); step != nil {
			d.Job = step.Job
		}
		if deadline, ok := st.Deadline(); ok {
			left := int64(deadline.Sub(now).Seconds())
			d.DeadlineSeconds = &left
		}
		for _, v := range st.CheckViews() {
			line := CheckLine{Name: v.Name, Status: "not run yet"}
			if v.Ran {
				line.Status = string(v.Result.Status)
				line.Message = v.Result.Message
			}
			d.Checks = append(d.Checks, line)
		}
	}
	events := st.Events
	for i := len(events) - 1; i >= 0 && len(d.History) < HistoryLimit; i-- {
		if text, ok := sentence(events[i]); ok {
			d.History = append(d.History, HistoryLine{Time: events[i].Time, Text: text})
		}
	}
	return d
}

func sentence(e Event) (string, bool) {
	switch e.Type {
	case EventStarted:
		return "run started", true
	case EventCheckResult:
		switch e.Status {
		case check.Pass:
			if e.Outcome != "" {
				return fmt.Sprintf("%s reported %s", e.Check, e.Outcome), true
			}
			return fmt.Sprintf("%s passed", e.Check), true
		case check.Fail:
			return fmt.Sprintf("%s failed: %s", e.Check, e.Message), true
		case check.Pending:
			return fmt.Sprintf("%s is pending: %s", e.Check, e.Message), true
		default:
			return fmt.Sprintf("%s is broken: %s", e.Check, e.Message), true
		}
	case EventStepPassed:
		switch e.Target {
		case workflow.TargetEnd:
			return fmt.Sprintf("%s passed, the workflow is finished", e.Step), true
		case workflow.TargetStop:
			return fmt.Sprintf("%s passed, the run was routed to stop", e.Step), true
		}
		return fmt.Sprintf("%s passed, moved to %s", e.Step, e.Target), true
	case EventArtifact:
		return fmt.Sprintf("artifact %s reported: %s", e.Name, e.Address), true
	case EventStopped:
		return "run stopped: " + e.Reason, true
	case EventEnded:
		return "run ended", true
	case EventGuardReleased:
		return e.Reason, true
	}
	return "", false
}

func (d Detail) Text() string {
	var b strings.Builder
	who := "does not need you"
	if d.NeedsYou {
		who = "NEEDS YOU"
	}
	fmt.Fprintf(&b, "%s  run %s  %s  %s  %s\n", d.Workflow, d.Run, d.Worktree, strings.ToUpper(string(d.State)), who)
	if d.State == StateEnded || d.State == StateStopped {
		fmt.Fprintf(&b, "started %s ago\n\n", ago(time.Duration(d.StartedSeconds)*time.Second))
	} else {
		fmt.Fprintf(&b, "started %s ago  in this step %s\n\n", ago(time.Duration(d.StartedSeconds)*time.Second), ago(time.Duration(d.InStepSeconds)*time.Second))
	}
	b.WriteString("Steps\n")
	width := 0
	for _, s := range d.Steps {
		if len(s.Name) > width {
			width = len(s.Name)
		}
	}
	for _, s := range d.Steps {
		mark := "[ ]"
		switch s.Status {
		case "passed":
			mark = "[x]"
		case "current":
			mark = "[>]"
		}
		visits := ""
		if s.Visits > 0 {
			visits = fmt.Sprintf("%d of %d", s.Visits, s.MaxVisits)
		}
		note := ""
		switch {
		case s.Status == "current":
			note = "current"
		case s.Status == "passed":
			note = "passed " + ago(time.Duration(s.PassedAgo)*time.Second) + " ago"
		default:
			note = "not reached"
		}
		fmt.Fprintf(&b, "  %s %-*s  %-8s  %s\n", mark, width, s.Name, visits, note)
	}
	b.WriteString("\n")
	switch d.State {
	case StateEnded:
		b.WriteString("The workflow is finished.\n\n")
	case StateStopped:
		fmt.Fprintf(&b, "Stopped at %s: %s\n\n", d.Step, d.StopReason)
	default:
		fmt.Fprintf(&b, "Now: %s\n", d.Step)
		if d.Job != "" {
			fmt.Fprintf(&b, "Job:\n%s\n", indent(strings.TrimRight(d.Job, "\n")))
		} else {
			b.WriteString("Job: none. This step waits for its checks.\n")
		}
		if d.DeadlineSeconds != nil {
			if *d.DeadlineSeconds >= 0 {
				fmt.Fprintf(&b, "Deadline: %s left\n", ago(time.Duration(*d.DeadlineSeconds)*time.Second))
			} else {
				b.WriteString("Deadline: passed\n")
			}
		}
		b.WriteString("Checks\n")
		for _, c := range d.Checks {
			if c.Message != "" {
				fmt.Fprintf(&b, "  %-12s %s: %s\n", c.Status, c.Name, firstLine(c.Message))
			} else {
				fmt.Fprintf(&b, "  %-12s %s\n", c.Status, c.Name)
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("Artifacts\n")
	if len(d.Artifacts) == 0 {
		b.WriteString("  none yet\n")
	}
	for _, a := range d.Artifacts {
		fmt.Fprintf(&b, "  %-14s %s\n", a.Name, a.Address)
	}
	b.WriteString("\nHistory\n")
	for _, h := range d.History {
		fmt.Fprintf(&b, "  %s  %s\n", clockTime(h.Time, d.now), firstLine(h.Text))
	}
	return b.String()
}

func clockTime(t, now time.Time) string {
	t = t.In(now.Location())
	if t.YearDay() == now.YearDay() && t.Year() == now.Year() {
		return t.Format("15:04")
	}
	return t.Format("Jan 2 15:04")
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

func firstLine(s string) string {
	line, _, more := strings.Cut(s, "\n")
	if more {
		return line + " ..."
	}
	return line
}
