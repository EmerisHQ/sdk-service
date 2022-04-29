//go:build sdk_v42
// +build sdk_v42

package sdkservice

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	staking "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	emoneyinflation "github.com/e-money/em-ledger/x/inflation/types"
	liquidity "github.com/gravity-devs/liquidity/x/liquidity/types"
	"github.com/tendermint/tendermint/abci/types"

	mint "github.com/cosmos/cosmos-sdk/x/mint/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/types"
	sdkutilities "github.com/emerishq/sdk-service-meta/gen/sdk_utilities"

	gaia "github.com/cosmos/gaia/v3/app"
	"google.golang.org/grpc"
)

var (
	grpcPort                 = 9090
	cdc      codec.Marshaler = nil
	cdcOnce  sync.Once
)

const (
	// TODO : this can be used used once relvant code was uncommented
	// transferMsgType = "transfer"

	emoneyChainName = "emoney"
)

func initCodec() {
	c, _ := gaia.MakeCodecs()
	cdc = c
}

func getCodec() codec.Marshaler {
	cdcOnce.Do(initCodec)

	return cdc
}

func QuerySupply(ctx context.Context, chainName string, port *int, paginationKey *string) (sdkutilities.Supply2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.Supply2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	bankQuery := bank.NewQueryClient(grpcConn)

	suppRes, err := bankQuery.TotalSupply(ctx, &bank.QueryTotalSupplyRequest{})
	if err != nil {
		return sdkutilities.Supply2{}, err
	}

	ret := sdkutilities.Supply2{}

	ret.Pagination = &sdkutilities.Pagination{}

	for _, s := range suppRes.Supply {
		ret.Coins = append(ret.Coins, &sdkutilities.Coin{
			Denom:  s.Denom,
			Amount: s.Amount.String(),
		})
	}

	return ret, nil
}

func SupplyDenom(ctx context.Context, chainName string, port *int, denom *string) (*sdkutilities.Supply2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return &sdkutilities.Supply2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	bankQuery := bank.NewQueryClient(grpcConn)
	suppRes, err := bankQuery.SupplyOf(ctx, &bank.QuerySupplyOfRequest{Denom: *denom})
	if err != nil {
		return &sdkutilities.Supply2{}, err
	}

	ret := sdkutilities.Supply2{Coins: []*sdkutilities.Coin{{Denom: *denom, Amount: suppRes.Amount.String()}}}

	return &ret, nil
}

func GetTxFromHash(ctx context.Context, chainName string, port *int, hash string) ([]byte, error) {
	if port == nil {
		port = &grpcPort
	}

	grpcConn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", chainName, *port),
		grpc.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	txClient := sdktx.NewServiceClient(grpcConn)

	grpcRes, err := txClient.GetTx(ctx, &sdktx.GetTxRequest{Hash: hash})
	if err != nil {
		return nil, err
	}

	return getCodec().MarshalJSON(grpcRes)
}

func BroadcastTx(ctx context.Context, chainName string, port *int, txBytes []byte) (string, error) {
	if port == nil {
		port = &grpcPort
	}

	grpcConn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", chainName, *port), // Or your gRPC server address.
		grpc.WithInsecure(),                    // The SDK doesn't support any transport security mechanism.
	)

	if err != nil {
		return "", fmt.Errorf("cannot create grpc dialer, %w", err)
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	txClient := sdktx.NewServiceClient(grpcConn)
	// We then call the BroadcastTx method on this client.
	grpcRes, err := txClient.BroadcastTx(
		ctx,
		&sdktx.BroadcastTxRequest{
			Mode:    sdktx.BroadcastMode_BROADCAST_MODE_SYNC,
			TxBytes: txBytes, // Proto-binary of the signed transaction, see previous step.
		},
	)

	if err != nil {
		return "", err
	}

	if grpcRes.TxResponse.Code != types.CodeTypeOK {
		return "", fmt.Errorf("transaction relaying error: code %d, %s", grpcRes.TxResponse.Code, grpcRes.TxResponse.RawLog)
	}

	return grpcRes.TxResponse.TxHash, nil
}

