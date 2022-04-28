module github.com/emerishq/sdk-service

go 1.16

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require (
	github.com/emerishq/sdk-service-meta v0.0.0-20220331063503-f6dcfa168e93
	github.com/tendermint/budget v1.1.1 // indirect
	goa.design/goa/v3 v3.6.2
)
