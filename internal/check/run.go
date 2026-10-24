package check

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/JustinDFuller/workflow/internal/workflow"
)

const DefaultTimeout = 2 * time.Minute

type Env struct {
	RunID       string
	Step        string
	Worktree    string
	WorkflowDir string
	Entered     time.Time
	Inputs      map[string]string
	Route       map[string]string
}

func (e Env) Vars() []string {
	vars := []string{
		"WF_RUN_ID=" + e.RunID,
		"WF_STEP=" + e.Step,
		"WF_WORKTREE=" + e.Worktree,
		"WF_WORKFLOW_DIR=" + e.WorkflowDir,
		"WF_STEP_ENTERED=" + e.Entered.UTC().Format(time.RFC3339),
	}
	for k, v := range e.Inputs {
		vars = append(vars, "WF_INPUT_"+envName(k)+"="+v)
	}
	for k, v := range e.Route {
		vars = append(vars, "WF_ROUTE_"+envName(k)+"="+v)
	}
	return vars
}

func envName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

type limitedBuffer struct {
	buf bytes.Buffer
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	room := MaxMessage*2 - l.buf.Len()
	if room > 0 {
		if len(p) < room {
			room = len(p)
		}
		l.buf.Write(p[:room])
	}
	return len(p), nil
}

func (l *limitedBuffer) String() string {
	return l.buf.String()
}

type Runner struct {
	Cache *Cache
}

func (r *Runner) Run(ctx context.Context, c *workflow.Check, env Env) Result {
	var key string
	if r.Cache != nil && c.Cache == workflow.CacheCommit {
		fp, err := Fingerprint(ctx, env.Worktree)
		if err == nil {
			key = CacheKey(c, env.Worktree, fp, env.Inputs)
			if hit, ok := r.Cache.Get(key); ok {
				return hit
			}
		}
	}
	result := execute(ctx, c, env)
	if key != "" && result.Cacheable() {
		r.Cache.Put(key, result)
	}
	return result
}

func execute(ctx context.Context, c *workflow.Check, env Env) Result {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	cmd := exec.Command("sh", "-c", c.Run)
	cmd.Dir = env.Worktree
	cmd.Env = append(os.Environ(), env.Vars()...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return Result{Status: Broken, Message: fmt.Sprintf("check %q could not start: %v", c.Name, err)}
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		killGroup(cmd)
		if errors.Is(err, exec.ErrWaitDelay) {
			err = nil
			if !cmd.ProcessState.Success() {
				err = &exec.ExitError{ProcessState: cmd.ProcessState}
			}
		}
		return interpret(c, err, stdout.String(), stderr.String())
	case <-timer.C:
		killGroup(cmd)
		<-done
		return Result{Status: Broken, Message: fmt.Sprintf("check %q timed out after %s", c.Name, workflow.CompactDuration(timeout))}
	case <-ctx.Done():
		killGroup(cmd)
		<-done
		return Result{Status: Broken, Message: fmt.Sprintf("check %q was cancelled", c.Name)}
	}
}

func killGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func interpret(c *workflow.Check, err error, stdout, stderr string) Result {
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return Result{Status: Broken, Message: fmt.Sprintf("check %q failed to run: %v", c.Name, err)}
		}
		code = exit.ExitCode()
	}
	message, artifacts := splitArtifacts(stdout)
	switch code {
	case 0:
		result := Result{Status: Pass, Message: cut(message), Artifacts: artifacts}
		if c.HasOutcomes() {
			return outcome(c, result, message)
		}
		return result
	case 1:
		return Result{Status: Fail, Message: cut(message), Artifacts: artifacts}
	case 2:
		return Result{Status: Pending, Message: cut(message), Artifacts: artifacts}
	default:
		parts := []string{fmt.Sprintf("check %q exited with code %d", c.Name, code)}
		if message != "" {
			parts = append(parts, message)
		}
		if e := strings.TrimSpace(stderr); e != "" {
			parts = append(parts, e)
		}
		return Result{Status: Broken, Message: cut(strings.Join(parts, "\n")), Artifacts: artifacts}
	}
}

func outcome(c *workflow.Check, base Result, text string) Result {
	broken := func(format string, args ...any) Result {
		base.Status = Broken
		base.Message = fmt.Sprintf(format, args...)
		return base
	}
	var payload struct {
		Outcome string            `json:"outcome"`
		Values  map[string]string `json:"values"`
	}
	dec := json.NewDecoder(strings.NewReader(text))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return broken("check %q declares outcomes but did not print a JSON object with an outcome: %v", c.Name, err)
	}
	if dec.More() {
		return broken("check %q printed more than one JSON value", c.Name)
	}
	for _, o := range c.Outcomes {
		if o == payload.Outcome {
			base.Outcome = payload.Outcome
			base.Values = payload.Values
			base.Message = payload.Outcome
			return base
		}
	}
	return broken("check %q reported the outcome %q which it does not declare", c.Name, payload.Outcome)
}
