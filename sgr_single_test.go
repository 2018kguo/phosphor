package phosphor

import (
	"bytes"
	"fmt"
	"testing"
)

func TestSingleSGRMatchesGenericParameterDispatch(t *testing.T) {
	for _, prelude := range []string{
		"",
		"\x1b[1;2;3;4:3;48;2;12;34;56m",
		"\x1b[1\"q\x1b]8;id=live;https://example.test\x1b\\\x1b[58;2;1;2;3m",
	} {
		for parameter := 0; parameter <= 107; parameter++ {
			fast, generic := New(20, 4), New(20, 4)
			fast.Write([]byte(prelude))
			generic.Write([]byte(prelude))
			fast.Write(fmt.Appendf(nil, "\x1b[%dm", parameter))
			// Unknown SGR999 leaves the pen unchanged and forces the
			// multi-parameter route for the following operation.
			generic.Write(fmt.Appendf(nil, "\x1b[999;%dm", parameter))
			if fast.cur.pen != generic.cur.pen {
				t.Fatalf("prelude %q SGR%d pen = %+v, generic %+v", prelude, parameter, fast.cur.pen, generic.cur.pen)
			}
			for _, term := range []*Terminal{fast, generic} {
				term.Write([]byte("same\x1b[H"))
			}
			if !bytes.Equal(fast.Snapshot(), generic.Snapshot()) {
				t.Fatalf("prelude %q SGR%d: continued state differs", prelude, parameter)
			}
		}
	}
}

func TestSingleSGRColonGroupsStayOnGenericPath(t *testing.T) {
	for _, input := range []string{"\x1b[0:m", "\x1b[31:m", "\x1b[0::m", "\x1b[31:1m"} {
		term := New(20, 4)
		term.Write([]byte("\x1b[1;34m"))
		before := term.cur.pen
		term.Write([]byte(input))
		if term.cur.pen != before {
			t.Fatalf("%q changed pen %+v to %+v", input, before, term.cur.pen)
		}
	}
}
