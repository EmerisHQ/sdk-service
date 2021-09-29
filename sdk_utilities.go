package sdkservicev42

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"

	log "github.com/allinbits/sdk-service-meta/gen/log"

	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
)

// sdk-utilities service example implementation.
// The example methods log the requests and return zero values.
type sdkUtilitiessrvc struct {
	logger *log.Logger
	debug  bool
	cdc    codec.Marshaler
}

// NewSdkUtilities returns the sdk-utilities service implementation.
func NewSdkUtilities(logger *log.Logger, debug bool, cdc codec.Marshaler) sdkutilities.Service {
	return &sdkUtilitiessrvc{
		logger: logger,
		debug:  debug,
		cdc:    cdc,
	}
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

func (s *sdkUtilitiessrvc) QueryTx(ctx context.Context, payload *sdkutilities.QueryTxPayload) (res []byte, err error) {
	return GetTxFromHash(payload.ChainName, payload.Port, payload.Hash, s.cdc)
}
