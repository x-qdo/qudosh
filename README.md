# qudosh


qudosh is a tty proxy that records to ttyrec and uploads to S3. It runs a new 
process of the selected shell and proxies stdin and stdout.

## Installation

You can download precopiled binary from releases or use `go install`

```
go install github.com/x-qdo/qudosh@latest
```

## Usage

This will start a new process of the selected shell (by default, zsh) and proxy stdin and stdout. 
It will also record metrics every 10 seconds from stdin and stdout activity to a CSV file.

You can configure qudosh by setting the following environment variables:

* `QUDOSH_SHELL`: The shell to be executed under the hood (defaults to zsh).
* `LOCAL_PREFIX`: The location for recording files on the disk.
* `S3_BUCKET`: The bucket name to upload to.
* `S3_PREFIX`: The path inside the bucket.

## License

qudosh is licensed under the MIT license. Please see the LICENSE file for more information.

## Acknowledgments

qudosh was inspired by similar tools such as [https://github.com/KubeOperator/webkubectl](webkubectl), and [https://pkg.go.dev/maze.io/x/ttyrec](ttyrec). 
We thank the developers of these tools for their contributions to the open source community.
