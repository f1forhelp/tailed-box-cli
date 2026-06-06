package main

import "testing"

func TestHasCommandArg(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: false},
		{name: "help flag only", args: []string{"--help"}, want: false},
		{name: "global flags only", args: []string{"--state-dir", "/tmp/state", "--json"}, want: false},
		{name: "global equals flags only", args: []string{"--config=/tmp/config.json"}, want: false},
		{name: "command", args: []string{"logs", "--follow"}, want: true},
		{name: "command after global flag", args: []string{"--state-dir", "/tmp/state", "logs", "--follow"}, want: true},
		{name: "help command", args: []string{"help", "logs"}, want: true},
		{name: "command after separator", args: []string{"--", "logs"}, want: true},
		{name: "empty separator", args: []string{"--"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasCommandArg(tt.args); got != tt.want {
				t.Fatalf("hasCommandArg(%#v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
