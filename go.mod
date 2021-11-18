module github.com/allinbits/sdk-service

go 1.16

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require goa.design/goa/v3 v3.5.2

require (
	github.com/99designs/keyring v1.1.6 // indirect
	github.com/allinbits/sdk-service-meta v0.0.0-20211118153638-07410dcb036a
	github.com/cosmos/cosmos-sdk v0.44.3
	github.com/cosmos/gaia/v6 v6.0.0-rc3
	github.com/cosmos/ibc-go v1.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/tendermint/liquidity v1.4.2
	github.com/tendermint/tendermint v0.34.14
	go.uber.org/zap v1.19.1
	google.golang.org/grpc v1.41.0
)
