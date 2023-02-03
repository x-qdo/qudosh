package main

import (
	"context"
	"fmt"
	"github.com/creack/pty"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/x-qdo/qudosh/packages/localcommand"
	"github.com/x-qdo/qudosh/packages/tty"
)

func main() {
	// TODO: Replace to asci art
	logrus.Info("Welcome to use HDO.")

	// TODO read from env with default
	shell := "zsh"
	ctx, cancel := context.WithCancel(context.Background())

	options := localcommand.Options{CloseSignal: 1}
	factory, err := localcommand.NewFactory(shell, nil, &options)
	if err != nil {
		exit(err, 3)
	}

	slave, err := factory.New(nil)
	defer slave.Close()

	// We need to make sure that we will read each symbol separately
	_, err = terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		exit(err, 1)
	}

	timeNow := time.Now().Format("2006_02_01_15_04_05")
	fileName := fmt.Sprintf("lab/session_%s.ttyrec", timeNow)

	proxyTTY, err := tty.New(
		os.Stdin,
		os.Stdout,
		slave,
		tty.WithPermitWrite(),
		tty.WithTtyRecording(os.Getenv("LOCAL_PREFIX"), fileName, saveFileHandler()),
	)

	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)

	go func() {
		// Send initial resize at start
		_ = resizeBasedOnCurrentShell(os.Stdin, proxyTTY.ResizeEvents)

		for {
			select {
			case <-sigwinch:
				_ = resizeBasedOnCurrentShell(os.Stdin, proxyTTY.ResizeEvents)
			}
		}
	}()

	errs := make(chan error, 1)
	go func() {
		errs <- proxyTTY.Run(ctx)
	}()
	err = waitSignals(errs, cancel)
	if err != nil && err != context.Canceled {
		exit(err, 8)
	}
}

func resizeBasedOnCurrentShell(stdin *os.File, resizeEvents chan *tty.ArgResizeTerminal) error {
	rows, cols, err := pty.Getsize(stdin)
	if err != nil {
		return err
	}
	initialSize := tty.ArgResizeTerminal{Columns: cols, Rows: rows}
	resizeEvents <- &initialSize
	return nil
}

func saveFileHandler() tty.Hook {
	return func(r *tty.Recorder) error {
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		save := func(postfix string) error {
			s3FileName := fmt.Sprintf("%s/%s%s", os.Getenv("S3_PREFIX"), r.FileName, postfix)
			fmt.Printf("Uploading to s3: %s\r\n", s3FileName)

			fileName := fmt.Sprintf("%s/%s%s", r.FilePrefix, r.FileName, postfix)
			file, err := os.Open(fileName)
			if err != nil {
				return err
			}
			defer file.Close()

			uploader := s3manager.NewUploaderWithClient(s3.New(sess, aws.NewConfig()))
			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket:               aws.String(os.Getenv("S3_BUCKET")),
				ACL:                  aws.String("private"),
				Key:                  aws.String(s3FileName),
				ServerSideEncryption: aws.String("AES256"),
				Body:                 file,
			})
			return err
		}

		err := save("")
		if err != nil {
			fmt.Printf("ERROR: Uploading ttyrec failed. %s \r\n", err)
			return err
		}

		err = save(".csv")
		if err != nil {
			fmt.Printf("ERROR: Uploading csv failed. %s \r\n", err)
			return err
		}

		return nil
	}
}

func exit(err error, code int) {
	if err != nil {
		fmt.Printf("Error: %s\r\n", err)
	}
	os.Exit(code)
}

func waitSignals(errs chan error, cancel context.CancelFunc) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(
		sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	select {
	case err := <-errs:
		return err

	case s := <-sigChan:
		switch s {
		case syscall.SIGINT:
			// gracefulCancel()
			fmt.Println("C-C to force close")
			select {
			case err := <-errs:
				return err
			case <-sigChan:
				fmt.Println("Force closing...")
				cancel()
				return <-errs
			}
		default:
			cancel()
			return <-errs
		}
	}
}
