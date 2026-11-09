package telegram

import "testing"

func TestHasPipeTable(t *testing.T) {
	cases := map[string]bool{
		"| a | b |\n|---|---|\n| 1 | 2 |":     true,
		"a | b\n:---|---:\n1 | 2":               true,
		"text\n\n| a | b |\n| --- | --- |\nx": true,
		"just | a pipe\nno divider":            false,
		"---\nhorizontal rule":                 false,
		"```\n|---|---|\n```":                  false,
	}
	for in, want := range cases {
		if got := hasPipeTable(in); got != want {
			t.Errorf("hasPipeTable(%q) = %v, want %v", in, got, want)
		}
	}
}
