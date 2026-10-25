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
	github.com/dustin/go-humanize v1.0.1 
	github.com/google/uuid v1.6.0 
	github.com/mattn/go-isatty v0.0.24 
	github.com/ncruces/go-strftime v1.0.0 
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec 
	golang.org/x/mod v0.41.0 
	golang.org/x/sys v0.48.0 
	modernc.org/libc v1.75.7 
	modernc.org/mathutil v1.7.1 
	modernc.org/memory v1.12.1 
)

require (
	github.com/mfridman/interpolate v0.0.2 
	github.com/sethvargo/go-retry v0.4.0 
	go.uber.org/multierr v1.11.0 
	golang.org/x/sync v0.23.0 
	modernc.org/sqlite v1.59.0
)
