package main

import (
	"github.com/morehao/starman/internal/cli"
	"github.com/morehao/starman/internal/version"
)

func main() {
	cli.Run(version.Info())
}
