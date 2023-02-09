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
		stdinCounter := metrics.NewMeter()
		stdoutCounter := metrics.NewMeter()
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

			// write csv header
			fmt.Fprintf(
				metricsFile,
				"timestamp;stdin_delta;stdin_total;stdout_delta;stdout_total\n",
			)

			defer func() {
				// write the last line
				fmt.Fprintf(
					metricsFile,
					"%d;%d;%d;%d;%d\n",
					makeTimestamp(),
					0,
					stdinCounter.Count(),
					0,
					stdoutCounter.Count(),
				)

				stdinCounter.Stop()
				stdoutCounter.Stop()
				f.Close()
				metricsFile.Close()
			}()

			go func() {
				var stdinTotal int64 = 0
				var stdoutTotal int64 = 0

				writeLine := func() {
					stdinDelta := stdinCounter.Count() - stdinTotal
					stdinTotal = stdinCounter.Count()

					stdoutDelta := stdoutCounter.Count() - stdoutTotal
					stdoutTotal = stdoutCounter.Count()

					fmt.Fprintf(
						metricsFile,
						"%d;%d;%d;%d;%d\n",
						makeTimestamp(),
						stdinDelta,
						stdinTotal,
						stdoutDelta,
						stdoutTotal,
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
			KeystrokesMeter: stdinCounter,
			OutputMeter:     stdoutCounter,
			FileName:        fileName,
			FilePrefix:      filePrefix,
			logger:          ttyrec.NewEncoder(f),
			Hook:            finishedHandler,
			Cancel:          cancel,
		}

		return nil
	}
}
