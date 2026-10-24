package openspecchange

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/JustinDFuller/workflow/internal/workflow"
)

const fakeGh = `#!/usr/bin/env bash
dir="$FAKE_GH"
if [ -f "$dir/denied" ]; then
  echo "HTTP 403: Resource not accessible by integration" >&2
  exit 1
fi
serve() {
  if [ -f "$dir/$1" ]; then
    cat "$dir/$1"
    exit 0
  fi
  echo "fake gh has no $1" >&2
  exit 1
}
case "$1 $2" in
  "pr view")
    if [ ! -f "$dir/pr.json" ]; then
      echo 'no pull requests found for branch "feature"' >&2
      exit 1
    fi
    serve pr.json ;;
  "repo view") serve repo.json ;;
esac
if [ "$1" = "api" ]; then
  for arg in "$@"; do
    case "$arg" in
      */reviews/*/comments*) serve inline.json ;;
      */reviews*) serve reviews.json ;;
      */check-runs*) serve checkruns.json ;;
      */status) serve status.json ;;
    esac
  done
fi
echo "fake gh does not know: $*" >&2
exit 1
`

const fakeOpenspec = `#!/usr/bin/env bash
if [ -f "$FAKE_GH/openspec_fail" ]; then
  cat "$FAKE_GH/openspec_fail"
  exit 1
fi
exit 0
`

type sandbox struct {
	t      *testing.T
	repo   string
	remote string
	fake   string
	bin    string
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.test",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.test",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newSandbox(t *testing.T) *sandbox {
	t.Helper()
	root := t.TempDir()
	s := &sandbox{
		t:      t,
		repo:   filepath.Join(root, "repo"),
		remote: filepath.Join(root, "remote.git"),
		fake:   filepath.Join(root, "fake"),
		bin:    filepath.Join(root, "bin"),
	}
	for _, d := range []string{s.repo, s.remote, s.fake, s.bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	s.write(filepath.Join(s.bin, "gh"), fakeGh, 0o755)
	s.write(filepath.Join(s.bin, "openspec"), fakeOpenspec, 0o755)
	git(t, s.remote, "init", "-q", "--bare", "-b", "main")
	git(t, s.repo, "init", "-q", "-b", "main")
	git(t, s.repo, "remote", "add", "origin", s.remote)
	s.write(filepath.Join(s.repo, "README.md"), "hello\n", 0o644)
	git(t, s.repo, "add", ".")
	git(t, s.repo, "commit", "-q", "-m", "first")
	git(t, s.repo, "push", "-q", "-u", "origin", "main")
	git(t, s.repo, "remote", "set-head", "origin", "main")
	git(t, s.repo, "checkout", "-q", "-b", "feature")
	return s
}

func (s *sandbox) write(path, content string, mode os.FileMode) {
	s.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		s.t.Fatal(err)
	}
}

func (s *sandbox) file(rel, content string) {
	s.t.Helper()
	s.write(filepath.Join(s.repo, rel), content, 0o644)
}

func (s *sandbox) fakeFile(name, content string) {
	s.t.Helper()
	s.write(filepath.Join(s.fake, name), content, 0o644)
}

func (s *sandbox) fakeJSON(name string, v any) {
	s.t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		s.t.Fatal(err)
	}
	s.fakeFile(name, string(data))
}

func (s *sandbox) commit(msg string) {
	s.t.Helper()
	git(s.t, s.repo, "add", "-A")
	git(s.t, s.repo, "commit", "-q", "-m", msg)
}

func (s *sandbox) push() {
	s.t.Helper()
	git(s.t, s.repo, "push", "-q", "-u", "origin", "HEAD")
}

func (s *sandbox) change(name string) {
	s.t.Helper()
	s.file("openspec/changes/"+name+"/proposal.md", "why\n")
	s.commit("add " + name)
}

func scriptPath(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("checks", name))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func (s *sandbox) run(script string, extra ...string) (int, string) {
	s.t.Helper()
	return s.runArgs(script, nil, extra...)
}

