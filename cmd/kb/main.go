package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/matheus3301/kb/internal/cli"
	"github.com/matheus3301/kb/internal/core"
	"github.com/matheus3301/kb/internal/mcpserver"
	"github.com/matheus3301/kb/internal/write"
)

var version = "dev"
var commit = "unknown"

func dependencies() cli.Dependencies {
	return cli.Dependencies{Version: version, Commit: commit, ResolveRoot: core.ResolveRoot,
		Open: func(root string) (mcpserver.Reader, mcpserver.Writer, error) {
			s, err := core.Open(root)
			if err != nil {
				return nil, nil, err
			}
			return s, write.New(s), nil
		},
		Init: func(ctx context.Context, root string, para bool) (mcpserver.Reader, mcpserver.Writer, error) {
			s, err := core.Init(ctx, root, para)
			if err != nil {
				return nil, nil, err
			}
			return s, write.New(s), nil
		},
	}
}

func run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	return cli.Run(ctx, args, in, out, errOut, dependencies())
}

func main() { os.Exit(mainCode()) }

func mainCode() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}
