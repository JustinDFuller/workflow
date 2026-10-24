package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JustinDFuller/workflow/internal/run"
)

func wf(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := Run(args, Env{Stdout: &out, Stderr: &errOut})
	return code, out.String(), errOut.String()
}

func TestUsageAndUnknown(t *testing.T) {
	code, out, _ := wf()
	if code != 0 || !strings.Contains(out, "Usage:") {
		t.Fatalf("usage: %d %q", code, out)
	}
	code, _, errOut := wf("nope")
	if code != ExitUsage || !strings.Contains(errOut, "unknown command") {
		t.Fatalf("unknown: %d %q", code, errOut)
	}
}

func TestValidate(t *testing.T) {
	code, out, _ := wf("validate", "testdata/valid.yaml")
	if code != 0 || !strings.Contains(out, "is valid") {
		t.Fatalf("good: %d %q", code, out)
	}
	code, _, errOut := wf("validate", "testdata/several_problems.yaml")
	if code == 0 || !strings.Contains(errOut, "several_problems.yaml:") || !strings.Contains(errOut, "problem(s)") {
		t.Fatalf("bad: %d %q", code, errOut)
	}
	code, _, errOut = wf("validate", "testdata/absent.yaml")
	if code == 0 || !strings.Contains(errOut, "wf:") {
		t.Fatalf("absent: %d %q", code, errOut)
	}
	code, _, errOut = wf("validate")
	if code != ExitUsage || !strings.Contains(errOut, "usage:") {
		t.Fatalf("no args: %d %q", code, errOut)
	}
}

func TestExitCodesAreDistinct(t *testing.T) {
	seen := map[int]bool{}
	for _, c := range []int{run.ExitWork, run.ExitPending, run.ExitBroken, run.ExitStopped, run.ExitEnded} {
		if seen[c] || c == ExitUsage {
			t.Fatalf("code %d repeats or is the usage code", c)
		}
		seen[c] = true
	}
}
