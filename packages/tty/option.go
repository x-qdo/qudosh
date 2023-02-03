package tty

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"
	"github.com/x-qdo/qudosh/packages/ttyrec"
)

const MetricsInterval = 10 * time.Second

// Option is an option for WebTTY.
type Option func(*ProxyTTY) error

// WithPermitWrite sets a ProxyTTY to accept input from slaves.
func WithPermitWrite() Option {
	return func(ptty *ProxyTTY) error {
		ptty.permitWrite = true
		return nil
	}
}

func WithTtyRecording(filePrefix, fileName string, finishedHandler Hook) Option {
	return func(ptty *ProxyTTY) error {
		f, err := os.Create(fmt.Sprintf("%s/%s", filePrefix, fileName))
		if err != nil {
			log.Print(errors.Wrapf(err, "error opening %s: %v\n", fileName, err))
			return errors.Wrapf(err, "error opening %s: %v\n", fileName, err)
		}

		metricsFile, err := os.Create(fmt.Sprintf("%s/%s.csv", filePrefix, fileName))
		if err != nil {
			log.Print(errors.Wrapf(err, "error opening %s: %v\n", fileName, err))
			return errors.Wrapf(err, "error opening %s: %v\n", fileName, err)
		}

		m := metrics.NewMeter()
		go func() {
			var counter int64
			counter = 0

			fmt.Fprintf(
				metricsFile,
				"%s;%d;%d;%.2f;%.2f\n",
				time.Now().Format("2006-01-02T15:04:05"),
				counter,
				counter,
				m.Rate1(),
				m.RateMean(),
			)

			for range time.Tick(MetricsInterval) {
				delta := m.Count() - counter
				counter = m.Count()

				fmt.Fprintf(
					metricsFile,
					"%s;%d;%d;%.2f;%.2f\n",
					time.Now().Format("2006-01-02T15:04:05"),
					delta,
					counter,
					m.Rate1(),
					m.RateMean(),
				)
			}
		}()

		ptty.logger = &Recorder{
			MetricsFile:     metricsFile,
			KeystrokesMeter: m,
			File:            f,
			FileName:        fileName,
			FilePrefix:      filePrefix,
			logger:          ttyrec.NewEncoder(f),
			Hook:            finishedHandler,
		}

		return nil
	}
}
