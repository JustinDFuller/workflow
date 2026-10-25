package workflow

import (
	"strings"
	"testing"
	"time"
)

func loadFixture(t *testing.T, name string) (*Workflow, Problems) {
	t.Helper()
	w, err := LoadFile("testdata/" + name)
	if err == nil {
		return w, nil
	}
	problems, ok := err.(Problems)
	if !ok {
		t.Fatalf("unexpected error type: %v", err)
	}
	return w, problems
}

func problemText(p Problems) string {
	return p.Error()
}

func TestValidFileLoads(t *testing.T) {
	w, problems := loadFixture(t, "valid.yaml")
	if len(problems) > 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if w.Name != "demo" || w.Start != "write" {
		t.Fatalf("name %q start %q", w.Name, w.Start)
	}
	if got := w.Steps["write"].VisitLimit(); got != 3 {
		t.Fatalf("visit limit %d", got)
	}
	if got := w.Steps["waiting"].VisitLimit(); got != DefaultMaxVisits {
		t.Fatalf("default visit limit %d", got)
	}
	if got := w.Checks["review"].Timeout; got != 30*time.Second {
		t.Fatalf("timeout %v", got)
	}
	if got := w.Steps["waiting"].Timeout; got != 14*24*time.Hour {
		t.Fatalf("step timeout %v", got)
	}
	if !w.Steps["waiting"].NeedsYou || w.Steps["waiting"].HasJob() {
		t.Fatal("waiting step flags")
	}
	if got := w.Steps["write"].DenyTools; len(got) != 1 || got[0] != "WebFetch" {
		t.Fatalf("deny tools %v", got)
	}
}

func TestUnknownKeyNamesKeyFileAndLine(t *testing.T) {
	_, problems := loadFixture(t, "unknown_key.yaml")
	if len(problems) != 1 {
		t.Fatalf("problems: %v", problems)
	}
	p := problems[0]
	if !strings.Contains(p.Message, `"colour"`) || p.Line != 11 || !strings.HasSuffix(p.File, "unknown_key.yaml") {
		t.Fatalf("problem: %+v", p)
	}
}

func TestInputs(t *testing.T) {
	w, _ := loadFixture(t, "valid.yaml")
	topic, ok := w.Input("topic")
	if !ok || !topic.Required {
		t.Fatalf("topic: %+v", topic)
	}
	mode, ok := w.Input("mode")
	if !ok || mode.Required || mode.Default != "fast" {
		t.Fatalf("mode: %+v", mode)
	}
	if _, ok := w.Input("absent"); ok {
		t.Fatal("absent input found")
	}
}

