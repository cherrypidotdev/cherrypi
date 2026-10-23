module cherrypi.vladfaust.com/shared

go 1.26.4

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	github.com/wasilibs/go-protoc-gen-es/v2/cmd/protoc-gen-es
	google.golang.org/protobuf/cmd/protoc-gen-go
)

require (
	connectrpc.com/connect v1.20.0
	google.golang.org/protobuf v1.36.12
)

require golang.org/x/mod v0.41.0

require (
	github.com/tetratelabs/wazero v1.12.0 // indirect
	github.com/wasilibs/go-protoc-gen-es/v2 v2.15.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)
