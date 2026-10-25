package main

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFlagsGoComments(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package a\n\n// hello\nvar X = 1\n")
	found, err := scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("want 1 finding, got %v", found)
	}
}

func TestScanIgnoresStringsAndDirectives(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "//go:build linux\n\npackage a\n\nvar URL = \"https://example.test\"\n")
	found, err := scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("want no findings, got %v", found)
	}
}

func TestScanFlagsShellComments(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.sh", "#!/usr/bin/env bash\n# explain\necho hi # trailing is not a leading comment\n")
	found, err := scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("want 1 finding, got %v", found)
	}
}

func TestScanSkipsVendoredDirectories(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "node_modules/x/a.go", "package a\n// c\n")
	write(t, dir, "ok.go", "package a\n")
	found, err := scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("want no findings, got %v", found)
	}
}

func TestScanReportsParseErrors(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "bad.go", "this is not go")
	if _, err := scan(dir); err == nil {
		t.Fatal("want error for unparsable file")
	}
}
