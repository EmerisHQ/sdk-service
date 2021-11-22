module github.com/allinbits/sdk-service

go 1.16

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)

require goa.design/goa/v3 v3.5.2

require (
	github.com/allinbits/sdk-service-meta v0.0.0-20211118153638-07410dcb036a
)
