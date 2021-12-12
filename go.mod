module github.com/allinbits/sdk-service

go 1.16

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require goa.design/goa/v3 v3.5.3

require (
	github.com/allinbits/sdk-service-meta v0.0.0-20211212183412-eb82e7f68eed
	github.com/armon/go-metrics v0.3.9 // indirect
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/cosmos/cosmos-sdk v0.42.10
	github.com/cosmos/gaia/v5 v5.0.8
	github.com/golang/mock v1.6.0 // indirect
	github.com/gravity-devs/liquidity v1.2.9
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/lib/pq v1.10.2 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/onsi/ginkgo v1.16.4 // indirect
	github.com/onsi/gomega v1.13.0 // indirect
	github.com/prometheus/common v0.29.0 // indirect
	github.com/rs/zerolog v1.23.0 // indirect
	github.com/spf13/cobra v1.2.1 // indirect
	github.com/tendermint/tendermint v0.34.14
	go.uber.org/zap v1.19.1
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	google.golang.org/grpc v1.42.0
)
