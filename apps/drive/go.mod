module github.com/magicbox/drive

go 1.25.7

require (
	github.com/google/uuid v1.6.0
	github.com/magicbox/core v0.0.0
	github.com/mattn/go-sqlite3 v1.14.46
	google.golang.org/grpc v1.81.1
)

require (
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/magicbox/core => ../../
