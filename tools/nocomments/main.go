package main

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func goComments(path string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, group := range file.Comments {
		for _, c := range group.List {
			if strings.HasPrefix(c.Text, "//go:") {
				continue
			}
			found = append(found, fmt.Sprintf("%s: code comment", fset.Position(c.Pos())))
		}
	}
	return found, nil
}

func shellComments(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var found []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(text, "#") {
			continue
		}
		if line == 1 && strings.HasPrefix(text, "#!") {
			continue
		}
		found = append(found, fmt.Sprintf("%s:%d: code comment", path, line))
	}
	return found, scanner.Err()
}

func skipDir(name string) bool {
	return name == ".git" || name == "node_modules" || name == ".claude" || name == "vendor"
}

func scan(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		var hits []string
		var scanErr error
		switch filepath.Ext(path) {
		case ".go":
			hits, scanErr = goComments(path)
		case ".sh":
			hits, scanErr = shellComments(path)
		default:
			return nil
		}
		if scanErr != nil {
			return scanErr
		}
		found = append(found, hits...)
		return nil
	})
	return found, err
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	found, err := scan(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, line := range found {
		fmt.Println(line)
	}
	if len(found) > 0 {
		fmt.Fprintf(os.Stderr, "%d code comments found\n", len(found))
		os.Exit(1)
	}
}
