package cmdrunner

import (
	"bytes"
	"context"
	"strings"

	"github.com/morehao/starman/internal/cli"
	"github.com/morehao/starman/internal/tui"
)

var (
	ErrInteractiveRequired = tui.ErrInteractiveRequired
	ErrBlockingRequired    = tui.ErrBlockingRequired
)

type Runner struct {
	version string
}

func NewRunner(version string) *Runner {
	return &Runner{version: version}
}

func (r *Runner) Run(ctx context.Context, input string) (stdout, stderr string, err error) {
	args := shellSplit(input)
	if len(args) == 0 {
		return "", "", nil
	}

	if len(args) >= 2 && args[0] == "config" && args[1] == "init" {
		return "", "", tui.ErrInteractiveRequired
	}

	if args[0] == "sync" {
		for _, a := range args[1:] {
			if a == "--watch" || strings.HasPrefix(a, "--watch=") {
				return "", "", tui.ErrBlockingRequired
			}
		}
	}

	cmd := cli.NewRootCmd(r.version)
	cmd.SetArgs(args)

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.ExecuteContext(ctx)
	return outBuf.String(), errBuf.String(), err
}

func shellSplit(input string) []string {
	var args []string
	var current []byte
	inSingle := false
	inDouble := false

	for i := 0; i < len(input); i++ {
		c := input[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == ' ' && !inSingle && !inDouble:
			if len(current) > 0 {
				args = append(args, string(current))
				current = current[:0]
			}
		default:
			current = append(current, c)
		}
	}
	if len(current) > 0 {
		args = append(args, string(current))
	}
	return args
}