func (s *sandbox) runArgs(script string, args []string, extra ...string) (int, string) {
	s.t.Helper()
	cmd := exec.Command(scriptPath(s.t, script), args...)
	cmd.Dir = s.repo
	cmd.Env = append(os.Environ(),
		"PATH="+s.bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_GH="+s.fake,
		"WF_WORKFLOW_DIR="+filepath.Dir(scriptPath(s.t, script)),
		"WF_STEP_ENTERED=2026-01-01T00:00:00Z",
		"WF_INPUT_CHANGE=",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	cmd.Env = append(cmd.Env, extra...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			s.t.Fatalf("%s: %v", script, err)
		}
		code = exit.ExitCode()
	}
	out := stdout.String()
	if stderr.Len() > 0 {
		out += "[stderr] " + stderr.String()
	}
	return code, out
}

func (s *sandbox) expect(script string, wantCode int, wantText string, extra ...string) string {
	s.t.Helper()
	return s.expectArgs(script, nil, wantCode, wantText, extra...)
}

func (s *sandbox) expectArgs(script string, args []string, wantCode int, wantText string, extra ...string) string {
	s.t.Helper()
	code, out := s.runArgs(script, args, extra...)
	if code != wantCode || !strings.Contains(out, wantText) {
		s.t.Fatalf("%s: exit %d, want %d containing %q:\n%s", script, code, wantCode, wantText, out)
	}
	return out
}

func prJSON(overrides map[string]any) map[string]any {
	pr := map[string]any{
		"number": 12, "url": "https://example.test/pull/12", "title": "feat: Proposal for the thing",
		"state": "OPEN", "isDraft": false, "baseRefName": "main", "headRefName": "feature", "reviewDecision": "",
	}
	for k, v := range overrides {
		pr[k] = v
	}
	return pr
}

func review(id int, login, state, at, body string) map[string]any {
	return map[string]any{"id": id, "user": map[string]any{"login": login}, "state": state, "submitted_at": at, "body": body}
}

var writeCommands = []*regexp.Regexp{
	regexp.MustCompile(`\bgit (commit|push|add|checkout|switch|reset|merge|rebase|fetch|pull|stash|tag|mv|rm|clean|restore|cherry-pick|apply|am|init|clone|config|remote (add|set-url|remove)|branch +-[dDmMcC]|worktree (add|remove|prune))(\s|$)`),
	regexp.MustCompile(`\bgh (pr (create|edit|merge|ready|close|reopen|comment|review)|issue|release|repo (create|edit|delete)|stack)\b`),
	regexp.MustCompile(`\bgh api\b.*(--method|-X |--field|-f |-F |--raw-field|--input)`),
	regexp.MustCompile(`\bopenspec (archive|new|init|update|sync)\b`),
}

func TestEveryScriptIsExecutableAndReadOnly(t *testing.T) {
	entries, err := os.ReadDir("checks")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("%s is not executable", e.Name())
		}
		data, err := os.ReadFile(filepath.Join("checks", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			for _, w := range writeCommands {
				if w.MatchString(line) {
					t.Errorf("%s:%d changes something: %s", e.Name(), i+1, line)
				}
			}
		}
	}
}

func TestChangePresent(t *testing.T) {
	s := newSandbox(t)
	s.expect("change-present.sh", 1, "no OpenSpec change is present")

	s.file("openspec/changes/add-thing/proposal.md", "why\n")
	out := s.expect("change-present.sh", 0, "the change add-thing is present")
	if !strings.Contains(out, "artifact: change openspec/changes/add-thing\n") {
		t.Fatalf("missing artifact line:\n%s", out)
	}

	s.commit("add change")
	s.expect("change-present.sh", 0, "artifact: change openspec/changes/add-thing")

	s.change("other-thing")
	out = s.expect("change-present.sh", 1, "2 OpenSpec changes differ")
	if !strings.Contains(out, "--input change=<name>") {
		t.Fatalf("does not say how to choose:\n%s", out)
	}
	s.expect("change-present.sh", 0, "artifact: change openspec/changes/other-thing", "WF_INPUT_CHANGE=other-thing")
	s.expect("change-present.sh", 1, "has no directory", "WF_INPUT_CHANGE=missing")
}

