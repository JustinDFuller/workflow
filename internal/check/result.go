package check

import "strings"

type Status string

const (
	Pass    Status = "pass"
	Fail    Status = "fail"
	Pending Status = "pending"
	Broken  Status = "broken"
)

const MaxMessage = 64 * 1024

const CutMarker = "\n[output cut: longer than 64 KiB]"

type Artifact struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type Result struct {
	Status    Status            `json:"status"`
	Message   string            `json:"message"`
	Outcome   string            `json:"outcome,omitempty"`
	Values    map[string]string `json:"values,omitempty"`
	Artifacts []Artifact        `json:"artifacts,omitempty"`
}

func (r Result) Passed() bool {
	return r.Status == Pass
}

func (r Result) Cacheable() bool {
	return r.Status == Pass || r.Status == Fail
}

func parseArtifact(line string) (Artifact, bool) {
	rest, ok := strings.CutPrefix(line, "artifact:")
	if !ok {
		return Artifact{}, false
	}
	fields := strings.Fields(rest)
	if len(fields) != 2 {
		return Artifact{}, false
	}
	name, address := fields[0], fields[1]
	if !validAddress(address) {
		return Artifact{}, false
	}
	return Artifact{Name: name, Address: address}, true
}

func validAddress(address string) bool {
	if strings.Contains(address, "://") {
		return true
	}
	if strings.HasPrefix(address, "/") {
		return false
	}
	for _, part := range strings.Split(address, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

func splitArtifacts(out string) (string, []Artifact) {
	var artifacts []Artifact
	var kept []string
	for _, line := range strings.Split(out, "\n") {
		if a, ok := parseArtifact(line); ok {
			artifacts = append(artifacts, a)
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n")), artifacts
}

func cut(s string) string {
	if len(s) <= MaxMessage {
		return s
	}
	return s[:MaxMessage] + CutMarker
}
