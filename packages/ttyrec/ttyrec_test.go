package ttyrec_test

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/x-qdo/qudosh/packages/ttyrec"
)

func ExampleNewDecoder() {
	f, err := os.Open("ttyrecord")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	d := ttyrec.NewDecoder(f)
	stream, _ := d.DecodeStream()

	var previous *ttyrec.Frame
	for frame := range stream {
		if previous != nil {
			time.Sleep(frame.Time.Sub(previous.Time))
		}
		os.Stdout.Write(frame.Data)
		previous = frame
	}
}

func ExampleNewEncoder() {
	f, err := os.Create("ttyrecord")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	e := ttyrec.NewEncoder(f)
	io.Copy(e, os.Stdout)
}

func TestTimeVal_Set(t *testing.T) {
	for _, test := range []struct {
		Test time.Duration
		Want ttyrec.TimeVal
	}{
		{0, ttyrec.TimeVal{}},
		{time.Microsecond, ttyrec.TimeVal{0, 1}},
		{time.Second, ttyrec.TimeVal{1, 0}},
		{time.Microsecond + time.Second, ttyrec.TimeVal{1, 1}},
		{9876543210 * time.Nanosecond, ttyrec.TimeVal{9, 876543}},
		{1234567890 * time.Microsecond, ttyrec.TimeVal{1234, 567890}},
		{-time.Minute, ttyrec.TimeVal{}},
	} {
		t.Run(test.Test.String(), func(t *testing.T) {
			var tv ttyrec.TimeVal
			tv.Set(test.Test)
			if tv.Seconds != test.Want.Seconds || tv.MicroSeconds != test.Want.MicroSeconds {
				t.Errorf("expected %v, got %v", test.Want, tv)
			}
		})
	}
}

func TestTimeVal_Sub(t *testing.T) {
	for _, test := range []struct {
		A, B ttyrec.TimeVal
		Want time.Duration
	}{
		{ttyrec.TimeVal{}, ttyrec.TimeVal{}, 0},
		{ttyrec.TimeVal{2, 1}, ttyrec.TimeVal{1, 1}, time.Second},
		{ttyrec.TimeVal{1234, 567890}, ttyrec.TimeVal{123, 456789}, 1111*time.Second + 111101*time.Microsecond},
	} {
		t.Run(test.Want.String(), func(t *testing.T) {
			if v := test.A.Sub(test.B); v != test.Want {
				t.Errorf("expected %v.Sub(%v) to be %s, got %s", test.A, test.B, test.Want, v)
			}
		})
	}
}