func TestChangePresentSeesArchivedChange(t *testing.T) {
	s := newSandbox(t)
	s.file("openspec/changes/archive/2026-02-03-add-thing/proposal.md", "why\n")
	s.commit("archive")
	s.expect("change-present.sh", 0, "artifact: change openspec/changes/archive/2026-02-03-add-thing")

	s.file("openspec/changes/add-thing/proposal.md", "why\n")
	s.commit("again")
	s.expect("change-present.sh", 0, "artifact: change openspec/changes/add-thing\n")
}

func TestChangePresentNeedsADefaultBranchBase(t *testing.T) {
	s := newSandbox(t)
	git(t, s.repo, "remote", "remove", "origin")
	s.fakeJSON("repo.json", map[string]any{"defaultBranchRef": map[string]any{"name": "main"}})
	s.change("add-thing")
	s.expect("change-present.sh", 0, "the change add-thing is present")
}

func TestChangeValid(t *testing.T) {
	s := newSandbox(t)
	s.change("add-thing")
	s.expect("change-valid.sh", 0, "passes openspec validate --strict")
	s.fakeFile("openspec_fail", "proposal.md: missing Why section\n")
	out := s.expect("change-valid.sh", 1, "fails for the change add-thing")
	if !strings.Contains(out, "missing Why section") {
		t.Fatalf("validate output not shown:\n%s", out)
	}
}

func TestChangeValidNeedsAChange(t *testing.T) {
	s := newSandbox(t)
	s.expect("change-valid.sh", 1, "no OpenSpec change is present")
}

func TestPushed(t *testing.T) {
	s := newSandbox(t)
	s.change("add-thing")
	s.expect("pushed.sh", 1, "has not been pushed")

	s.push()
	s.expect("pushed.sh", 0, "is pushed at")

	s.file("openspec/changes/add-thing/design.md", "how\n")
	s.expect("pushed.sh", 1, "uncommitted changes under openspec/changes")

	s.commit("design")
	s.expect("pushed.sh", 1, "1 commit(s) are not on origin/feature")

	s.push()
	s.expect("pushed.sh", 0, "is pushed at")

	git(t, s.repo, "checkout", "-q", "--detach")
	s.expect("pushed.sh", 1, "HEAD is detached")
}

func TestPushedNewCommitSinceEntry(t *testing.T) {
	s := newSandbox(t)
	s.change("add-thing")
	s.push()
	s.expectArgs("pushed.sh", []string{"--new-commit"}, 1, "no new commit has been pushed since this step began", "WF_STEP_ENTERED=2099-01-01T00:00:00Z")
	s.expectArgs("pushed.sh", []string{"--new-commit"}, 0, "is pushed at", "WF_STEP_ENTERED=2020-01-01T00:00:00Z")

	s.file("openspec/changes/add-thing/design.md", "how\n")
	s.commit("design")
	s.expectArgs("pushed.sh", []string{"--new-commit"}, 1, "not pushed", "WF_STEP_ENTERED=2020-01-01T00:00:00Z")
}

func TestProposalPR(t *testing.T) {
	s := newSandbox(t)
	s.expect("proposal-pr.sh", 1, "no pull request exists for the branch feature")

	s.fakeJSON("pr.json", prJSON(map[string]any{"state": "CLOSED"}))
	s.expect("proposal-pr.sh", 1, "is CLOSED, not open")

	s.fakeJSON("pr.json", prJSON(map[string]any{"title": "feat: Implementation for the thing"}))
	out := s.expect("proposal-pr.sh", 1, "must start with Proposal")
	if strings.Contains(out, "artifact:") {
		t.Fatalf("a wrong title reported the artifact:\n%s", out)
	}

	for _, title := range []string{"Proposal for the thing", "feat: Proposal for the thing", "feat(cli)!: Proposal for the thing"} {
		s.fakeJSON("pr.json", prJSON(map[string]any{"title": title, "isDraft": true}))
		out = s.expect("proposal-pr.sh", 0, "pull request #12 is open")
		if !strings.Contains(out, "artifact: proposal_pr https://example.test/pull/12\n") {
			t.Fatalf("%q: missing artifact line:\n%s", title, out)
		}
	}

	s.fakeFile("denied", "")
	s.expect("proposal-pr.sh", 3, "HTTP 403")
}

