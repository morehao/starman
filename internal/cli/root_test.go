package cli

import "testing"

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
		{name: "help flag", args: []string{"--help"}, want: false},
		{name: "version flag", args: []string{"--version"}, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasSubcommandArgs(tc.args); got != tc.want {
				t.Fatalf("hasSubcommandArgs(%v)=%v want=%v", tc.args, got, tc.want)
			}
		})
	}
}
