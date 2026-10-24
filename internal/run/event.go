package run

import (
	"time"

	"github.com/JustinDFuller/workflow/internal/check"
)

const (
	EventStarted     = "started"
	EventStepEntered = "step_entered"
	EventCheckResult = "check_result"
	EventStepPassed  = "step_passed"
	EventArtifact    = "artifact"
	EventStopped     = "stopped"
	EventEnded       = "ended"
)

type Event struct {
	Time      time.Time         `json:"time"`
	Type      string            `json:"type"`
	Step      string            `json:"step,omitempty"`
	Visit     int               `json:"visit,omitempty"`
	Check     string            `json:"check,omitempty"`
	Status    check.Status      `json:"status,omitempty"`
	Message   string            `json:"message,omitempty"`
	Outcome   string            `json:"outcome,omitempty"`
	Values    map[string]string `json:"values,omitempty"`
	Artifacts []check.Artifact  `json:"artifacts,omitempty"`
	Name      string            `json:"name,omitempty"`
	Address   string            `json:"address,omitempty"`
	Target    string            `json:"target,omitempty"`
	From      string            `json:"from,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	Workflow  string            `json:"workflow,omitempty"`
	YAML      string            `json:"yaml,omitempty"`
	File      string            `json:"file,omitempty"`
	Dir       string            `json:"dir,omitempty"`
	Worktree  string            `json:"worktree,omitempty"`
	Inputs    map[string]string `json:"inputs,omitempty"`
}
