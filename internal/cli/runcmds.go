package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/JustinDFuller/workflow/internal/run"
	"github.com/JustinDFuller/workflow/internal/workflow"
)

func (e Env) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func printReport(ev *run.Evaluation, asJSON bool, env Env) int {
	report := run.Describe(ev.State, ev.Passed, env.now())
	if asJSON {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return env.fail(err)
		}
		env.printf("%s\n", data)
	} else {
		env.printf("%s", report.Text())
	}
	return run.ExitCode(report.State)
}

func start(args []string, o *options, env Env) int {
	if len(args) != 1 {
		fmt.Fprintln(env.Stderr, "usage: wf start <workflow-file> [--input name=value]...")
		return ExitUsage
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return env.fail(err)
	}
	w, err := workflow.LoadFile(args[0])
	if err != nil {
		if problems, ok := err.(workflow.Problems); ok {
			for _, p := range problems {
				fmt.Fprintln(env.Stderr, p)
			}
			fmt.Fprintf(env.Stderr, "wf: no run was started: %d problem(s) in %s\n", len(problems), args[0])
			return ExitUsage
		}
		return env.fail(err)
	}
	given := map[string]string{}
	for _, kv := range o.inputs {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return env.fail(fmt.Errorf("input %q must be name=value", kv))
		}
		given[k] = v
	}
	cwd, err := env.cwd()
	if err != nil {
		return env.fail(err)
	}
	root, err := worktreeRoot(cwd)
	if err != nil {
		return env.fail(err)
	}
	store, err := env.store()
	if err != nil {
		return env.fail(err)
	}
	engine := run.NewEngine(store)
	id, err := engine.Start(w, data, root, given)
	if err != nil {
		return env.fail(err)
	}
	ev, err := engine.Evaluate(env.context(), id)
	if err != nil {
		return env.fail(err)
	}
	printReport(ev, o.json, env)
	return 0
}

func next(args []string, o *options, env Env) int {
	if len(args) != 0 {
		fmt.Fprintln(env.Stderr, "usage: wf next [--json] [--run id]")
		return ExitUsage
	}
	store, st, err := findRun(o, env)
	if err != nil {
		return env.fail(err)
	}
	ev, err := run.NewEngine(store).Evaluate(env.context(), st.ID)
	if err != nil {
		return env.fail(err)
	}
	return printReport(ev, o.json, env)
}

func wait(args []string, o *options, env Env) int {
	if len(args) != 0 {
		fmt.Fprintln(env.Stderr, "usage: wf wait [--timeout d] [--interval d] [--json] [--run id]")
		return ExitUsage
	}
	store, st, err := findRun(o, env)
	if err != nil {
		return env.fail(err)
	}
	ev, err := run.NewEngine(store).Wait(env.context(), st.ID, run.WaitOptions{Timeout: o.timeout.d, Interval: o.interval.d})
	if err != nil {
		return env.fail(err)
	}
	return printReport(ev, o.json, env)
}