func TestCIGreen(t *testing.T) {
	s := newSandbox(t)
	s.fakeJSON("status.json", map[string]any{"statuses": []any{}})

	s.fakeJSON("checkruns.json", map[string]any{"check_runs": []any{}})
	s.expect("ci-green.sh", 2, "has not reported any checks")

	runs := func(items ...map[string]any) {
		s.fakeJSON("checkruns.json", map[string]any{"check_runs": items})
	}
	run := func(name, status, conclusion string) map[string]any {
		return map[string]any{"name": name, "status": status, "conclusion": conclusion}
	}

	runs(run("go", "completed", "success"), run("lint", "completed", "skipped"), run("docs", "completed", "neutral"))
	s.expect("ci-green.sh", 0, "CI is green at")

	runs(run("go", "completed", "success"), run("lint", "in_progress", ""), run("docs", "queued", ""))
	s.expect("ci-green.sh", 2, "CI is still running at")
	out := s.expect("ci-green.sh", 2, "lint, docs")
	if strings.Contains(out, "go,") {
		t.Fatalf("a passing check was named as running:\n%s", out)
	}

	runs(run("go", "completed", "failure"), run("lint", "in_progress", ""))
	s.expect("ci-green.sh", 1, "CI is failing at")

	runs(run("go", "completed", "cancelled"))
	s.expect("ci-green.sh", 1, "go")

	runs(run("go", "completed", "success"))
	s.fakeJSON("status.json", map[string]any{"statuses": []any{map[string]any{"context": "CodeRabbit", "state": "pending"}}})
	s.expect("ci-green.sh", 2, "CodeRabbit")
	s.fakeJSON("status.json", map[string]any{"statuses": []any{map[string]any{"context": "CodeRabbit", "state": "failure"}}})
	s.expect("ci-green.sh", 1, "CodeRabbit")
	s.fakeJSON("status.json", map[string]any{"statuses": []any{map[string]any{"context": "CodeRabbit", "state": "success"}}})
	s.expect("ci-green.sh", 0, "(2 checks)")
}

func TestPRReady(t *testing.T) {
	s := newSandbox(t)
	s.fakeJSON("pr.json", prJSON(map[string]any{"isDraft": true}))
	s.expect("pr-ready.sh", 1, "still a draft")
	s.fakeJSON("pr.json", prJSON(nil))
	s.expect("pr-ready.sh", 0, "is ready for review")
}

