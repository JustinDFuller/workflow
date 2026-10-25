package check

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JustinDFuller/workflow/internal/workflow"
)

type Cache struct {
	Dir string
}

func (c *Cache) path(key string) string {
	return filepath.Join(c.Dir, key+".json")
}

func (c *Cache) Get(key string) (Result, bool) {
	data, err := os.ReadFile(c.path(key))
	if err != nil {
		return Result{}, false
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return Result{}, false
	}
	return r, true
}

func (c *Cache) Put(key string, r Result) {
	data, err := json.Marshal(r)
	if err != nil {
		return
	}
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(c.Dir, "put-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		return
	}
	if err := os.Rename(tmp.Name(), c.path(key)); err != nil {
		os.Remove(tmp.Name())
	}
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

func Fingerprint(ctx context.Context, worktree string) (string, error) {
	head, err := git(ctx, worktree, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	diff, err := git(ctx, worktree, "diff", "HEAD", "--binary")
	if err != nil {
		return "", err
	}
	untracked, err := git(ctx, worktree, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	fmt.Fprintf(h, "head\n%s\ndiff\n%s\n", strings.TrimSpace(head), diff)
	for _, name := range strings.Split(untracked, "\x00") {
		if name == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(worktree, name))
		if err != nil {
			continue
		}
		fmt.Fprintf(h, "untracked\n%s\n%d\n", name, len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func CacheKey(c *workflow.Check, worktree, fingerprint string, inputs map[string]string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s", c.Name, c.Run, worktree, fingerprint)
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(h, "\x00%s=%s", name, inputs[name])
	}
	return hex.EncodeToString(h.Sum(nil))
}
