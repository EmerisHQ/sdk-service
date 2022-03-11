module github.com/emerishq/sdk-service

go 1.16

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require goa.design/goa/v3 v3.6.1

require (
	github.com/emerishq/sdk-service-meta v0.0.0-20220312063413-09a3229c4633 // indirect
	goa.design/goa/v3 v3.6.1
)
