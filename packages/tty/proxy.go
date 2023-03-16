package tty

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"
	"github.com/x-qdo/qudosh/packages/ttyrec"
)

type Hook func(r *Recorder) error

type Recorder struct {
	logger          *ttyrec.Encoder
	Hook            Hook
	FileName        string
	FilePrefix      string
	KeystrokesMeter metrics.Meter
	OutputMeter     metrics.Meter
	Cancel          context.CancelFunc
}

func (r Recorder) Write(data []byte) (int, error) {
	return r.logger.Write(data)
}

type ArgResizeTerminal struct {
	Columns int
	Rows    int
}

// ProxyTTY bridges a PTY slave and its PTY master.
// To support text-based streams and side channel commands such as
// terminal resizing, ProxyTTY uses an original protocol.
type ProxyTTY struct {
	// PTY Master
	masterStdin  io.Reader
	masterStdout io.Writer
	// PTY Slave
	slave Slave

	windowTitle []byte
	permitWrite bool
	columns     int
	rows        int

	bufferSize   int
	writeMutex   sync.Mutex
	lastPingTime time.Time
	logger       *Recorder

	ResizeEvents chan *ArgResizeTerminal
}

const (
	MaxBufferSize = 1024 * 1024 * 1
)

func New(masterStdin io.Reader, masterStdout io.Writer, slave Slave, options ...Option) (*ProxyTTY, error) {
	ptty := &ProxyTTY{
		masterStdin:  masterStdin,
		masterStdout: masterStdout,
		slave:        slave,
		logger:       nil,

		permitWrite: false,
		columns:     0,
		rows:        0,

		bufferSize:   MaxBufferSize,
		lastPingTime: time.Now(),
		ResizeEvents: make(chan *ArgResizeTerminal, 1),
	}

	for _, option := range options {
		err := option(ptty)
		if err != nil {
			return nil, err
		}
	}

	return ptty, nil
}

// Run starts the main process of the PProxyTTY
// This method blocks until the context is canceled.
// Note that the master and slave are left intact even
// after the context is canceled. Closing them is caller's
// responsibility.
// If the connection to one end gets closed, returns ErrSlaveClosed or ErrMasterClosed.
func (ptty *ProxyTTY) Run(ctx context.Context) error {
	var err error
	errs := make(chan error, 3)

	slaveBuffer := make([]byte, ptty.bufferSize)
	go func() {
		errs <- func() error {
			defer func() {
				if e := recover(); e != nil {
				}
			}()
			for {
				if slaveBuffer == nil {
					return ErrSlaveClosed
				}
				n, err := ptty.slave.Read(slaveBuffer)
				if err != nil {
					return ErrSlaveClosed
				}
				err = ptty.handleSlaveReadEvent(slaveBuffer[:n])
				if err != nil {
					return err
				}
			}
		}()
	}()
	masterBuffer := make([]byte, 4)
	bufferedStdin := bufio.NewReader(ptty.masterStdin)
	go func() {
		errs <- func() error {
			defer func() {
				if e := recover(); e != nil {
				}
			}()
			for {
				if masterBuffer == nil {
					return ErrMasterClosed
				}
				n, err := bufferedStdin.Read(masterBuffer)
				if err != nil {
					return ErrMasterClosed
				}
				err = ptty.handleMasterReadEvent(masterBuffer[:n])
				if err != nil {
					return err
				}
			}
		}()
	}()

	go func() {
		errs <- func() error {
			defer func() {
				if e := recover(); e != nil {
				}
			}()
			for {
				select {
				case newSize := <-ptty.ResizeEvents:
					err := ptty.slave.ResizeTerminal(newSize.Columns, newSize.Rows)
					if err != nil {
						return err
					}

					if ptty.logger != nil {
						_, err := ptty.logger.Write([]byte(fmt.Sprintf("\u001B[8;%d;%dt", newSize.Rows, newSize.Columns)))
						if err != nil {
							return err
						}
					}
				}
			}
		}()
	}()

	defer func() {
		slaveBuffer = nil
		masterBuffer = nil
		if ptty.logger != nil {

			// trigger cancel context
			ptty.logger.Cancel()

			if ptty.logger.Hook != nil {
				ptty.logger.Hook(ptty.logger)
			}
		}
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errs:
	}

	return err
}

func (ptty *ProxyTTY) handleSlaveReadEvent(data []byte) error {
	if ptty.logger != nil {
		ptty.logger.Write(data)
		ptty.logger.OutputMeter.Mark(int64(1))
	}
	err := ptty.masterWrite(data)
	if err != nil {
		return errors.Wrapf(err, "failed to send message to master")
	}

	return nil
}

func (ptty *ProxyTTY) masterWrite(data []byte) error {
	ptty.writeMutex.Lock()
	defer ptty.writeMutex.Unlock()

	_, err := ptty.masterStdout.Write(data)
	if err != nil {
		return errors.Wrapf(err, "failed to write to master")
	}

	return nil
}

func (ptty *ProxyTTY) handleMasterReadEvent(buf []byte) error {
	if !ptty.permitWrite {
		return nil
	}

	if ptty.logger != nil {
		ptty.logger.KeystrokesMeter.Mark(int64(1))
	}
	_, err := ptty.slave.Write(buf)
	if err != nil {
		return errors.Wrapf(err, "failed to write received data to slave")
	}

	return nil
}
