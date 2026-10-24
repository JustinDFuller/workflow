package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"go.yaml.in/yaml/v3"
)

type loader struct {
	file     string
	problems Problems
}

func (l *loader) add(line int, format string, args ...any) {
	l.problems = append(l.problems, Problem{File: l.file, Line: line, Message: fmt.Sprintf(format, args...)})
}

func LoadFile(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return Load(data, path, filepath.Dir(abs))
}

func Load(data []byte, file, dir string) (*Workflow, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, Problems{{File: file, Message: fmt.Sprintf("invalid YAML: %v", err)}}
	}
	l := &loader{file: file}
	w := &Workflow{
		File:   file,
		Dir:    dir,
		Checks: map[string]*Check{},
		Steps:  map[string]*Step{},
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		l.add(1, "workflow file must be a mapping")
		return nil, l.problems
	}
	root := doc.Content[0]
	l.mapping(root, func(key string, k, v *yaml.Node) bool {
		switch key {
		case "name":
			w.Name = l.str(v)
		case "description":
			w.Description = l.str(v)
		case "start":
			w.Start = l.str(v)
			w.startLine = v.Line
		case "inputs":
			l.inputs(w, v)
		case "checks":
			l.checks(w, v)
		case "steps":
			l.steps(w, v)
		default:
			return false
		}
		return true
	})
	if w.Name == "" {
		l.add(root.Line, "workflow needs a name")
	}
	problems := append(Problems{}, l.problems...)
	problems = append(problems, Validate(w)...)
	if len(problems) > 0 {
		sort.SliceStable(problems, func(i, j int) bool { return problems[i].Line < problems[j].Line })
		return w, problems
	}
	return w, nil
}

func (l *loader) mapping(n *yaml.Node, fn func(key string, k, v *yaml.Node) bool) {
	if n.Kind != yaml.MappingNode {
		l.add(n.Line, "expected a mapping")
		return
	}
	seen := map[string]bool{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if seen[k.Value] {
			l.add(k.Line, "duplicate key %q", k.Value)
			continue
		}
		seen[k.Value] = true
		if !fn(k.Value, k, v) {
			l.add(k.Line, "unknown key %q", k.Value)
		}
	}
}

func (l *loader) str(n *yaml.Node) string {
	if n.Kind != yaml.ScalarNode {
		l.add(n.Line, "expected a text value")
		return ""
	}
	return n.Value
}

func (l *loader) boolean(n *yaml.Node) bool {
	if n.Kind != yaml.ScalarNode || (n.Value != "true" && n.Value != "false") {
		l.add(n.Line, "expected true or false")
		return false
	}
	return n.Value == "true"
}

func (l *loader) list(n *yaml.Node) []string {
	if n.Kind != yaml.SequenceNode {
		l.add(n.Line, "expected a list")
		return nil
	}
	out := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		out = append(out, l.str(item))
	}
	return out
}

func (l *loader) duration(n *yaml.Node) time.Duration {
	if n.Kind != yaml.ScalarNode {
		l.add(n.Line, "expected a duration such as 90s, 5m, 2h, or 14d")
		return 0
	}
	d, err := ParseDuration(n.Value)
	if err != nil {
		l.add(n.Line, "%v", err)
		return 0
	}
	return d
}

func (l *loader) inputs(w *Workflow, n *yaml.Node) {
	l.mapping(n, func(name string, k, v *yaml.Node) bool {
		in := Input{Name: name, Line: k.Line}
		if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
			in.Required = true
			w.Inputs = append(w.Inputs, in)
			return true
		}
		hasDefault := false
		l.mapping(v, func(key string, _, vv *yaml.Node) bool {
			switch key {
			case "required":
				in.Required = l.boolean(vv)
			case "default":
				in.Default = l.str(vv)
				hasDefault = true
			default:
				return false
			}
			return true
		})
		if hasDefault && in.Required {
			l.add(k.Line, "input %q cannot be both required and have a default", name)
		}
		if !hasDefault && !in.Required {
			in.Required = true
		}
		w.Inputs = append(w.Inputs, in)
		return true
	})
}

func (l *loader) checks(w *Workflow, n *yaml.Node) {
	l.mapping(n, func(name string, k, v *yaml.Node) bool {
		c := &Check{Name: name, Line: k.Line}
		l.mapping(v, func(key string, _, vv *yaml.Node) bool {
			switch key {
			case "run":
				c.Run = l.str(vv)
			case "timeout":
				c.Timeout = l.duration(vv)
			case "outcomes":
				c.Outcomes = l.list(vv)
			case "cache":
				c.Cache = l.str(vv)
				if c.Cache != CacheCommit {
					l.add(vv.Line, "cache must be %q", CacheCommit)
				}
			default:
				return false
			}
			return true
		})
		if c.Run == "" {
			l.add(k.Line, "check %q needs a run command", name)
		}
		w.Checks[name] = c
		w.CheckOrder = append(w.CheckOrder, name)
		return true
	})
}

func (l *loader) steps(w *Workflow, n *yaml.Node) {
	l.mapping(n, func(name string, k, v *yaml.Node) bool {
		s := &Step{Name: name, Line: k.Line, On: map[string]string{}, lines: map[string]int{}}
		l.mapping(v, func(key string, kk, vv *yaml.Node) bool {
			s.lines[key] = kk.Line
			switch key {
			case "job":
				s.Job = l.str(vv)
			case "checks":
				s.Checks = l.list(vv)
			case "then":
				s.Then = l.str(vv)
			case "on":
				l.mapping(vv, func(outcome string, _, target *yaml.Node) bool {
					s.On[outcome] = l.str(target)
					s.OnOrder = append(s.OnOrder, outcome)
					s.lines["on."+outcome] = target.Line
					return true
				})
			case "max_visits":
				if vv.Kind != yaml.ScalarNode {
					l.add(vv.Line, "expected a whole number")
					break
				}
				var n int
				if _, err := fmt.Sscanf(vv.Value, "%d", &n); err != nil {
					l.add(vv.Line, "max_visits must be a whole number")
					break
				}
				if n < 1 {
					l.add(vv.Line, "max_visits must be at least 1")
				}
				s.MaxVisits = n
			case "timeout":
				s.Timeout = l.duration(vv)
			case "deny_tools":
				s.DenyTools = l.list(vv)
			case "slow":
				s.Slow = l.boolean(vv)
			case "needs_you":
				s.NeedsYou = l.boolean(vv)
			default:
				return false
			}
			return true
		})
		w.Steps[name] = s
		w.StepOrder = append(w.StepOrder, name)
		return true
	})
}
