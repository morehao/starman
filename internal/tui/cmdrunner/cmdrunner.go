package cmdrunner

import (
	"context"
	"errors"
)

var (
	ErrInteractiveRequired = errors.New("interactive required")
	ErrBlockingRequired    = errors.New("blocking required")
)

type Runner struct {
	version string
}

func NewRunner(version string) *Runner {
	return &Runner{version: version}
}

func (r *Runner) Run(ctx context.Context, input string) (stdout, stderr string, err error) {
	return "", "", nil
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
