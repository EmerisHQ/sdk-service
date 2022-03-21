module github.com/emerishq/sdk-service

go 1.16

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require (
    goa.design/goa/v3 v3.6.2
    github.com/emerishq/sdk-service-meta v0.0.0-20220321045904-ff7c06107a05 // indirect
)