func outcomeOf(t *testing.T, out string) (string, map[string]string) {
	t.Helper()
	var payload struct {
		Outcome string            `json:"outcome"`
		Values  map[string]string `json:"values"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("not an outcome object: %v\n%s", err, out)
	}
	return payload.Outcome, payload.Values
}

func TestProposalReview(t *testing.T) {
	s := newSandbox(t)
	s.fakeJSON("pr.json", prJSON(nil))
	entered := "WF_STEP_ENTERED=2026-03-01T12:00:00Z"

	s.fakeJSON("reviews.json", []any{})
	s.expect("proposal-review.sh", 2, "no approving or change-requesting review", entered)

	s.fakeJSON("reviews.json", []any{review(1, "pat", "APPROVED", "2026-03-01T11:59:59Z", "")})
	s.expect("proposal-review.sh", 2, "since 2026-03-01T12:00:00Z", entered)

	s.fakeJSON("reviews.json", []any{review(1, "pat", "COMMENTED", "2026-03-01T13:00:00Z", "looks fine")})
	s.expect("proposal-review.sh", 2, "no approving", entered)

	s.fakeJSON("reviews.json", []any{review(1, "pat", "DISMISSED", "2026-03-01T13:00:00Z", "")})
	s.expect("proposal-review.sh", 2, "no approving", entered)

	s.fakeJSON("reviews.json", []any{review(1, "pat", "APPROVED", "2026-03-01T13:00:00Z", "")})
	_, out := s.run("proposal-review.sh", entered)
	if name, values := outcomeOf(t, out); name != "approved" || len(values) != 0 {
		t.Fatalf("%s %v", name, values)
	}

	s.fakeJSON("reviews.json", []any{
		review(1, "pat", "APPROVED", "2026-03-01T12:30:00Z", ""),
		review(2, "sam", "CHANGES_REQUESTED", "2026-03-01T13:00:00Z", "Please split the design."),
	})
	s.fakeJSON("inline.json", []any{map[string]any{"path": "design.md", "line": 14, "body": "This is unclear."}, map[string]any{"path": "tasks.md", "body": "Missing a task."}})
	code, out := s.run("proposal-review.sh", entered)
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	name, values := outcomeOf(t, out)
	if name != "changes_requested" {
		t.Fatal(name)
	}
	for _, want := range []string{"sam", "Please split the design.", "design.md:14: This is unclear.", "tasks.md: Missing a task.", "https://example.test/pull/12"} {
		if !strings.Contains(values["comments"], want) {
			t.Errorf("comments lack %q:\n%s", want, values["comments"])
		}
	}

	s.fakeJSON("reviews.json", []any{
		review(2, "sam", "CHANGES_REQUESTED", "2026-03-01T13:00:00Z", "No."),
		review(3, "sam", "APPROVED", "2026-03-01T14:00:00Z", ""),
	})
	_, out = s.run("proposal-review.sh", entered)
	if name, _ := outcomeOf(t, out); name != "approved" {
		t.Fatalf("a later approval should win: %s", name)
	}

	s.fakeJSON("pr.json", prJSON(map[string]any{"state": "CLOSED"}))
	_, out = s.run("proposal-review.sh", entered)
	if name, _ := outcomeOf(t, out); name != "closed" {
		t.Fatalf("closed: %s", name)
	}

	s.fakeJSON("pr.json", prJSON(map[string]any{"state": "MERGED"}))
	_, out = s.run("proposal-review.sh", entered)
	if name, _ := outcomeOf(t, out); name != "approved" {
		t.Fatalf("merged: %s", name)
	}

	s.fakeFile("denied", "")
	s.expect("proposal-review.sh", 3, "Resource not accessible", entered)
}

func TestInvariants(t *testing.T) {
	s := newSandbox(t)
	s.expect("branch-not-default.sh", 0, "not on the default branch main")
	git(t, s.repo, "checkout", "-q", "main")
	s.expect("branch-not-default.sh", 1, "Invariant failed, the work must be on a branch other than the default branch")

	s.fakeJSON("pr.json", prJSON(map[string]any{"reviewDecision": "APPROVED"}))
	s.expect("proposal-approved.sh", 0, "is approved")
	s.fakeJSON("pr.json", prJSON(map[string]any{"reviewDecision": "CHANGES_REQUESTED"}))
	s.expect("proposal-approved.sh", 1, "Invariant failed, the proposal pull request must be approved")
	s.fakeJSON("pr.json", prJSON(map[string]any{"reviewDecision": "REVIEW_REQUIRED"}))
	s.expect("proposal-approved.sh", 1, "still needs a review")

	s.fakeJSON("pr.json", prJSON(nil))
	s.fakeJSON("reviews.json", []any{})
	s.expect("proposal-approved.sh", 1, "has no approving review")
	s.fakeJSON("reviews.json", []any{review(1, "pat", "APPROVED", "2026-03-01T13:00:00Z", "")})
	s.expect("proposal-approved.sh", 0, "is approved")
	s.fakeJSON("reviews.json", []any{review(1, "pat", "APPROVED", "2026-03-01T13:00:00Z", ""), review(2, "sam", "CHANGES_REQUESTED", "2026-03-01T14:00:00Z", "")})
	s.expect("proposal-approved.sh", 1, "has no approving review")

	s.fakeJSON("status.json", map[string]any{"statuses": []any{}})
	s.fakeJSON("checkruns.json", map[string]any{"check_runs": []any{map[string]any{"name": "go", "status": "completed", "conclusion": "failure"}}})
	s.expect("ci-still-green.sh", 1, "Invariant failed, CI must be green: CI is failing")
	s.fakeJSON("checkruns.json", map[string]any{"check_runs": []any{map[string]any{"name": "go", "status": "in_progress"}}})
	s.expect("ci-still-green.sh", 2, "CI is still running")
	s.fakeJSON("checkruns.json", map[string]any{"check_runs": []any{map[string]any{"name": "go", "status": "completed", "conclusion": "success"}}})
	s.expect("ci-still-green.sh", 0, "CI is green")
}

func TestDeniedAccessIsBrokenWithTheReason(t *testing.T) {
	s := newSandbox(t)
	s.fakeFile("denied", "")
	for _, script := range []string{"proposal-pr.sh", "pr-ready.sh", "ci-green.sh", "proposal-approved.sh", "proposal-review.sh", "ci-still-green.sh"} {
		s.fakeJSON("pr.json", prJSON(nil))
		code, out := s.run(script)
		if code != 3 || !strings.Contains(out, "GitHub did not allow or could not finish") || !strings.Contains(out, "HTTP 403") {
			t.Errorf("%s: exit %d\n%s", script, code, out)
		}
	}
}

func TestMissingToolIsBroken(t *testing.T) {
	s := newSandbox(t)
	only := t.TempDir()
	dirname, err := exec.LookPath("dirname")
	if err != nil {
		t.Skip("dirname is not installed")
	}
	if err := os.Symlink(dirname, filepath.Join(only, "dirname")); err != nil {
		t.Fatal(err)
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	cmd := exec.Command(bash, scriptPath(t, "ci-green.sh"))
	cmd.Dir = s.repo
	cmd.Env = []string{"PATH=" + only}
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 3 || !strings.Contains(string(out), "git is not installed or is not on the PATH") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestWorkflowFileIsValidAndUsesEveryScript(t *testing.T) {
	w, err := workflow.LoadFile("workflow.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if problems := workflow.Validate(w); len(problems) > 0 {
		t.Fatal(problems)
	}
	used := map[string]bool{"lib.sh": true}
	for name, c := range w.Checks {
		start := strings.Index(c.Run, "checks/")
		if start < 0 {
			t.Fatalf("check %s does not run a script in checks/: %s", name, c.Run)
		}
		script := strings.Fields(c.Run[start+len("checks/"):])[0]
		script = strings.TrimSuffix(script, `"`)
		used[script] = true
		if _, err := os.Stat(filepath.Join("checks", script)); err != nil {
			t.Errorf("check %s: %v", name, err)
		}
	}
	entries, err := os.ReadDir("checks")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !used[e.Name()] {
			t.Errorf("%s is not used by any check", e.Name())
		}
	}
}

func TestProposeStepOrdersCheapChecksFirst(t *testing.T) {
	w, err := workflow.LoadFile("workflow.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"change-present", "change-valid", "pushed", "proposal-pr", "ci-green", "pr-ready"}
	if got := w.Steps["propose"].Checks; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%v", got)
	}
	review := w.Steps["proposal_review"]
	if !review.NeedsYou || review.Timeout != 14*24*time.Hour || review.HasJob() {
		t.Fatalf("%+v", review)
	}
	if review.On["approved"] != "invariants" || review.On["changes_requested"] != "revise" || review.On["closed"] != workflow.TargetStop {
		t.Fatalf("%v", review.On)
	}
	if w.Steps["revise"].Then != "proposal_review" {
		t.Fatal("revise must return to the review")
	}
	if got := strings.Join(w.Steps["invariants"].Checks, ","); got != "branch-not-default,proposal-approved,ci-still-green" || w.Steps["invariants"].HasJob() {
		t.Fatalf("%s", got)
	}
}