func TxMetadata(ctx context.Context, txBytes []byte) (sdkutilities.TxMessagesMetadata, error) {
	txObj := sdktx.Tx{}

	if err := getCodec().UnmarshalBinaryBare(txBytes, &txObj); err != nil {
		return sdkutilities.TxMessagesMetadata{}, fmt.Errorf("cannot unmarshal transaction, %w", err)
	}

	ret := sdkutilities.TxMessagesMetadata{}

	// Don't include ibc-go momentarily even though v42 isn't affected,
	// for consistency reasons.
	// TODO: reintroduce once terra fixes their stuff
	/*for idx, m := range txObj.GetMsgs() {
		txm := sdkutilities.MsgMetadata{}
		txm.MsgType = m.Type()

		switch m.Type() {
		case transferMsgType:
			mt, ok := m.(*ibcTypes.MsgTransfer)
			if !ok {
				return sdkutilities.TxMessagesMetadata{}, fmt.Errorf("transaction message %d: expected MsgTransfer, got %T", idx, m)
			}

			it := sdkutilities.IBCTransferMetadata{
				SourcePort:    &mt.SourcePort,
				SourceChannel: &mt.SourceChannel,
				Token: &sdkutilities.Coin{
					Denom:  mt.Token.Denom,
					Amount: mt.Token.Amount.String(),
				},
				Sender:   &mt.Sender,
				Receiver: &mt.Receiver,
				TimeoutHeight: &sdkutilities.IBCHeight{
					RevisionNumber: &mt.TimeoutHeight.RevisionNumber,
					RevisionHeight: &mt.TimeoutHeight.RevisionHeight,
				},
				TiemoutTimestamp: &mt.TimeoutTimestamp,
			}

			txm.IbcTransferMetadata = &it
		}
	}*/

	return ret, nil
}

func Block(ctx context.Context, chainName string, port *int, height int64) (sdkutilities.BlockData, error) {
	if port == nil {
		port = &grpcPort
	}

	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.BlockData{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	sc := tmservice.NewServiceClient(grpcConn)
	resp, err := sc.GetBlockByHeight(ctx, &tmservice.GetBlockByHeightRequest{
		Height: height,
	})

	if err != nil {
		return sdkutilities.BlockData{}, err
	}

	ret := sdkutilities.BlockData{}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.BlockData{}, fmt.Errorf("cannot json marshal response from block height, %w", err)
	}

	ret.Height = height
	ret.Block = respJSON

	return ret, nil
}

func LiquidityParams(ctx context.Context, chainName string, port *int) (sdkutilities.LiquidityParams2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.LiquidityParams2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	lq := liquidity.NewQueryClient(grpcConn)

	resp, err := lq.Params(ctx, &liquidity.QueryParamsRequest{})

	if err != nil {
		return sdkutilities.LiquidityParams2{}, nil
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.LiquidityParams2{}, fmt.Errorf("cannot json marshal response from liquidity params, %w", err)
	}

	ret := sdkutilities.LiquidityParams2{
		LiquidityParams: respJSON,
	}

	return ret, nil
}

func LiquidityPools(ctx context.Context, chainName string, port *int) (sdkutilities.LiquidityPools2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.LiquidityPools2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	lq := liquidity.NewQueryClient(grpcConn)

	resp, err := lq.LiquidityPools(ctx, &liquidity.QueryLiquidityPoolsRequest{})

	if err != nil {
		return sdkutilities.LiquidityPools2{}, nil
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.LiquidityPools2{}, fmt.Errorf("cannot json marshal response from liquidity pools, %w", err)
	}

	ret := sdkutilities.LiquidityPools2{
		LiquidityPools: respJSON,
	}

	return ret, nil
}

func MintInflation(ctx context.Context, chainName string, port *int) (sdkutilities.MintInflation2, error) {
	if chainName == emoneyChainName {
		// emoney inflation is different from the traditional cosmos sdk inflation,
		// and does not have an annualprovisions endpoint. Instead it uses a flat inflation
		// rate provided in the endpoint.
		return emoneyInflation(ctx, chainName, port)
	}

	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.MintInflation2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	mq := mint.NewQueryClient(grpcConn)

	resp, err := mq.Inflation(ctx, &mint.QueryInflationRequest{})

	if err != nil {
		return sdkutilities.MintInflation2{}, err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.MintInflation2{}, fmt.Errorf("cannot json marshal response from mint inflation, %w", err)
	}

	ret := sdkutilities.MintInflation2{
		MintInflation: respJSON,
	}

	return ret, nil
}

