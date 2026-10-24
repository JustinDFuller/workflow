package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JustinDFuller/workflow/internal/workflow"
)

func runCheck(t *testing.T, c *workflow.Check, dir string) Result {
	t.Helper()
	r := &Runner{}
	return r.Run(context.Background(), c, Env{Worktree: dir, WorkflowDir: dir, RunID: "r1", Step: "s", Entered: time.Unix(0, 0)})
}

func TestExitCodes(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		run     string
		status  Status
		message string
	}{
		{"exit 0", Pass, ""},
		{`echo "3 tasks remain"; exit 1`, Fail, "3 tasks remain"},
		{`echo "checks are running"; exit 2`, Pending, "checks are running"},
		{`echo out; echo oops >&2; exit 7`, Broken, "oops"},
	}
	for _, tc := range cases {
		got := runCheck(t, &workflow.Check{Name: "c", Run: tc.run}, dir)
		if got.Status != tc.status || !strings.Contains(got.Message, tc.message) {
			t.Errorf("%q: %+v", tc.run, got)
		}
	}
}

func TestBrokenCarriesStdoutAndStderr(t *testing.T) {
	got := runCheck(t, &workflow.Check{Name: "c", Run: `echo said; echo err >&2; exit 9`}, t.TempDir())
	if got.Status != Broken || !strings.Contains(got.Message, "said") || !strings.Contains(got.Message, "err") || !strings.Contains(got.Message, "code 9") {
		t.Fatalf("%+v", got)
	}
}

func TestStdoutOnlyForNonBroken(t *testing.T) {
	got := runCheck(t, &workflow.Check{Name: "c", Run: `echo shown; echo hidden >&2; exit 1`}, t.TempDir())
	if got.Message != "shown" {
		t.Fatalf("%q", got.Message)
	}
}

func TestEnvironmentAndWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	real, _ := filepath.EvalSymlinks(dir)
	env := Env{
		RunID: "run-9", Step: "apply", Worktree: dir, WorkflowDir: "/wfdir",
		Entered: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Inputs:  map[string]string{"change-name": "add-x"},
		Route:   map[string]string{"comments": "fix it"},
	}
	c := &workflow.Check{Name: "c", Run: `pwd; echo $WF_RUN_ID $WF_STEP $WF_WORKTREE $WF_WORKFLOW_DIR $WF_STEP_ENTERED; echo $WF_INPUT_CHANGE_NAME; echo $WF_ROUTE_COMMENTS`}
	got := (&Runner{}).Run(context.Background(), c, env)
	lines := strings.Split(got.Message, "\n")
	if len(lines) != 4 {
		t.Fatalf("%q", got.Message)
	}
	if gotDir, _ := filepath.EvalSymlinks(lines[0]); gotDir != real {
		t.Errorf("cwd %q want %q", lines[0], real)
	}
	want := "run-9 apply " + dir + " /wfdir 2026-01-02T03:04:05Z"
	if lines[1] != want {
		t.Errorf("env %q want %q", lines[1], want)
	}
	if lines[2] != "add-x" || lines[3] != "fix it" {
		t.Errorf("inputs/route %q", lines[2:])
	}
}

func TestOutcomes(t *testing.T) {
	c := &workflow.Check{Name: "review", Outcomes: []string{"approved", "changes_requested"}}
	dir := t.TempDir()

	c.Run = `echo '{"outcome":"changes_requested","values":{"comments":"rename it"}}'`
	got := runCheck(t, c, dir)
	if got.Status != Pass || got.Outcome != "changes_requested" || got.Values["comments"] != "rename it" {
		t.Fatalf("with values: %+v", got)
	}

	c.Run = `echo '{"outcome":"maybe"}'`
	got = runCheck(t, c, dir)
	if got.Status != Broken || !strings.Contains(got.Message, `"maybe"`) {
		t.Fatalf("undeclared: %+v", got)
	}

	c.Run = `echo not json`
	if got = runCheck(t, c, dir); got.Status != Broken {
		t.Fatalf("not json: %+v", got)
	}

	c.Run = `echo '{"outcome":"approved"} {"outcome":"approved"}'`
	if got = runCheck(t, c, dir); got.Status != Broken || !strings.Contains(got.Message, "more than one") {
		t.Fatalf("two values: %+v", got)
	}

	c.Run = `echo '{"outcome":"approved","other":1}'`
	if got = runCheck(t, c, dir); got.Status != Broken {
		t.Fatalf("unknown field: %+v", got)
	}

	c.Run = `echo waiting; exit 2`
	if got = runCheck(t, c, dir); got.Status != Pending {
		t.Fatalf("pending keeps meaning: %+v", got)
	}
	c.Run = `echo bad; exit 1`
	if got = runCheck(t, c, dir); got.Status != Fail {
		t.Fatalf("fail keeps meaning: %+v", got)
	}
}

