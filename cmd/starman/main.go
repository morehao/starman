package main

import (
	"fmt"
	"os"

	"github.com/morehao/starman/internal/cli"
	"github.com/morehao/starman/internal/tui"
	"github.com/morehao/starman/internal/tui/cmdrunner"
	"github.com/morehao/starman/internal/version"
)

func main() {
	tui.Runner = cmdrunner.NewRunner(version.Info())
	if err := cli.Run(version.Info()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