func TestSeveralProblemsAreAllReported(t *testing.T) {
	_, problems := loadFixture(t, "several_problems.yaml")
	text := problemText(problems)
	for _, want := range []string{
		`check "missing" which is not in the check library`,
		`routes to "nowhere"`,
		`step "orphan" cannot be reached`,
		`step "wait" cannot be reached`,
		`step "wait" has no job and no timeout`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	if len(problems) < 5 {
		t.Fatalf("expected at least 5 problems, got %d", len(problems))
	}
}

func loadText(t *testing.T, text string) Problems {
	t.Helper()
	_, err := Load([]byte(text), "inline.yaml", ".")
	if err == nil {
		return nil
	}
	return err.(Problems)
}

const header = `name: demo
start: a
checks:
  plain:
    run: "true"
  outcome:
    run: "true"
    outcomes: [yes, no]
`

func expectProblem(t *testing.T, text, want string) {
	t.Helper()
	problems := loadText(t, header+text)
	if !strings.Contains(problemText(problems), want) {
		t.Fatalf("want %q in:\n%s", want, problemText(problems))
	}
}

func TestRoutingRules(t *testing.T) {
	expectProblem(t, `steps:
  a:
    job: x
    checks: [outcome]
    on:
      yes: end
`, `does not route the outcome "no"`)
	expectProblem(t, `steps:
  a:
    job: x
    checks: [outcome]
    then: end
    on:
      yes: end
      no: end
`, "plain next target and a final check that declares outcomes")
	expectProblem(t, `steps:
  a:
    job: x
    checks: [plain]
    on:
      yes: end
    then: end
`, "maps outcomes but its final check declares none")
	expectProblem(t, `steps:
  a:
    job: x
    checks: [outcome]
    on:
      yes: end
      no: end
      maybe: end
`, `routes outcome "maybe"`)
	expectProblem(t, `steps:
  a:
    job: x
    checks: [plain]
`, "needs a next target")
}

func TestOutcomeCheckMustBeLast(t *testing.T) {
	expectProblem(t, `steps:
  a:
    job: x
    checks: [outcome, plain]
    on:
      yes: end
      no: end
`, `check "outcome", which declares outcomes, before another check`)
}

func TestPlainAndOutcomeRoutingWork(t *testing.T) {
	text := header + `steps:
  a:
    job: x
    checks: [plain]
    then: b
  b:
    timeout: 1h
    checks: [plain, outcome]
    on:
      yes: end
      no: a
`
	w, err := Load([]byte(text), "inline.yaml", ".")
	if err != nil {
		t.Fatal(err)
	}
	if target, ok := w.Target(w.Steps["a"], "pass"); !ok || target != "b" {
		t.Fatalf("plain target %q %v", target, ok)
	}
	if target, ok := w.Target(w.Steps["b"], "no"); !ok || target != "a" {
		t.Fatalf("outcome target %q %v", target, ok)
	}
	if _, ok := w.Target(w.Steps["b"], "other"); ok {
		t.Fatal("unmapped outcome routed")
	}
	if got := w.Targets(w.Steps["b"]); len(got) != 2 {
		t.Fatalf("targets %v", got)
	}
}

func TestDurationsAndVisits(t *testing.T) {
	expectProblem(t, `steps:
  a:
    job: x
    checks: [plain]
    then: end
    timeout: soon
`, `invalid duration "soon"`)
	expectProblem(t, `steps:
  a:
    job: x
    checks: [plain]
    then: end
    max_visits: 0
`, "max_visits must be at least 1")
	expectProblem(t, `steps:
  a:
    job: x
    checks: [plain]
    then: end
    max_visits: many
`, "max_visits must be a whole number")
	expectProblem(t, `steps:
  a:
    job: x
    checks: [plain]
    then: end
    max_visits: 3 times
`, "max_visits must be a whole number")
	expectProblem(t, `steps:
  a:
    job: x
    checks: [plain]
    then: end
    max_visits: 2.9
`, "max_visits must be a whole number")
}

func TestNoWayOut(t *testing.T) {
	expectProblem(t, `steps:
  a:
    job: x
    checks: [plain]
    then: b
  b:
    job: x
    checks: [plain]
    then: a
`, "no path leads from the start step to end or stop")
}

func TestMissingPieces(t *testing.T) {
	problems := loadText(t, "start: nope\nsteps: {}\n")
	text := problemText(problems)
	if !strings.Contains(text, "workflow needs a name") {
		t.Fatalf("name: %s", text)
	}
	problems = loadText(t, "name: x\nstart: nope\nchecks:\n  c:\n    run: \"true\"\nsteps:\n  a:\n    job: x\n    checks: [c]\n    then: end\n")
	if !strings.Contains(problemText(problems), `start step "nope" is not a step`) {
		t.Fatalf("start: %s", problemText(problems))
	}
	problems = loadText(t, "name: x\nchecks:\n  c:\n    run: \"true\"\nsteps:\n  a:\n    job: x\n    checks: [c]\n    then: end\n")
	if !strings.Contains(problemText(problems), "needs a start step") {
		t.Fatalf("no start: %s", problemText(problems))
	}
	problems = loadText(t, "name: x\nstart: a\nsteps:\n  a:\n    job: x\n    then: end\n")
	if !strings.Contains(problemText(problems), "needs at least one check") {
		t.Fatalf("no checks: %s", problemText(problems))
	}
}

func TestLoaderShapeErrors(t *testing.T) {
	cases := map[string]string{
		"- a\n":                       "must be a mapping",
		"name: [a]\n":                 "expected a text value",
		"name: x\nsteps: 3\n":         "expected a mapping",
		"name: x\nchecks:\n  c: {}\n": `check "c" needs a run command`,
		"name: x\nchecks:\n  c:\n    run: a\n    cache: forever\n": `cache must be "commit"`,
		"name: x\nname: y\n":                                           `duplicate key "name"`,
		"name: x\nsteps:\n  a:\n    slow: maybe\n":                     "expected true or false",
		"name: x\nsteps:\n  a:\n    checks: c\n":                       "expected a list",
		"name: x\nsteps:\n  a:\n    timeout: [1]\n":                    "expected a duration",
		"name: x\ninputs:\n  i:\n    required: true\n    default: z\n": "cannot be both required and have a default",
		"name: x\nsteps:\n  a:\n    max_visits: [1]\n":                 "expected a whole number",
		"name: [unclosed":                                              "invalid YAML",
	}
	for text, want := range cases {
		problems := loadText(t, text)
		if !strings.Contains(problemText(problems), want) {
			t.Errorf("input %q: want %q in:\n%s", text, want, problemText(problems))
		}
	}
}

func TestInputShorthandIsRequired(t *testing.T) {
	text := header + "inputs:\n  topic:\n  other: {}\nsteps:\n  a:\n    job: x\n    checks: [plain]\n    then: end\n"
	w, err := Load([]byte(text), "inline.yaml", ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"topic", "other"} {
		in, _ := w.Input(name)
		if !in.Required {
			t.Errorf("%s not required", name)
		}
	}
}

func TestLoadFileMissing(t *testing.T) {
	if _, err := LoadFile("testdata/absent.yaml"); err == nil {
		t.Fatal("expected error")
	}
}

func TestProblemString(t *testing.T) {
	cases := map[string]Problem{
		"f.yaml:3: m": {File: "f.yaml", Line: 3, Message: "m"},
		"f.yaml: m":   {File: "f.yaml", Message: "m"},
		"m":           {Message: "m"},
	}
	for want, p := range cases {
		if p.String() != want {
			t.Errorf("got %q want %q", p.String(), want)
		}
	}
}

func TestDurations(t *testing.T) {
	good := map[string]time.Duration{
		"90s": 90 * time.Second, "5m": 5 * time.Minute, "2h": 2 * time.Hour,
		"14d": 14 * 24 * time.Hour, "1w": 7 * 24 * time.Hour, "1h30m": 90 * time.Minute,
	}
	for text, want := range good {
		got, err := ParseDuration(text)
		if err != nil || got != want {
			t.Errorf("%s: %v %v", text, got, err)
		}
	}
	for _, text := range []string{"", "5", "m", "5x", "0s", "-5s", "1.5h"} {
		if _, err := ParseDuration(text); err == nil {
			t.Errorf("%q parsed", text)
		}
	}
	formats := map[time.Duration]string{
		-time.Second: "0s", 5 * time.Second: "5s", 90 * time.Second: "1m 30s",
		90 * time.Minute: "1h 30m", 50 * time.Hour: "2d 2h",
	}
	for d, want := range formats {
		if got := FormatDuration(d); got != want {
			t.Errorf("FormatDuration(%v)=%q want %q", d, got, want)
		}
	}
	compact := map[time.Duration]string{
		-time.Second: "1s", 0: "1s", 90 * time.Second: "1m30s", 14 * 24 * time.Hour: "14d", 25 * time.Hour: "1d1h",
	}
	for d, want := range compact {
		if got := CompactDuration(d); got != want {
			t.Errorf("CompactDuration(%v)=%q want %q", d, got, want)
		}
	}
}

func TestResolveInputs(t *testing.T) {
	w, _ := loadFixture(t, "valid.yaml")
	got, err := w.ResolveInputs(map[string]string{"topic": "x"})
	if err != nil || got["topic"] != "x" || got["mode"] != "fast" {
		t.Fatalf("defaults: %v %v", got, err)
	}
	got, err = w.ResolveInputs(map[string]string{"topic": "x", "mode": "slow"})
	if err != nil || got["mode"] != "slow" {
		t.Fatalf("override: %v %v", got, err)
	}
	if _, err := w.ResolveInputs(nil); err == nil || !strings.Contains(err.Error(), `"topic"`) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := w.ResolveInputs(map[string]string{"topic": "x", "bogus": "y"}); err == nil {
		t.Fatal("unknown input accepted")
	}
}

func TestSemanticProblemsAreHeldBackWhenTheShapeIsBroken(t *testing.T) {
	problems := loadText(t, header+`steps:
  a:
    checks: [plain]
    then: [broken
`)
	if len(problems) == 0 {
		t.Fatal("expected problems")
	}
	if strings.Contains(problemText(problems), "cannot be reached") {
		t.Fatalf("semantic noise after a loader error:\n%s", problemText(problems))
	}
}

func TestLoaderAndSemanticProblemsAreReportedTogether(t *testing.T) {
	problems := loadText(t, header+`steps:
  a:
    checks: [plain]
    colour: red
    then: end
  orphan:
    checks: [plain]
    then: end
`)
	text := problemText(problems)
	for _, want := range []string{`unknown key "colour"`, `step "orphan" cannot be reached`} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}
