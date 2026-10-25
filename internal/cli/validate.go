package cli

import (
	"fmt"

	"github.com/JustinDFuller/workflow/internal/workflow"
)

func validate(args []string, _ *options, env Env) int {
	if len(args) != 1 {
		fmt.Fprintln(env.Stderr, "usage: wf validate <workflow-file>")
		return ExitUsage
	}
	_, err := workflow.LoadFile(args[0])
	if err == nil {
		env.printf("%s is valid\n", args[0])
		return 0
	}
	if problems, ok := err.(workflow.Problems); ok {
		for _, p := range problems {
			fmt.Fprintln(env.Stderr, p)
		}
		fmt.Fprintf(env.Stderr, "%d problem(s) in %s\n", len(problems), args[0])
		return ExitUsage
	}
	return env.fail(err)
}
