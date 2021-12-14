module github.com/allinbits/sdk-service

go 1.16

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require goa.design/goa/v3 v3.5.3

require (
	github.com/allinbits/sdk-service-meta v0.0.0-20211213140844-1ad0f7cce207
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/cosmos/cosmos-sdk v0.44.5
	github.com/cosmos/gaia/v6 v6.0.0-rc3
	github.com/cosmos/ibc-go/v2 v2.0.0
	github.com/gravity-devs/liquidity v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/tendermint/tendermint v0.34.14
	github.com/terra-money/core v0.5.12
	go.uber.org/zap v1.19.1
	google.golang.org/grpc v1.42.0
)
