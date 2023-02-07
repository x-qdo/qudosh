package tty

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"
	"github.com/x-qdo/qudosh/packages/ttyrec"
)

const MetricsInterval = 10 * time.Second

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// Option is an option for WebTTY.
type Option func(*ProxyTTY) error

// WithPermitWrite sets a ProxyTTY to accept input from slaves.
func WithPermitWrite() Option {
	return func(ptty *ProxyTTY) error {
		ptty.permitWrite = true
		return nil
	}
}

func WithTtyRecording(parent context.Context, filePrefix, fileName string, finishedHandler Hook) Option {
	return func(ptty *ProxyTTY) error {
		m := metrics.NewMeter()
		f, err := os.Create(fmt.Sprintf("%s/%s", filePrefix, fileName))
		if err != nil {
			log.Print(errors.Wrapf(err, "error opening %s: %v\n", fileName, err))
			return errors.Wrapf(err, "error opening %s: %v\n", fileName, err)
		}

		ctx, cancel := context.WithCancel(parent)

		go func() error {
			metricsFile, err := os.Create(fmt.Sprintf("%s/%s.csv", filePrefix, fileName))
			if err != nil {
				log.Print(errors.Wrapf(err, "error opening %s: %v\n", fileName, err))
				return errors.Wrapf(err, "error opening %s: %v\n", fileName, err)
			}

			defer func() {
				// write the last line
				fmt.Fprintf(
					metricsFile,
					"%d;%d;%d\n",
					makeTimestamp(),
					0,
					m.Count(),
				)

				m.Stop()
				f.Close()
				metricsFile.Close()
			}()

			go func() {
				var counter int64
				counter = 0

				writeLine := func() {
					delta := m.Count() - counter
					counter = m.Count()

					fmt.Fprintf(
						metricsFile,
						"%d;%d;%d\n",
						makeTimestamp(),
						delta,
						counter,
					)
				}

				// write the first line
				writeLine()

				for range time.Tick(MetricsInterval) {
					writeLine()
				}
			}()

			select {
			case <-ctx.Done():
				return ctx.Err()
			}
		}()

		ptty.logger = &Recorder{
			KeystrokesMeter: m,
			FileName:        fileName,
			FilePrefix:      filePrefix,
			logger:          ttyrec.NewEncoder(f),
			Hook:            finishedHandler,
			Cancel:          cancel,
		}

		return nil
	}
}
