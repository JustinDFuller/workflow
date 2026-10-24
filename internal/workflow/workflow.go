package workflow

import (
	"fmt"
	"time"
)

const (
	TargetEnd  = "end"
	TargetStop = "stop"

	DefaultMaxVisits = 10
	CacheCommit      = "commit"
)

type Problem struct {
	File    string
	Line    int
	Message string
}

func (p Problem) String() string {
	switch {
	case p.File != "" && p.Line > 0:
		return fmt.Sprintf("%s:%d: %s", p.File, p.Line, p.Message)
	case p.File != "":
		return fmt.Sprintf("%s: %s", p.File, p.Message)
	default:
		return p.Message
	}
}

type Problems []Problem

func (p Problems) Error() string {
	out := ""
	for i, problem := range p {
		if i > 0 {
			out += "\n"
		}
		out += problem.String()
	}
	return out
}

type Input struct {
	Name     string
	Required bool
	Default  string
	Line     int
}

type Check struct {
	Name     string
	Run      string
	Timeout  time.Duration
	Outcomes []string
	Cache    string
	Line     int
}

func (c *Check) HasOutcomes() bool {
	return len(c.Outcomes) > 0
}

type Step struct {
	Name      string
	Job       string
	Checks    []string
	Then      string
	On        map[string]string
	OnOrder   []string
	MaxVisits int
	Timeout   time.Duration
	DenyTools []string
	Slow      bool
	NeedsYou  bool
	Line      int
	lines     map[string]int
}

func (s *Step) HasJob() bool {
	return s.Job != ""
}

func (s *Step) VisitLimit() int {
	if s.MaxVisits > 0 {
		return s.MaxVisits
	}
	return DefaultMaxVisits
}

func (s *Step) line(key string) int {
	if n, ok := s.lines[key]; ok {
		return n
	}
	return s.Line
}

type Workflow struct {
	Name        string
	Description string
	Inputs      []Input
	Start       string
	Checks      map[string]*Check
	CheckOrder  []string
	Steps       map[string]*Step
	StepOrder   []string
	File        string
	Dir         string
	startLine   int
}

func (w *Workflow) Input(name string) (Input, bool) {
	for _, in := range w.Inputs {
		if in.Name == name {
			return in, true
		}
	}
	return Input{}, false
}

func (w *Workflow) FinalCheck(s *Step) *Check {
	if len(s.Checks) == 0 {
		return nil
	}
	return w.Checks[s.Checks[len(s.Checks)-1]]
}

func (w *Workflow) Target(s *Step, outcome string) (string, bool) {
	if final := w.FinalCheck(s); final != nil && final.HasOutcomes() {
		target, ok := s.On[outcome]
		return target, ok
	}
	if s.Then == "" {
		return "", false
	}
	return s.Then, true
}

func (w *Workflow) Targets(s *Step) []string {
	if final := w.FinalCheck(s); final != nil && final.HasOutcomes() {
		out := make([]string, 0, len(s.OnOrder))
		for _, o := range s.OnOrder {
			out = append(out, s.On[o])
		}
		return out
	}
	if s.Then == "" {
		return nil
	}
	return []string{s.Then}
}

func (w *Workflow) ResolveInputs(given map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for name := range given {
		if _, ok := w.Input(name); !ok {
			return nil, fmt.Errorf("unknown input %q", name)
		}
	}
	for _, in := range w.Inputs {
		if v, ok := given[in.Name]; ok {
			out[in.Name] = v
			continue
		}
		if in.Required {
			return nil, fmt.Errorf("missing required input %q", in.Name)
		}
		out[in.Name] = in.Default
	}
	return out, nil
}
