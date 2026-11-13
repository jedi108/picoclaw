package telegram

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestHasPipeTable(t *testing.T) {
	cases := map[string]bool{
		"| a | b |\n|---|---|\n| 1 | 2 |":     true,
		"a | b\n:---|---:\n1 | 2":             true,
		"text\n\n| a | b |\n| --- | --- |\nx": true,
		"just | a pipe\nno divider":           false,
		"---\nhorizontal rule":                false,
		"```\n|---|---|\n```":                 false,
	}
	for in, want := range cases {
		if got := hasPipeTable(in); got != want {
			t.Errorf("hasPipeTable(%q) = %v, want %v", in, got, want)
		}
	}
}

// gobond#300: a rich table must reach sendRichMessage whole, not split at 4000 by the manager.
func TestMaxMessageLengthFor(t *testing.T) {
	table := "| a | b |\n|---|---|\n" + strings.Repeat("| x | y |\n", 600) // ~6000 chars
	base := channels.NewBaseChannel("telegram", nil, nil, nil, channels.WithMaxMessageLength(4000))
	ch := &TelegramChannel{BaseChannel: base, tgCfg: &config.TelegramSettings{RichMessages: true}}
	if got := ch.MaxMessageLengthFor(table); got != richMessageMaxChars {
		t.Fatalf("rich table: %d", got)
	}
	if got := ch.MaxMessageLengthFor(strings.Repeat("plain ", 1000)); got != 4000 {
		t.Fatalf("plain text: %d", got)
	}
	if got := ch.MaxMessageLengthFor(strings.Repeat(table, 6)); got != 4000 { // > 32768: rich impossible
		t.Fatalf("oversized table: %d", got)
	}
	ch.richUnsupported.Store(true)
	if got := ch.MaxMessageLengthFor(table); got != 4000 {
		t.Fatalf("server without rich: %d", got)
	}
	ch.tgCfg.RichMessages = false
	ch.richUnsupported.Store(false)
	if got := ch.MaxMessageLengthFor(table); got != 4000 {
		t.Fatalf("rich_messages off: %d", got)
	}
}
