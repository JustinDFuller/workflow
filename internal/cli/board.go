package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/JustinDFuller/workflow/internal/run"
)

const defaultRedraw = 2 * time.Second

const clearScreen = "\x1b[2J\x1b[H"

func draw(o *options, env Env, render func() (string, error)) int {
	if !o.watch {
		text, err := render()
		if err != nil {
			return env.fail(err)
		}
		env.printf("%s", text)
		return 0
	}
	interval := o.interval.d
	if interval <= 0 {
		interval = defaultRedraw
	}
	ctx := env.context()
	for {
		text, err := render()
		if err != nil {
			return env.fail(err)
		}
		env.printf("%s%s", clearScreen, text)
		select {
		case <-ctx.Done():
			return 0
		case <-time.After(interval):
		}
	}
}

func asJSON(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func runs(args []string, o *options, env Env) int {
	if len(args) != 0 {
		fmt.Fprintln(env.Stderr, "usage: wf runs [--all] [--watch] [--interval d] [--json]")
		return ExitUsage
	}
	store, err := env.store()
	if err != nil {
		return env.fail(err)
	}
	return draw(o, env, func() (string, error) {
		states, err := store.All()
		if err != nil {
			return "", err
		}
		list := run.SummarizeAll(states, o.all, env.now())
		if o.json {
			return asJSON(list)
		}
		return run.RunsText(list), nil
	})
}

func show(args []string, o *options, env Env) int {
	if len(args) != 0 {
		fmt.Fprintln(env.Stderr, "usage: wf show [--run id] [--watch] [--interval d] [--json]")
		return ExitUsage
	}
	store, st, err := findRun(o, env)
	if err != nil {
		return env.fail(err)
	}
	return draw(o, env, func() (string, error) {
		fresh, err := store.Load(st.ID)
		if err != nil {
			return "", err
		}
		detail := run.Show(fresh, env.now())
		if o.json {
			return asJSON(detail)
		}
		return detail.Text(), nil
	})
}
