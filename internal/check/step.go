package check

import (
	"context"

	"github.com/JustinDFuller/workflow/internal/workflow"
)

type Named struct {
	Name string `json:"name"`
	Result
}

type StepResult struct {
	Checks []Named
	Over   bool
}

func (s StepResult) Last() Named {
	return s.Checks[len(s.Checks)-1]
}

func (s StepResult) Result() Result {
	return s.Last().Result
}

type Evaluator interface {
	Run(ctx context.Context, c *workflow.Check, env Env) Result
}

func EvaluateStep(ctx context.Context, w *workflow.Workflow, step *workflow.Step, ev Evaluator, env Env) StepResult {
	var out StepResult
	for _, name := range step.Checks {
		result := ev.Run(ctx, w.Checks[name], env)
		out.Checks = append(out.Checks, Named{Name: name, Result: result})
		if !result.Passed() {
			return out
		}
	}
	out.Over = true
	return out
}
