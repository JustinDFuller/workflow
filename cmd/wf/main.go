package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/JustinDFuller/workflow/internal/cli"
)

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return cli.Run(args, cli.Env{Stdout: stdout, Stderr: stderr, Stdin: os.Stdin, Ctx: ctx})
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