func MintParams(ctx context.Context, chainName string, port *int) (sdkutilities.MintParams2, error) {
	if chainName == emoneyChainName {
		// emoney inflation is different from the traditional cosmos sdk inflation,
		// and does not have an annualprovisions endpoint. Instead it uses a flat inflation
		// rate provided in the endpoint.
		return sdkutilities.MintParams2{}, nil
	}
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.MintParams2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	mq := mint.NewQueryClient(grpcConn)

	resp, err := mq.Params(ctx, &mint.QueryParamsRequest{})

	if err != nil {
		return sdkutilities.MintParams2{}, err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.MintParams2{}, fmt.Errorf("cannot json marshal response from mint params, %w", err)
	}

	ret := sdkutilities.MintParams2{
		MintParams: respJSON,
	}

	return ret, nil
}

func MintAnnualProvision(ctx context.Context, chainName string, port *int) (sdkutilities.MintAnnualProvision2, error) {
	if chainName == emoneyChainName {
		// emoney inflation is different from the traditional cosmos sdk inflation,
		// and does not have an annualprovisions endpoint. Instead it uses a flat inflation
		// rate provided in the endpoint.
		return sdkutilities.MintAnnualProvision2{}, nil
	}
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.MintAnnualProvision2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	mq := mint.NewQueryClient(grpcConn)

	resp, err := mq.AnnualProvisions(ctx, &mint.QueryAnnualProvisionsRequest{})

	if err != nil {
		return sdkutilities.MintAnnualProvision2{}, err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.MintAnnualProvision2{}, fmt.Errorf("cannot json marshal response from mint annual provision, %w", err)
	}

	ret := sdkutilities.MintAnnualProvision2{
		MintAnnualProvision: respJSON,
	}

	return ret, nil
}

func MintEpochProvisions(ctx context.Context, chainName string, port *int) (sdkutilities.MintEpochProvisions2, error) {
	return sdkutilities.MintEpochProvisions2{
		MintEpochProvisions: nil,
	}, nil
}

func AccountNumbers(ctx context.Context, chainName string, port *int, hexAddress string, bech32hrp string) (sdkutilities.AccountNumbers2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.AccountNumbers2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	addrBytes, err := hex.DecodeString(hexAddress)
	if err != nil {
		return sdkutilities.AccountNumbers2{}, err
	}

	addr, err := bech32.ConvertAndEncode(bech32hrp, addrBytes)
	if err != nil {
		return sdkutilities.AccountNumbers2{}, err
	}

	authQuery := auth.NewQueryClient(grpcConn)

	res, err := authQuery.Account(ctx, &auth.QueryAccountRequest{
		Address: addr,
	})

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return sdkutilities.AccountNumbers2{}, nil
		}

		return sdkutilities.AccountNumbers2{}, err
	}

	ret := sdkutilities.AccountNumbers2{}

	if res == nil {
		return ret, fmt.Errorf("account has no numbers associated")
	}

	// get a baseAccount
	var accountI auth.AccountI

	if err := getCodec().UnpackAny(res.Account, &accountI); err != nil {
		return sdkutilities.AccountNumbers2{}, err
	}

	ret.AccountNumber = int64(accountI.GetAccountNumber())
	ret.SequenceNumber = int64(accountI.GetSequence())
	ret.Bech32Address = addr

	return ret, nil
}

func DelegatorRewards(ctx context.Context, chainName string, port *int, hexAddress string, bech32hrp string) (sdkutilities.DelegatorRewards2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.DelegatorRewards2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	addrBytes, err := hex.DecodeString(hexAddress)
	if err != nil {
		return sdkutilities.DelegatorRewards2{}, err
	}

	addr, err := bech32.ConvertAndEncode(bech32hrp, addrBytes)
	if err != nil {
		return sdkutilities.DelegatorRewards2{}, err
	}

	distributionQuery := distribution.NewQueryClient(grpcConn)

	res, err := distributionQuery.DelegationTotalRewards(ctx, &distribution.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: addr,
	})

	if err != nil {
		return sdkutilities.DelegatorRewards2{}, err
	}

	ret := sdkutilities.DelegatorRewards2{}

	for _, d := range res.Rewards {
		r := &sdkutilities.DelegationDelegatorReward{
			ValidatorAddress: d.ValidatorAddress,
		}

		for _, rr := range d.Reward {
			r.Rewards = append(r.Rewards, sdkDecCoinToUtilCoin(rr))
		}

		ret.Rewards = append(ret.Rewards, r)
	}

	for _, d := range res.Total {
		ret.Total = append(ret.Total, sdkDecCoinToUtilCoin(d))
	}

	return ret, nil
}

