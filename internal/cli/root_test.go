package cli

import (
	"testing"

	"github.com/morehao/starman/internal/config"
)

func TestHasSubcommandArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: false},
		{name: "only global bool flag", args: []string{"--verbose"}, want: false},
		{name: "global string flag with value", args: []string{"--config", "cfg.yaml"}, want: false},
		{name: "global string equals value", args: []string{"--token=abc"}, want: false},
		{name: "subcommand only", args: []string{"sync"}, want: true},
		{name: "global flags and subcommand", args: []string{"--verbose", "search", "go"}, want: true},
		{name: "help flag", args: []string{"--help"}, want: true},
		{name: "version flag", args: []string{"--version"}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasSubcommandArgs(tc.args); got != tc.want {
				t.Fatalf("hasSubcommandArgs(%v)=%v want=%v", tc.args, got, tc.want)
			}
		})
	}
}

func TestConfigPathForArgs(t *testing.T) {
	defaultPath, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no args", args: nil, want: defaultPath},
		{name: "only global bool flag", args: []string{"--verbose"}, want: defaultPath},
		{name: "config with separate value", args: []string{"--config", "./custom.yaml"}, want: "./custom.yaml"},
		{name: "config with equals value", args: []string{"--config=./custom.yaml"}, want: "./custom.yaml"},
		{name: "config mixed with other global flags", args: []string{"--token=abc", "--config", "./custom.yaml"}, want: "./custom.yaml"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := configPathForArgs(tc.args)
			if err != nil {
				t.Fatalf("configPathForArgs(%v) error=%v", tc.args, err)
			}
			if got != tc.want {
				t.Fatalf("configPathForArgs(%v)=%q want=%q", tc.args, got, tc.want)
			}
		})
	}
}
