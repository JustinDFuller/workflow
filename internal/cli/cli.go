package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JustinDFuller/workflow/internal/run"
	"github.com/JustinDFuller/workflow/internal/workflow"
)

const ExitUsage = run.ExitUsage

const usage = `wf keeps a coding agent on a workflow.

Usage:
  wf <command> [arguments]

Commands:
  validate <file>              check a workflow file without running it
  start <file> [--input k=v]   start a run in this worktree
  next [--json] [--run id]     evaluate the current step and say what to do
  wait [--timeout d] [--json]  wait for a pending step to change
  runs [--all] [--watch]       show every run that has not ended
  show [--run id] [--watch]    show one run in detail

Running wf with no command shows the same list as wf runs.
`

type Env struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	Getenv func(string) string
	Cwd    string
	Now    func() time.Time
	Ctx    context.Context
}

func (e Env) env(key string) string {
	if e.Getenv == nil {
		return os.Getenv(key)
	}
	return e.Getenv(key)
}

func (e Env) context() context.Context {
	if e.Ctx != nil {
		return e.Ctx
	}
	return context.Background()
}

func (e Env) cwd() (string, error) {
	if e.Cwd != "" {
		return e.Cwd, nil
	}
	return os.Getwd()
}

func (e Env) store() (*run.Store, error) {
	home, err := run.DefaultHome(e.env)
	if err != nil {
		return nil, err
	}
	return &run.Store{Home: home, Now: e.Now}, nil
}

func (e Env) printf(format string, args ...any) {
	fmt.Fprintf(e.Stdout, format, args...)
}

func (e Env) fail(err error) int {
	fmt.Fprintf(e.Stderr, "wf: %v\n", err)
	return ExitUsage
}

type command struct {
	setup func(fs *flag.FlagSet, o *options)
	run   func(args []string, o *options, env Env) int
}

type options struct {
	json     bool
	all      bool
	watch    bool
	runID    string
	timeout  duration
	interval duration
	inputs   multi
}

type duration struct {
	d time.Duration
}

func (d *duration) String() string { return workflow.CompactDuration(d.d) }

func (d *duration) Set(s string) error {
	v, err := workflow.ParseDuration(s)
	if err != nil {
		return err
	}
	d.d = v
	return nil
}

type multi []string

func (m *multi) String() string { return strings.Join(*m, ",") }

func (m *multi) Set(s string) error {
	*m = append(*m, s)
	return nil
}

var commands = map[string]command{
	"validate": {setup: func(*flag.FlagSet, *options) {}, run: validate},
	"start": {
		setup: func(fs *flag.FlagSet, o *options) {
			fs.Var(&o.inputs, "input", "workflow input as name=value")
		},
		run: start,
	},
	"next": {
		setup: func(fs *flag.FlagSet, o *options) {
			fs.BoolVar(&o.json, "json", false, "print JSON")
			fs.StringVar(&o.runID, "run", "", "run id")
		},
		run: next,
	},
	"runs": {
		setup: func(fs *flag.FlagSet, o *options) {
			fs.BoolVar(&o.json, "json", false, "print JSON")
			fs.BoolVar(&o.all, "all", false, "include ended runs")
			fs.BoolVar(&o.watch, "watch", false, "redraw until interrupted")
			fs.Var(&o.interval, "interval", "time between redraws")
		},
		run: runs,
	},
	"show": {
		setup: func(fs *flag.FlagSet, o *options) {
			fs.BoolVar(&o.json, "json", false, "print JSON")
			fs.StringVar(&o.runID, "run", "", "run id")
			fs.BoolVar(&o.watch, "watch", false, "redraw until interrupted")
			fs.Var(&o.interval, "interval", "time between redraws")
		},
		run: show,
	},
	"wait": {
		setup: func(fs *flag.FlagSet, o *options) {
			fs.BoolVar(&o.json, "json", false, "print JSON")
			fs.StringVar(&o.runID, "run", "", "run id")
			fs.Var(&o.timeout, "timeout", "longest to wait")
			fs.Var(&o.interval, "interval", "time between evaluations")
		},
		run: wait,
	},
}

func Names() []string {
	names := make([]string, 0, len(commands))
	for n := range commands {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func Flags(name string) []string {
	cmd, ok := commands[name]
	if !ok {
		return nil
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	cmd.setup(fs, &options{})
	var out []string
	fs.VisitAll(func(f *flag.Flag) { out = append(out, f.Name) })
	return out
}

func parse(name string, args []string, env Env) (*options, []string, error) {
	cmd := commands[name]
	o := &options{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	cmd.setup(fs, o)
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, nil, err
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
	return o, positional, nil
}

func Run(args []string, env Env) int {
	if len(args) == 0 {
		args = []string{"runs"}
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(env.Stdout, usage)
		return 0
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(env.Stderr, "wf: unknown command %q\n", args[0])
		return ExitUsage
	}
	o, positional, err := parse(args[0], args[1:], env)
	if err != nil {
		return ExitUsage
	}
	return cmd.run(positional, o, env)
}

func worktreeRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", errors.New("this directory is not inside a git worktree")
	}
	root := strings.TrimSpace(string(out))
	if real, err := filepath.EvalSymlinks(root); err == nil {
		root = real
	}
	return root, nil
}

func findRun(o *options, env Env) (*run.Store, *run.RunState, error) {
	store, err := env.store()
	if err != nil {
		return nil, nil, err
	}
	if o.runID != "" {
		st, err := store.Load(o.runID)
		return store, st, err
	}
	cwd, err := env.cwd()
	if err != nil {
		return nil, nil, err
	}
	root, err := worktreeRoot(cwd)
	if err != nil {
		return nil, nil, err
	}
	st, err := store.ForWorktree(root)
	if err != nil {
		if errors.Is(err, run.ErrNoRun) {
			return nil, nil, fmt.Errorf("there is no run for %s; start one with: wf start <workflow-file>", root)
		}
		return nil, nil, err
	}
	return store, st, nil
}
