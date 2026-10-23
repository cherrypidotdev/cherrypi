module cherrypi.vladfaust.com/agent

go 1.26.4

require cherrypi.vladfaust.com/shared v0.0.0

replace cherrypi.vladfaust.com/shared => ../shared

require (
	connectrpc.com/connect v1.20.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/coder/websocket v1.8.15
	github.com/creack/pty v1.1.24
	github.com/hashicorp/yamux v0.1.2
	github.com/joho/godotenv v1.5.1
	github.com/pressly/goose/v3 v3.28.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	modernc.org/libc v1.75.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

require (
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/sethvargo/go-retry v0.4.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	modernc.org/sqlite v1.59.0
)
