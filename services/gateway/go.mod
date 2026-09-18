module github.com/alesplll/opens3-rebac/services/gateway

go 1.24.1

require (
	github.com/alesplll/opens3-rebac/shared v0.0.0-20260401221801-a711779ac92c
	google.golang.org/grpc v1.76.0
)

require (
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace github.com/alesplll/opens3-rebac/shared => ../../shared
