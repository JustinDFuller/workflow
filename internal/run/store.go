package run

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

var (
	errFirstEvent = errors.New("run log does not begin with a started event")
	errNoStep     = errors.New("run log never entered a step")
	ErrNoRun      = errors.New("no run")
)

const HomeEnv = "WF_HOME"

type Store struct {
	Home string
	Now  func() time.Time
}

func DefaultHome(getenv func(string) string) (string, error) {
	if v := getenv(HomeEnv); v != "" {
		return filepath.Abs(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".wf"), nil
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Store) runsDir() string {
	return filepath.Join(s.Home, "runs")
}

func (s *Store) CacheDir() string {
	return filepath.Join(s.Home, "cache")
}

func (s *Store) runDir(id string) string {
	return filepath.Join(s.runsDir(), id)
}

func (s *Store) logPath(id string) string {
	return filepath.Join(s.runDir(id), "events.jsonl")
}

func newID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) append(id string, e Event) error {
	if e.Time.IsZero() {
		e.Time = s.now().UTC()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.logPath(id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func readEvents(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var events []Event
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 && line[len(line)-1] == '\n' {
			var e Event
			if jerr := json.Unmarshal(line, &e); jerr == nil {
				events = append(events, e)
			}
		}
		if err != nil {
			break
		}
	}
	return events, nil
}

func (s *Store) Load(id string) (*RunState, error) {
	events, err := readEvents(s.logPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w %q", ErrNoRun, id)
		}
		return nil, err
	}
	st, err := Replay(events)
	if err != nil {
		return nil, fmt.Errorf("run %s: %w", id, err)
	}
	st.ID = id
	return st, nil
}

func (s *Store) IDs() ([]string, error) {
	entries, err := os.ReadDir(s.runsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

func (s *Store) All() ([]*RunState, error) {
	ids, err := s.IDs()
	if err != nil {
		return nil, err
	}
	var runs []*RunState
	for _, id := range ids {
		st, err := s.Load(id)
		if err != nil {
			continue
		}
		runs = append(runs, st)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].StartedAt.After(runs[j].StartedAt) })
	return runs, nil
}

func (s *Store) ForWorktree(worktree string) (*RunState, error) {
	runs, err := s.All()
	if err != nil {
		return nil, err
	}
	for _, r := range runs {
		if r.Worktree == worktree {
			return r, nil
		}
	}
	return nil, fmt.Errorf("%w for %s", ErrNoRun, worktree)
}

type lockFile struct {
	f *os.File
}

func lock(ctx context.Context, path string) (*lockFile, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &lockFile{f: f}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-time.After(lockPoll):
		}
	}
}

func (l *lockFile) unlock() {
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	l.f.Close()
}

const lockPoll = 25 * time.Millisecond

func (s *Store) lockRun(ctx context.Context, id string) (*lockFile, error) {
	return lock(ctx, filepath.Join(s.runDir(id), "lock"))
}
