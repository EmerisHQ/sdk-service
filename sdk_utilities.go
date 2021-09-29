package sdkservicev42

import (
	"context"

	log "github.com/allinbits/sdk-service-meta/gen/log"

	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
)

// sdk-utilities service example implementation.
// The example methods log the requests and return zero values.
type sdkUtilitiessrvc struct {
	logger *log.Logger
}

// NewSdkUtilities returns the sdk-utilities service implementation.
func NewSdkUtilities(logger *log.Logger) sdkutilities.Service {
	return &sdkUtilitiessrvc{logger}
}

// Supply implements supply.
func (s *sdkUtilitiessrvc) Supply(
	ctx context.Context,
	p *sdkutilities.SupplyPayload,
) (res *sdkutilities.Supply2, err error) {
	ret, err := QuerySupply(p.ChainName, p.Port)
	if err != nil {
		return nil, err
	}

	res = &ret
	return
}
