package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRunDelegates(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(context.Background(), nil, &out, &errOut); code != 0 || out.Len() == 0 {
		t.Fatalf("code %d out %q", code, out.String())
	}
	if code := run(context.Background(), []string{"nope"}, &out, &errOut); code != 1 {
		t.Fatalf("code %d", code)
	}
}
