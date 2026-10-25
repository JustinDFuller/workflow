package workflow

import (
	"fmt"
	"sort"
)

func isTerminal(target string) bool {
	return target == TargetEnd || target == TargetStop
}

func Validate(w *Workflow) Problems {
	var problems Problems
	add := func(line int, format string, args ...any) {
		problems = append(problems, Problem{File: w.File, Line: line, Message: fmt.Sprintf(format, args...)})
	}

	if w.Start == "" {
		add(1, "workflow needs a start step")
	} else if _, ok := w.Steps[w.Start]; !ok {
		add(w.startLine, "start step %q is not a step", w.Start)
	}
	if len(w.Steps) == 0 {
		add(1, "workflow needs at least one step")
	}

	checkNames := map[string]bool{}
	for _, name := range w.StepOrder {
		s := w.Steps[name]
		if len(s.Checks) == 0 {
			add(s.Line, "step %q needs at least one check", name)
		}
		for i, cn := range s.Checks {
			c, ok := w.Checks[cn]
			if !ok {
				add(s.line("checks"), "step %q lists check %q which is not in the check library", name, cn)
				continue
			}
			checkNames[cn] = true
			if c.HasOutcomes() && i != len(s.Checks)-1 {
				add(s.line("checks"), "step %q lists check %q, which declares outcomes, before another check; outcome checks must be last", name, cn)
			}
		}
		if !s.HasJob() && s.Timeout == 0 {
			add(s.Line, "step %q has no job and no timeout; a step that waits needs a timeout", name)
		}
		validateRouting(w, s, add)
	}

	if w.Start != "" && w.Steps[w.Start] != nil {
		reached, terminal := reachable(w)
		for _, name := range w.StepOrder {
			if !reached[name] {
				add(w.Steps[name].Line, "step %q cannot be reached from the start step", name)
			}
		}
		if !terminal {
			add(w.Steps[w.Start].Line, "no path leads from the start step to end or stop")
		}
	}
	return problems
}

func validateRouting(w *Workflow, s *Step, add func(int, string, ...any)) {
	final := w.FinalCheck(s)
	byOutcome := final != nil && final.HasOutcomes()
	if byOutcome {
		if s.Then != "" {
			add(s.line("then"), "step %q has a plain next target and a final check that declares outcomes; use only one", s.Name)
		}
		declared := map[string]bool{}
		for _, o := range final.Outcomes {
			declared[o] = true
			if _, ok := s.On[o]; !ok {
				add(s.Line, "step %q does not route the outcome %q declared by check %q", s.Name, o, final.Name)
			}
		}
		for _, o := range s.OnOrder {
			if !declared[o] {
				add(s.line("on."+o), "step %q routes outcome %q which check %q does not declare", s.Name, o, final.Name)
			}
		}
	} else {
		if len(s.On) > 0 {
			add(s.line("on"), "step %q maps outcomes but its final check declares none", s.Name)
		}
		if s.Then == "" && final != nil {
			add(s.Line, "step %q needs a next target", s.Name)
		}
	}
	targets := s.OnOrder
	lines := map[string]int{}
	for _, o := range s.OnOrder {
		lines[s.On[o]] = s.line("on." + o)
	}
	var all []string
	for _, o := range targets {
		all = append(all, s.On[o])
	}
	if s.Then != "" {
		all = append(all, s.Then)
		lines[s.Then] = s.line("then")
	}
	sort.Strings(all)
	for _, t := range all {
		if isTerminal(t) {
			continue
		}
		if _, ok := w.Steps[t]; !ok {
			add(lines[t], "step %q routes to %q which is not a step, end, or stop", s.Name, t)
		}
	}
}

func reachable(w *Workflow) (map[string]bool, bool) {
	reached := map[string]bool{}
	terminal := false
	queue := []string{w.Start}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if reached[name] {
			continue
		}
		s, ok := w.Steps[name]
		if !ok {
			continue
		}
		reached[name] = true
		for _, t := range w.Targets(s) {
			if isTerminal(t) {
				terminal = true
				continue
			}
			queue = append(queue, t)
		}
	}
	return reached, terminal
}
