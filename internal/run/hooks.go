package run

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const StopGuardLimit = 5

type StopDecision struct {
	Block  bool
	Reason string
}

func (e *Engine) guardPath(id string) string {
	return filepath.Join(e.Store.runDir(id), "stopguard")
}

func (e *Engine) guardBlocks(st *RunState) int {
	data, err := os.ReadFile(e.guardPath(st.ID))
	if err != nil {
		return 0
	}
	var events, blocks int
	if _, err := fmt.Sscanf(string(data), "%d %d", &events, &blocks); err != nil {
		return 0
	}
	if events != len(st.Events) {
		return 0
	}
	return blocks
}

func (e *Engine) setGuard(st *RunState, blocks int) {
	data := fmt.Sprintf("%d %d\n", len(st.Events), blocks)
	_ = os.WriteFile(e.guardPath(st.ID), []byte(data), 0o644)
}

func (e *Engine) DecideStop(st *RunState) StopDecision {
	switch st.State() {
	case StateEnded, StateStopped, StateBroken:
		return StopDecision{}
	case StatePending:
		if e.WaiterLive(st.ID) {
			return StopDecision{}
		}
	}
	blocks := e.guardBlocks(st)
	if blocks >= StopGuardLimit {
		reason := fmt.Sprintf("the stop hook let the agent stop after blocking %d times with nothing new recorded", blocks)
		if err := e.Store.append(st.ID, Event{Type: EventGuardReleased, Step: st.Step, Visit: st.Visit, Reason: reason}); err == nil {
			_ = os.Remove(e.guardPath(st.ID))
		}
		return StopDecision{}
	}
	e.setGuard(st, blocks+1)
	return StopDecision{Block: true, Reason: Describe(st, nil, e.Store.now()).Text()}
}

type ToolCall struct {
	Name    string
	Path    string
	Command string
	Cwd     string
}

var editTools = map[string]bool{"Edit": true, "Write": true, "MultiEdit": true, "NotebookEdit": true}

const stateDirReason = "The workflow state cannot be touched by the agent."

func resolve(p string) string {
	p = filepath.Clean(p)
	rest := ""
	for {
		if real, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(real, rest)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return filepath.Join(p, rest)
		}
		rest = filepath.Join(filepath.Base(p), rest)
		p = parent
	}
}

func inside(path, dir string) bool {
	rel, err := filepath.Rel(resolve(dir), resolve(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func mentions(command, dir string) bool {
	names := []string{dir, resolve(dir), "$" + HomeEnv, "${" + HomeEnv + "}"}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, dir); err == nil && !strings.HasPrefix(rel, "..") {
			names = append(names, "~/"+rel)
		}
	}
	for _, n := range names {
		if n != "" && strings.Contains(command, n) {
			return true
		}
	}
	return false
}

func CheckTool(st *RunState, stateDir string, call ToolCall) (string, bool) {
	if st.Lifecycle == Active {
		if step := st.CurrentStep(); step != nil {
			for _, denied := range step.DenyTools {
				if denied == call.Name {
					return fmt.Sprintf("The %s step does not allow the %s tool.", step.Name, call.Name), true
				}
			}
		}
	}
	if editTools[call.Name] && call.Path != "" {
		path := call.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(call.Cwd, path)
		}
		if inside(path, stateDir) {
			return stateDirReason, true
		}
	}
	if call.Name == "Bash" && mentions(call.Command, stateDir) {
		return stateDirReason, true
	}
	return "", false
}

func StatusLine(st *RunState) string {
	name := st.Workflow.Name
	var line string
	switch st.State() {
	case StateEnded:
		line = fmt.Sprintf("%s . ended", name)
	case StateStopped:
		line = fmt.Sprintf("%s > %s . stopped", name, st.Step)
	default:
		limit := 0
		if step := st.CurrentStep(); step != nil {
			limit = step.VisitLimit()
		}
		line = fmt.Sprintf("%s > %s (%d/%d) . %s", name, st.Step, st.Visit, limit, st.State())
	}
	if NeedsYou(st) {
		line += " . NEEDS YOU"
	}
	return line
}

func SessionContext(st *RunState, now time.Time) string {
	if st.State() == StateEnded {
		return ""
	}
	return Describe(st, nil, now).Text()
}