func FeeEstimate(ctx context.Context, chainName string, port *int, txBytes []byte) (sdkutilities.Simulation, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.Simulation{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	txObj := &sdktx.Tx{}

	if err := getCodec().UnmarshalBinaryBare(txBytes, txObj); err != nil {
		return sdkutilities.Simulation{}, fmt.Errorf("cannot unmarshal transaction, %w", err)
	}

	txSvcClient := sdktx.NewServiceClient(grpcConn)
	simRes, err := txSvcClient.Simulate(ctx, &sdktx.SimulateRequest{
		Tx: txObj,
	})
	if err != nil {
		return sdkutilities.Simulation{}, err
	}

	return sdkutilities.Simulation{
		GasWanted: simRes.GasInfo.GasWanted,
		GasUsed:   simRes.GasInfo.GasUsed,
	}, nil

}

func sdkDecCoinToUtilCoin(c sdktypes.DecCoin) *sdkutilities.Coin {
	return &sdkutilities.Coin{
		Denom:  c.Denom,
		Amount: c.Amount.String(),
	}
}

func StakingParams(ctx context.Context, chainName string, port *int) (sdkutilities.StakingParams2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.StakingParams2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	sq := staking.NewQueryClient(grpcConn)
	resp, err := sq.Params(ctx, &staking.QueryParamsRequest{})
	if err != nil {
		return sdkutilities.StakingParams2{}, nil
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.StakingParams2{}, fmt.Errorf("cannot json marshal response from staking params, %w", err)
	}

	return sdkutilities.StakingParams2{
		StakingParams: respJSON,
	}, nil
}

func StakingPool(ctx context.Context, chainName string, port *int) (sdkutilities.StakingPool2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.StakingPool2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	sq := staking.NewQueryClient(grpcConn)
	resp, err := sq.Pool(ctx, &staking.QueryPoolRequest{})
	if err != nil {
		return sdkutilities.StakingPool2{}, nil
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.StakingPool2{}, fmt.Errorf("cannot json marshal response from staking pool, %w", err)
	}

	return sdkutilities.StakingPool2{
		StakingPool: respJSON,
	}, nil
}

func emoneyInflation(ctx context.Context, chainName string, port *int) (sdkutilities.MintInflation2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.MintInflation2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	emc := emoneyinflation.NewQueryClient(grpcConn)
	resp, err := emc.Inflation(ctx, &emoneyinflation.QueryInflationRequest{})
	if err != nil {
		return sdkutilities.MintInflation2{}, nil
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.MintInflation2{}, fmt.Errorf("cannot json marshal response from emoney inflation, %w", err)
	}

	var ret sdkutilities.MintInflation2
	var data sdkutilities.EmoneyInflation2
	if err := json.Unmarshal(respJSON, &data); err != nil {
		return sdkutilities.MintInflation2{}, fmt.Errorf("cannot json marshal response from mint inflation, %w", err)
	}

	for _, v := range data.State.Assets {
		if v.Denom == "ungm" {
			ret.MintInflation = []byte(fmt.Sprintf("{\"inflation\":\"%s\"}", v.Inflation))
		}
	}

	return ret, nil
}

func DistributionParams(ctx context.Context, chainName string, port *int) (sdkutilities.DistributionParams2, error) {
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.DistributionParams2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	dc := distribution.NewQueryClient(grpcConn)
	resp, err := dc.Params(ctx, &distribution.QueryParamsRequest{})
	if err != nil {
		return sdkutilities.DistributionParams2{}, nil
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.DistributionParams2{}, fmt.Errorf("cannot json marshal response from distribution params, %w", err)
	}

	return sdkutilities.DistributionParams2{
		DistributionParams: respJSON,
	}, nil
}

func BudgetParams(ctx context.Context, chainName string, port *int) (sdkutilities.BudgetParams2, error) {
	return sdkutilities.BudgetParams2{}, fmt.Errorf("Cannont get budget params from sdk")
}
