module github.com/x-qdo/qudosh

go 1.19

replace gopkg.in/fsnotify.v1 => github.com/kolaente/fsnotify v1.4.10-0.20200411160148-1bc3c8ff4048

require (
	github.com/creack/pty v1.1.18
	github.com/pkg/errors v0.9.1
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/sirupsen/logrus v1.9.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
)

require golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
