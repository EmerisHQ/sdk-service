module github.com/allinbits/sdk-service-v42

go 1.16

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require goa.design/goa/v3 v3.5.2

require (
	github.com/allinbits/sdk-service-meta v0.0.0-20211006131905-2fae32a7a6f6
	github.com/cosmos/cosmos-sdk v0.42.8
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/tendermint/tendermint v0.34.11 // indirect
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.41.0
)
