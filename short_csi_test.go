package phosphor

import (
	"bytes"
	"fmt"
	"testing"
)

func TestShortCSIBytewiseContinuation(t *testing.T) {
	prefix := []byte("\x1b[?2027h\x1b[?1049hALT\x1b[?1049lmain\r\n\x1b]8;;https://example.test/\x1b\\\x1b[4:3;58:2::1:2:3m👩‍💻\x1b7\x1b[1\"q")
	for code := byte(0x40); code <= 0x7e; code++ {
		params := []string{"", "00", "01", "09"}
		for value := 0; value < 100; value++ {
			params = append(params, fmt.Sprint(value))
		}
		for _, param := range params {
			input := append(bytes.Clone(prefix), []byte("\x1b["+param+string(code))...)
			var gotReply, wantReply bytes.Buffer
			a := New(12, 5, WithReply(&gotReply))
			b := New(12, 5, WithReply(&wantReply))
			a.Write(input)
			for i := range input {
				b.Write(input[i : i+1])
			}
			if !bytes.Equal(a.Snapshot(), b.Snapshot()) || !bytes.Equal(gotReply.Bytes(), wantReply.Bytes()) {
				t.Fatalf("CSI %s%c changed complete input", param, code)
			}
			for _, suffix := range []string{"\x1b[3", "1mX\x1b[6n", "\x1b8Y\x1b[?1049h\x1b[?1049l"} {
				a.Write([]byte(suffix))
				for i := range suffix {
					b.Write([]byte(suffix[i : i+1]))
				}
				a.Resize(9, 4)
				b.Resize(9, 4)
				if !bytes.Equal(a.Snapshot(), b.Snapshot()) || !bytes.Equal(gotReply.Bytes(), wantReply.Bytes()) {
					t.Fatalf("CSI %s%c changed future input %q", param, code, suffix)
				}
			}
		}
	}
}
