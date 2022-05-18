package sdkservice

import (
	"context"

	"github.com/emerishq/sdk-service-meta/gen/log"
	sdkutilities "github.com/emerishq/sdk-service-meta/gen/sdk_utilities"
)

// sdk-utilities service example implementation.
// The example methods log the requests and return zero values.
type sdkUtilitiessrvc struct {
	logger *log.Logger
	debug  bool
}

// NewSdkUtilities returns the sdk-utilities service implementation.
func NewSdkUtilities(logger *log.Logger, debug bool) sdkutilities.Service {
	return &sdkUtilitiessrvc{
		logger: logger,
		debug:  debug,
	}
}

// Supply implements supply.
func (s *sdkUtilitiessrvc) Supply(ctx context.Context, payload *sdkutilities.SupplyPayload) (res *sdkutilities.Supply2, err error) {
	ret, err := QuerySupply(ctx, payload.ChainName, payload.Port, payload.PaginationKey)
	if err != nil {
		return nil, err
	}

	res = &ret

	return
}

func (s *sdkUtilitiessrvc) SupplyDenom(ctx context.Context, payload *sdkutilities.SupplyDenomPayload) (res *sdkutilities.Supply2, err error) {
	return SupplyDenom(ctx, payload.ChainName, payload.Port, payload.Denom)
}

func (s *sdkUtilitiessrvc) QueryTx(ctx context.Context, payload *sdkutilities.QueryTxPayload) (res []byte, err error) {
	return GetTxFromHash(ctx, payload.ChainName, payload.Port, payload.Hash)
}

func (s *sdkUtilitiessrvc) BroadcastTx(ctx context.Context, payload *sdkutilities.BroadcastTxPayload) (res *sdkutilities.TransactionResult, err error) {
	txHash, txErr := BroadcastTx(
		ctx,
		payload.ChainName,
		payload.Port,
		payload.TxBytes,
	)

	if txErr != nil {
		err = txErr
		return
	}

	res = &sdkutilities.TransactionResult{
		Hash: txHash,
	}

	return
}

func (s *sdkUtilitiessrvc) TxMetadata(ctx context.Context, payload *sdkutilities.TxMetadataPayload) (res *sdkutilities.TxMessagesMetadata, err error) {
	var ret sdkutilities.TxMessagesMetadata
	ret, err = TxMetadata(ctx, payload.TxBytes)
	res = &ret
	return
}

func (s *sdkUtilitiessrvc) Block(ctx context.Context, payload *sdkutilities.BlockPayload) (res *sdkutilities.BlockData, err error) {
	ret, err := Block(ctx, payload.ChainName, payload.Port, payload.Height)
	return &ret, err
}

// LiquidityParams implements liquidityParams.
func (s *sdkUtilitiessrvc) LiquidityParams(ctx context.Context, payload *sdkutilities.LiquidityParamsPayload) (res *sdkutilities.LiquidityParams2, err error) {
	ret, err := LiquidityParams(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

// LiquidityPools implements liquidityPools.
func (s *sdkUtilitiessrvc) LiquidityPools(ctx context.Context, payload *sdkutilities.LiquidityPoolsPayload) (res *sdkutilities.LiquidityPools2, err error) {
	ret, err := LiquidityPools(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

// MintInflation implements mintInflation.
func (s *sdkUtilitiessrvc) MintInflation(ctx context.Context, payload *sdkutilities.MintInflationPayload) (res *sdkutilities.MintInflation2, err error) {
	ret, err := MintInflation(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

// MintParams implements mintParams.
func (s *sdkUtilitiessrvc) MintParams(ctx context.Context, payload *sdkutilities.MintParamsPayload) (res *sdkutilities.MintParams2, err error) {
	ret, err := MintParams(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

// MintAnnualProvision implements mintAnnualProvision.
func (s *sdkUtilitiessrvc) MintAnnualProvision(ctx context.Context, payload *sdkutilities.MintAnnualProvisionPayload) (res *sdkutilities.MintAnnualProvision2, err error) {
	ret, err := MintAnnualProvision(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

// MintEpochProvisions implements mintEpochProvisions.
func (s *sdkUtilitiessrvc) MintEpochProvisions(ctx context.Context, payload *sdkutilities.MintEpochProvisionsPayload) (res *sdkutilities.MintEpochProvisions2, err error) {
	ret, err := MintEpochProvisions(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

func (s *sdkUtilitiessrvc) AccountNumbers(ctx context.Context, payload *sdkutilities.AccountNumbersPayload) (res *sdkutilities.AccountNumbers2, err error) {
	ret, err := AccountNumbers(ctx, payload.ChainName, payload.Port, *payload.AddresHex, *payload.Bech32Prefix)
	return &ret, err
}

func (s *sdkUtilitiessrvc) DelegatorRewards(ctx context.Context, payload *sdkutilities.DelegatorRewardsPayload) (res *sdkutilities.DelegatorRewards2, err error) {
	ret, err := DelegatorRewards(ctx, payload.ChainName, payload.Port, *payload.AddresHex, *payload.Bech32Prefix)
	return &ret, err
}

func (s *sdkUtilitiessrvc) EstimateFees(ctx context.Context, payload *sdkutilities.EstimateFeesPayload) (res *sdkutilities.Simulation, err error) {
	ret, err := FeeEstimate(ctx, payload.ChainName, payload.Port, payload.TxBytes)
	return &ret, err
}

func (s *sdkUtilitiessrvc) StakingParams(ctx context.Context, payload *sdkutilities.StakingParamsPayload) (*sdkutilities.StakingParams2, error) {
	ret, err := StakingParams(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

func (s *sdkUtilitiessrvc) StakingPool(ctx context.Context, payload *sdkutilities.StakingPoolPayload) (*sdkutilities.StakingPool2, error) {
	ret, err := StakingPool(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

func (s *sdkUtilitiessrvc) EmoneyInflation(ctx context.Context, payload *sdkutilities.EmoneyInflationPayload) (*sdkutilities.EmoneyInflation2, error) {
	// ret, err := EmoneyInflation(payload.ChainName, payload.Port)
	return &sdkutilities.EmoneyInflation2{}, nil
}

func (s *sdkUtilitiessrvc) BudgetParams(ctx context.Context, payload *sdkutilities.BudgetParamsPayload) (*sdkutilities.BudgetParams2, error) {
	ret, err := BudgetParams(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

func (s *sdkUtilitiessrvc) DistributionParams(ctx context.Context, payload *sdkutilities.DistributionParamsPayload) (*sdkutilities.DistributionParams2, error) {
	ret, err := DistributionParams(ctx, payload.ChainName, payload.Port)
	return &ret, err
}

func (s *sdkUtilitiessrvc) OsmoPools(ctx context.Context, payload *sdkutilities.OsmoPoolsPayload) (*sdkutilities.OsmoPools2, error) {
	ret, err := OsmoPools(ctx, payload.ChainName, payload.Port)
	return &ret, err
}