func TestEvaluateStepStopsAtFirstFailure(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "third-ran")
	w := &workflow.Workflow{Checks: map[string]*workflow.Check{
		"one":   {Name: "one", Run: "exit 0"},
		"two":   {Name: "two", Run: "echo nope; exit 1"},
		"three": {Name: "three", Run: "touch " + marker},
	}}
	step := &workflow.Step{Checks: []string{"one", "two", "three"}}
	res := EvaluateStep(context.Background(), w, step, &Runner{}, Env{Worktree: dir})
	if res.Over || len(res.Checks) != 2 || res.Last().Name != "two" || res.Result().Message != "nope" {
		t.Fatalf("%+v", res)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("third check ran")
	}
	step.Checks = []string{"one", "three"}
	res = EvaluateStep(context.Background(), w, step, &Runner{}, Env{Worktree: dir})
	if !res.Over || len(res.Checks) != 2 {
		t.Fatalf("all pass: %+v", res)
	}
}

func TestTimeoutKillsProcessGroup(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	c := &workflow.Check{Name: "slow", Timeout: 300 * time.Millisecond, Run: `sleep 60 & echo $! > ` + pidFile + `; wait`}
	start := time.Now()
	got := runCheck(t, c, dir)
	if got.Status != Broken || !strings.Contains(got.Message, "timed out") {
		t.Fatalf("%+v", got)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("did not stop promptly")
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := exec.Command("kill", "-0", strconv.Itoa(pid)).Run(); err != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("child %d still alive", pid)
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	got := (&Runner{}).Run(ctx, &workflow.Check{Name: "c", Run: "sleep 30"}, Env{Worktree: t.TempDir()})
	if got.Status != Broken || !strings.Contains(got.Message, "cancelled") {
		t.Fatalf("%+v", got)
	}
}

func TestOutputLimit(t *testing.T) {
	got := runCheck(t, &workflow.Check{Name: "c", Run: `head -c 200000 /dev/zero | tr '\0' x; exit 1`}, t.TempDir())
	if len(got.Message) != MaxMessage+len(CutMarker) || !strings.HasSuffix(got.Message, CutMarker) {
		t.Fatalf("len %d", len(got.Message))
	}
	short := runCheck(t, &workflow.Check{Name: "c", Run: `echo hi; exit 1`}, t.TempDir())
	if strings.Contains(short.Message, "cut") {
		t.Fatal("short output cut")
	}
}

func TestStartFailure(t *testing.T) {
	got := (&Runner{}).Run(context.Background(), &workflow.Check{Name: "c", Run: "true"}, Env{Worktree: "/definitely/not/here"})
	if got.Status != Broken || !strings.Contains(got.Message, "could not start") {
		t.Fatalf("%+v", got)
	}
}

func TestArtifacts(t *testing.T) {
	c := &workflow.Check{Name: "c", Run: `echo "artifact: proposal_pr https://example.test/pull/1"; echo "artifact: change openspec/changes/add-x"; echo "artifact: bad /etc/passwd"; echo "artifact: up ../x"; echo "artifact: toomany a b"; echo text`}
	got := runCheck(t, c, t.TempDir())
	if got.Status != Pass || len(got.Artifacts) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Artifacts[0] != (Artifact{"proposal_pr", "https://example.test/pull/1"}) || got.Artifacts[1] != (Artifact{"change", "openspec/changes/add-x"}) {
		t.Fatalf("%+v", got.Artifacts)
	}
	if !strings.Contains(got.Message, "artifact: bad /etc/passwd") || !strings.HasSuffix(got.Message, "text") {
		t.Fatalf("message %q", got.Message)
	}
	c.Run = `echo "artifact: change openspec/changes/add-x"; exit 1`
	if got = runCheck(t, c, t.TempDir()); got.Status != Fail || len(got.Artifacts) != 1 {
		t.Fatalf("artifact with fail: %+v", got)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@example.test"},
		{"config", "user.name", "T"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func countingCheck(dir, body string) *workflow.Check {
	return &workflow.Check{Name: "judge", Cache: workflow.CacheCommit, Run: `echo x >> ` + filepath.Join(dir, "..", filepath.Base(dir)+".count") + `; ` + body}
}

func countRuns(t *testing.T, dir string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "..", filepath.Base(dir)+".count"))
	if err != nil {
		return 0
	}
	return strings.Count(string(data), "x")
}

func TestCache(t *testing.T) {
	repo := initRepo(t)
	r := &Runner{Cache: &Cache{Dir: t.TempDir()}}
	c := countingCheck(repo, "echo ok")
	env := Env{Worktree: repo}

	if got := r.Run(context.Background(), c, env); got.Status != Pass {
		t.Fatalf("%+v", got)
	}
	r.Run(context.Background(), c, env)
	if n := countRuns(t, repo); n != 1 {
		t.Fatalf("reused: ran %d times", n)
	}

	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "second")
	r.Run(context.Background(), c, env)
	if n := countRuns(t, repo); n != 2 {
		t.Fatalf("new commit: ran %d times", n)
	}

	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.Run(context.Background(), c, env)
	if n := countRuns(t, repo); n != 3 {
		t.Fatalf("uncommitted new file: ran %d times", n)
	}
	r.Run(context.Background(), c, env)
	if n := countRuns(t, repo); n != 3 {
		t.Fatalf("unchanged uncommitted: ran %d times", n)
	}

	gitIn(t, repo, "add", "file.txt")
	gitIn(t, repo, "commit", "-q", "-m", "add file")
	r.Run(context.Background(), c, env)
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := countRuns(t, repo)
	r.Run(context.Background(), c, env)
	if n := countRuns(t, repo); n != before+1 {
		t.Fatalf("edited tracked file: ran %d times, before %d", n, before)
	}
}

func TestCacheSkipsPendingAndBroken(t *testing.T) {
	repo := initRepo(t)
	r := &Runner{Cache: &Cache{Dir: t.TempDir()}}
	env := Env{Worktree: repo}
	for _, body := range []string{"exit 2", "exit 9"} {
		dir := initRepo(t)
		c := countingCheck(dir, body)
		r.Run(context.Background(), c, Env{Worktree: dir})
		r.Run(context.Background(), c, Env{Worktree: dir})
		if n := countRuns(t, dir); n != 2 {
			t.Fatalf("%s: ran %d times", body, n)
		}
	}
	_ = env
}

func TestCacheStoresFailAndArtifacts(t *testing.T) {
	repo := initRepo(t)
	r := &Runner{Cache: &Cache{Dir: t.TempDir()}}
	c := countingCheck(repo, `echo "artifact: n a/b"; echo reason; exit 1`)
	first := r.Run(context.Background(), c, Env{Worktree: repo})
	second := r.Run(context.Background(), c, Env{Worktree: repo})
	if countRuns(t, repo) != 1 || second.Status != Fail || second.Message != "reason" || len(second.Artifacts) != 1 || first.Message != second.Message {
		t.Fatalf("%+v %+v", first, second)
	}
}

func TestCacheNotUsedWithoutSetting(t *testing.T) {
	repo := initRepo(t)
	r := &Runner{Cache: &Cache{Dir: t.TempDir()}}
	c := countingCheck(repo, "true")
	c.Cache = ""
	r.Run(context.Background(), c, Env{Worktree: repo})
	r.Run(context.Background(), c, Env{Worktree: repo})
	if n := countRuns(t, repo); n != 2 {
		t.Fatalf("ran %d times", n)
	}
}

func TestCacheOutsideRepoRunsEveryTime(t *testing.T) {
	dir := t.TempDir()
	r := &Runner{Cache: &Cache{Dir: t.TempDir()}}
	c := countingCheck(dir, "true")
	r.Run(context.Background(), c, Env{Worktree: dir})
	r.Run(context.Background(), c, Env{Worktree: dir})
	if n := countRuns(t, dir); n != 2 {
		t.Fatalf("ran %d times", n)
	}
}

func TestCacheCorruptEntryIgnored(t *testing.T) {
	c := &Cache{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(c.Dir, "k.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("k"); ok {
		t.Fatal("corrupt entry used")
	}
	if _, ok := c.Get("absent"); ok {
		t.Fatal("absent entry used")
	}
	blocker := filepath.Join(t.TempDir(), "file")
	os.WriteFile(blocker, nil, 0o644)
	(&Cache{Dir: filepath.Join(blocker, "sub")}).Put("k", Result{Status: Pass})
}

func TestEnvNames(t *testing.T) {
	if got := envName("review-comments.v2"); got != "REVIEW_COMMENTS_V2" {
		t.Fatal(got)
	}
}

func TestBackgroundProcessHoldingOutputDoesNotBreakTheCheck(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	c := &workflow.Check{Name: "bg", Run: `sleep 60 & echo $! > ` + pidFile + `; echo hi`}
	got := runCheck(t, c, dir)
	if got.Status != Pass || got.Message != "hi" {
		t.Fatalf("%+v", got)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := exec.Command("kill", "-0", strconv.Itoa(pid)).Run(); err != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("background process %d still alive", pid)
}

func TestCacheKeyIncludesInputs(t *testing.T) {
	repo := initRepo(t)
	r := &Runner{Cache: &Cache{Dir: t.TempDir()}}
	c := countingCheck(repo, "echo ok")
	r.Run(context.Background(), c, Env{Worktree: repo, Inputs: map[string]string{"change": "one"}})
	r.Run(context.Background(), c, Env{Worktree: repo, Inputs: map[string]string{"change": "one"}})
	if n := countRuns(t, repo); n != 1 {
		t.Fatalf("same inputs: ran %d times", n)
	}
	r.Run(context.Background(), c, Env{Worktree: repo, Inputs: map[string]string{"change": "two"}})
	if n := countRuns(t, repo); n != 2 {
		t.Fatalf("different inputs: ran %d times", n)
	}
}
