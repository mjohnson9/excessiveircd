package protocol_test

import (
	"testing"

	"github.com/nightexcessive/excessiveircd/protocol"
)

func TestIsValid_AlwaysTrue(t *testing.T) {
	alwaysTrue := func(i int, r rune) bool {
		return true
	}

	if !protocol.IsValid("abcd", alwaysTrue) {
		t.Error("false value given, expected true")
	}
}
