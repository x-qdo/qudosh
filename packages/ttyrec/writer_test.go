package ttyrec

import (
	"bytes"
	"testing"
	"time"
)

const (
	testWriteDelay      = 50 * time.Millisecond
	testWriteDelaySplay = 10 * time.Millisecond
)

func TestEncoder(t *testing.T) {
	var (
		parts  = []string{"this", "is", "a", "test", ""}
		buf    bytes.Buffer
		enc    = NewEncoder(&buf)
		dec    = NewDecoder(&buf)
		delays = time.NewTicker(testWriteDelay)
	)
	defer delays.Stop()
	for _, part := range parts {
		if n, err := enc.Write([]byte(part)); err != nil {
			t.Fatal(err)
		} else if n != len(part) {
			t.Errorf("expected write of size %d, got %d", len(part), n)
		}
		<-delays.C
	}

	var (
		frames, stop = dec.DecodeStream()
		previous     *Frame
	)
	for frame := range frames {
		if string(frame.Data) != parts[0] {
			t.Errorf("expected frame data %q, got %q", parts[0], frame.Data)
		}

		if previous != nil {
			var (
				delay = frame.Time.Sub(previous.Time)
				splay = delay - testWriteDelaySplay
			)
			if splay > testWriteDelay {
				t.Errorf("frame with delay %s, expected %s (±%s)",
					delay, testWriteDelay, testWriteDelaySplay)
			}
		}
		previous = frame

		if parts = parts[1:]; len(parts) == 0 {
			stop()
			return
		} else if parts[0] == "" {
			parts = parts[1:]
			break
		}
	}

	if len(parts) > 0 {
		t.Fatalf("%d frames did not read", len(parts))
	}
}
