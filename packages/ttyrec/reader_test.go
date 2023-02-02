package ttyrec

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDecoder_SeekToFrame(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "matrix.tty"))
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()

	var (
		dec      = NewDecoder(f)
		firstPos int64
		testPos  int64
		want     int
	)

	// Read first frame.
	if _, err = dec.DecodeFrame(); err != nil {
		t.Fatal(err)
	}
	// Record file reader position.
	if firstPos, err = f.Seek(0, io.SeekCurrent); err != nil {
		t.Skip(err)
	}
	// Increase frame read count.
	want++

	// Jump around!
	for _, offset := range []int{42, -23, 0, 111, -131} {
		if err = dec.SeekToFrame(offset, io.SeekCurrent); err != nil {
			t.Fatal(err)
		}
		want += offset
		if test := dec.Frame(); test != want {
			t.Fatalf("expected to be at frame %d, got %d", want, test)
		}
	}
	if want != 0 {
		t.Fatalf("expected to back at start, at frame %d", want)
	}

	// Read first frame again, so we can compare the position.
	if _, err = dec.DecodeFrame(); err != nil {
		t.Fatal(err)
	}
	if testPos, err = f.Seek(0, io.SeekCurrent); err != nil {
		t.Skip(err)
	}
	if testPos != firstPos {
		t.Errorf("expected to be at offset %d, is at %d ", firstPos, testPos)
	}

	// Explicit set to 1
	if _, err = dec.DecodeFrame(); err != nil {
		t.Fatal(err)
	}
	if err = dec.SeekToFrame(1, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if testPos, err = f.Seek(0, io.SeekCurrent); err != nil {
		t.Skip(err)
	}
	if testPos != firstPos {
		t.Errorf("expected to be at offset %d, is at %d ", firstPos, testPos)
	}

	// Seek to illegal offset
	if err = dec.SeekToFrame(-1, io.SeekStart); err != ErrIllegalSeek {
		t.Fatalf(`expected error "%v", got %v`, ErrIllegalSeek, err)
	}
	if err = dec.SeekToFrame(-1, io.SeekEnd); err != ErrIllegalSeek {
		t.Fatalf(`expected error "%v", got %v`, ErrIllegalSeek, err)
	}
}
