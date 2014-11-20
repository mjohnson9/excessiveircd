package server_test

import (
	"testing"

	"github.com/nightexcessive/excessiveircd/server"
)

var parseMessageTests = []struct {
	Line string

	Prefix  string
	Command string
	Params  []string
}{
	{
		":jamie!jamie@127.0.0.1 PRIVMSG #go-nuts :Hello!  I love Go!",
		"jamie!jamie@127.0.0.1", "PRIVMSG", []string{"#go-nuts", "Hello!  I love Go!"},
	},
	{
		":jamie!jamie@127.0.0.1 JOIN #go-nuts",
		"jamie!jamie@127.0.0.1", "JOIN", []string{"#go-nuts"},
	},
	{
		":jamie!jamie@127.0.0.1 JOIN #go-nuts key1 #chan2 key2 :Extra param",
		"jamie!jamie@127.0.0.1", "JOIN", []string{"#go-nuts", "key1", "#chan2", "key2", "Extra param"},
	},
	{
		":jamie!jamie@127.0.0.1 AWAY",
		"jamie!jamie@127.0.0.1", "AWAY", []string{},
	},

	{
		"PRIVMSG #go-nuts :Hello!  I love Go!",
		"", "PRIVMSG", []string{"#go-nuts", "Hello!  I love Go!"},
	},
	{
		"JOIN #go-nuts",
		"", "JOIN", []string{"#go-nuts"},
	},
	{
		"AWAY",
		"", "AWAY", []string{},
	},
}

func TestParseMessage(t *testing.T) {
	for _, test := range parseMessageTests {
		t.Logf("Parsing %q...", test.Line)

		parsed, err := server.ParseMessage(test.Line)
		if err != nil {
			t.Fatalf("Error in parsing message: %s", err)
			continue
		}

		if parsed.Prefix != test.Prefix {
			t.Errorf("prefix: expected %q, got %q", test.Prefix, parsed.Prefix)
		}

		if parsed.Command != test.Command {
			t.Errorf("command: expected %q, got %q", test.Command, parsed.Command)
		}

		if len(parsed.Params) != len(test.Params) {
			t.Errorf("Expected %d parameters, got %d", len(test.Params), len(parsed.Params))
		}

		for i, expected := range test.Params {
			if i >= len(parsed.Params) {
				break
			}

			got := parsed.Params[i]
			if expected != got {
				t.Errorf("param[%d]: expected %q, got %q", i, expected, got)
			}
		}

		if len(parsed.Params) > len(test.Params) {
			start := len(test.Params) - 1
			if start < 0 {
				start = 0
			}
			t.Errorf("extra parameters: %v", parsed.Params[:])
		}

		if str := parsed.String(); str != test.Line {
			t.Errorf("String(): expected %q, got %q", test.Line, str)
		}
	}
}

func benchmarkHelper(b *testing.B, line string) {
	for i := 0; i < b.N; i++ {
		if _, err := server.ParseMessage(line); err != nil {
			b.Fatalf("Error parsing message: %s", err)
		}
	}
}

func BenchmarkParseMessage(b *testing.B) {
	benchmarkHelper(b, ":jamie!jamie@127.0.0.1 PRIVMSG #go-nuts :Hello!  I love Go!")
}

func BenchmarkParseMessage_NoArgs(b *testing.B) {
	benchmarkHelper(b, ":jamie!jamie@127.0.0.1 AWAY")
}

func BenchmarkParseMessage_NoArgsNoPrefix(b *testing.B) {
	benchmarkHelper(b, "AWAY")
}

func BenchmarkParseMessage_NoPrefix(b *testing.B) {
	benchmarkHelper(b, "PRIVMSG #go-nuts :Hello!  I love Go!")
}
