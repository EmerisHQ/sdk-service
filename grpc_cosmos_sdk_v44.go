//go:build sdk_v44
// +build sdk_v44

package sdkservice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	junomint "github.com/CosmosContracts/juno/x/mint/types"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/types"
	mint "github.com/cosmos/cosmos-sdk/x/mint/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	gaia "github.com/cosmos/gaia/v6/app"
	crescentmint "github.com/crescent-network/crescent/x/mint/types"
	sdkutilities "github.com/emerishq/sdk-service-meta/gen/sdk_utilities"
	liquidity "github.com/gravity-devs/liquidity/x/liquidity/types"
	irismint "github.com/irisnet/irishub/modules/mint/types"
	gamm "github.com/osmosis-labs/osmosis/v7/x/gamm/types"
	osmomint "github.com/osmosis-labs/osmosis/v7/x/mint/types"
	budget "github.com/tendermint/budget/x/budget/types"
	"github.com/tendermint/tendermint/abci/types"
	"google.golang.org/grpc"
)

var (
	grpcPort             = 9090
	cdc      codec.Codec = nil
	cdcOnce  sync.Once
)

const (
	// TODO : this can be used used once relvant code was uncommented
	// transferMsgType = "transfer"

	junoChainName     = "juno"
	osmosisChainName  = "osmosis"
	irisChainName     = "iris"
	crescentChainName = "crescent"
)

func initCodec() {
	cfg := gaia.MakeEncodingConfig()
	cdc = cfg.Marshaler
}

func getCodec() codec.Codec {
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

	pagination := &sdkquery.PageRequest{}

	if paginationKey != nil {
		key, err := base64.StdEncoding.DecodeString(*paginationKey)
		if err == nil {
			pagination.Key = key
		}
	}

	suppRes, err := bankQuery.TotalSupply(ctx, &bank.QueryTotalSupplyRequest{Pagination: pagination})
	if err != nil {
		return sdkutilities.Supply2{}, err
	}

	ret := sdkutilities.Supply2{}

	var nextKey = base64.StdEncoding.EncodeToString(suppRes.Pagination.NextKey)
	var total = strconv.FormatUint(suppRes.Pagination.Total, 10)

	ret.Pagination = &sdkutilities.Pagination{
		NextKey: &nextKey,
		Total:   &total,
	}

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

	if err := getCodec().Unmarshal(txBytes, &txObj); err != nil {
		return sdkutilities.TxMessagesMetadata{}, fmt.Errorf("cannot unmarshal transaction, %w", err)
	}

	ret := sdkutilities.TxMessagesMetadata{}

	// Don't include ibc-go momentarily
	// TODO: reintroduce once terra fixes their stuff
	/*for idx, m := range txObj.GetMsgs() {
		txm := sdkutilities.MsgMetadata{}
		txm.MsgType = sdktypes.MsgTypeURL(m)

		switch txm.MsgType {
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

var mintFuncsMap = map[string]func(context.Context, *grpc.ClientConn) (sdkutilities.MintInflation2, error){
	junoChainName:     junoMintInflation,
	irisChainName:     irisMintInflation,
	osmosisChainName:  osmosisMintInflation,
	crescentChainName: crescentMintInflation,
}

func MintInflation(ctx context.Context, chainName string, port *int) (sdkutilities.MintInflation2, error) {
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

	if customMint, ok := mintFuncsMap[strings.ToLower(chainName)]; ok {
		return customMint(ctx, grpcConn)
	}
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

func junoMintInflation(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintInflation2, error) {
	mq := junomint.NewQueryClient(grpcConn)

	resp, err := mq.Inflation(ctx, &junomint.QueryInflationRequest{})
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

func irisMintInflation(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintInflation2, error) {
	iq := irismint.NewQueryClient(grpcConn)
	resp, err := iq.Params(ctx, &irismint.QueryParamsRequest{})
	if err != nil {
		return sdkutilities.MintInflation2{}, err
	}

	ret := sdkutilities.MintInflation2{
		MintInflation: []byte(fmt.Sprintf("{\"inflation\":\"%s\"}", resp.GetParams().Inflation.String())),
	}

	return ret, nil
}

func osmosisMintInflation(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintInflation2, error) {
	oq := osmomint.NewQueryClient(grpcConn)

	// inflation = (epochProvisions * reductionPeriodInEpochs) / supply

	epochProvResp, err := oq.EpochProvisions(ctx, &osmomint.QueryEpochProvisionsRequest{})
	if err != nil {
		return sdkutilities.MintInflation2{}, err
	}

	mintParamsResp, err := oq.Params(ctx, &osmomint.QueryParamsRequest{})
	if err != nil {
		return sdkutilities.MintInflation2{}, err
	}
	reductionPeriodInEpochs := mintParamsResp.GetParams().ReductionPeriodInEpochs

	bankQuery := bank.NewQueryClient(grpcConn)
	suppRes, err := bankQuery.SupplyOf(ctx, &bank.QuerySupplyOfRequest{Denom: mintParamsResp.GetParams().MintDenom})
	if err != nil {
		return sdkutilities.MintInflation2{}, err
	}
	supply := suppRes.GetAmount().Amount

	inflation := (epochProvResp.EpochProvisions.MulInt64(reductionPeriodInEpochs)).QuoInt(supply)
	ret := sdkutilities.MintInflation2{
		MintInflation: []byte(fmt.Sprintf("{\"inflation\":\"%f\"}", inflation)),
	}

	return ret, nil
}

func crescentMintInflation(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintInflation2, error) {
	cq := crescentmint.NewQueryClient(grpcConn)

	// inflation=current Inflation amount/total minted before schedule

	mintParamsResp, err := cq.Params(ctx, &crescentmint.QueryParamsRequest{})
	if err != nil {
		return sdkutilities.MintInflation2{}, err
	}

	now := time.Now()
	genesisSupply := sdktypes.NewInt(200000000000000)
	totalMintedBeforeSchedule := genesisSupply
	var currentInflationAmount sdktypes.Int

	for _, schedule := range mintParamsResp.GetParams().InflationSchedules {
		if schedule.StartTime.Before(now) && schedule.EndTime.Before(now) {
			totalMintedBeforeSchedule.Add(schedule.Amount)
		} else if schedule.StartTime.Before(now) && schedule.EndTime.After(now) {
			currentInflationAmount = schedule.Amount
		}
	}

	inflation := currentInflationAmount.Quo(totalMintedBeforeSchedule)

	ret := sdkutilities.MintInflation2{
		MintInflation: []byte(fmt.Sprintf("{\"inflation\":\"%f\"}", inflation)),
	}

	return ret, nil
}

var paramsFuncsMap = map[string]func(context.Context, *grpc.ClientConn) (sdkutilities.MintParams2, error){
	junoChainName:     junoMintParams,
	irisChainName:     irisMintParams,
	osmosisChainName:  osmosisMintParams,
	crescentChainName: crescentMintParams,
}

func MintParams(ctx context.Context, chainName string, port *int) (sdkutilities.MintParams2, error) {
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

	if customParams, ok := paramsFuncsMap[strings.ToLower(chainName)]; ok {
		return customParams(ctx, grpcConn)
	}

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

func junoMintParams(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintParams2, error) {
	mq := junomint.NewQueryClient(grpcConn)

	resp, err := mq.Params(ctx, &junomint.QueryParamsRequest{})
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

func irisMintParams(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintParams2, error) {
	iq := irismint.NewQueryClient(grpcConn)
	resp, err := iq.Params(ctx, &irismint.QueryParamsRequest{})
	if err != nil {
		return sdkutilities.MintParams2{}, err
	}

	respInterface := struct {
		Params irismint.Params `json:"params"`
	}{resp.GetParams()}
	respJSON, err := json.Marshal(respInterface)
	if err != nil {
		return sdkutilities.MintParams2{}, fmt.Errorf("cannot json marshal response from mint params, %w", err)
	}

	ret := sdkutilities.MintParams2{
		MintParams: respJSON,
	}

	return ret, nil
}

func osmosisMintParams(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintParams2, error) {
	oq := osmomint.NewQueryClient(grpcConn)
	resp, err := oq.Params(ctx, &osmomint.QueryParamsRequest{})
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

func crescentMintParams(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintParams2, error) {
	cq := crescentmint.NewQueryClient(grpcConn)
	resp, err := cq.Params(ctx, &crescentmint.QueryParamsRequest{})
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

var annualProvFuncsMap = map[string]func(context.Context, *grpc.ClientConn) (sdkutilities.MintAnnualProvision2, error){
	junoChainName:    junoMintAnnualProvisions,
	irisChainName:    irisMintAnnualProvisions,
	osmosisChainName: osmosisAnnualProvisions,
}

func MintAnnualProvision(ctx context.Context, chainName string, port *int) (sdkutilities.MintAnnualProvision2, error) {
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

	if customAnnualProv, ok := annualProvFuncsMap[strings.ToLower(chainName)]; ok {
		return customAnnualProv(ctx, grpcConn)
	}

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

func junoMintAnnualProvisions(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintAnnualProvision2, error) {
	mq := junomint.NewQueryClient(grpcConn)

	resp, err := mq.AnnualProvisions(ctx, &junomint.QueryAnnualProvisionsRequest{})
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

func irisMintAnnualProvisions(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintAnnualProvision2, error) {
	iq := irismint.NewQueryClient(grpcConn)
	resp, err := iq.Params(ctx, &irismint.QueryParamsRequest{})
	if err != nil {
		return sdkutilities.MintAnnualProvision2{}, err
	}

	// Welcome to the world of ugly code. How did I come up with this hack you may ask,
	// 1. The logic is taken from here: https://github.com/irisnet/irishub/blob/71503a902e193aecb8bce08b4a1a7dc0dc20c17c/modules/mint/types/minter.go#L45
	// 2. inflationBase is taken from here: https://github.com/irisnet/irishub/blob/71503a902e193aecb8bce08b4a1a7dc0dc20c17c/docs/features/mint.md
	// TODO: Tamjid - Fix when iris team exposes the annual_provision grpc endpoint!
	ap := resp.Params.Inflation.MulInt(sdktypes.NewIntWithDecimal(2000000000, 6))
	ret := sdkutilities.MintAnnualProvision2{
		MintAnnualProvision: []byte(fmt.Sprintf("{\"annual_provisions\":\"%s\"}", ap.String())),
	}

	return ret, nil
}

func osmosisAnnualProvisions(ctx context.Context, grpcConn *grpc.ClientConn) (sdkutilities.MintAnnualProvision2, error) {
	return sdkutilities.MintAnnualProvision2{
		MintAnnualProvision: nil,
	}, nil
}

func MintEpochProvisions(ctx context.Context, chainName string, port *int) (sdkutilities.MintEpochProvisions2, error) {
	if chainName != "osmosis" {
		return sdkutilities.MintEpochProvisions2{
			MintEpochProvisions: nil,
		}, nil
	}

	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.MintEpochProvisions2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	mq := osmomint.NewQueryClient(grpcConn)

	resp, err := mq.EpochProvisions(ctx, &osmomint.QueryEpochProvisionsRequest{})

	if err != nil {
		return sdkutilities.MintEpochProvisions2{}, err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.MintEpochProvisions2{}, fmt.Errorf("cannot json marshal response from mint epoch provision, %w", err)
	}

	ret := sdkutilities.MintEpochProvisions2{
		MintEpochProvisions: respJSON,
	}

	return ret, nil
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

	txSvcClient := sdktx.NewServiceClient(grpcConn)
	simRes, err := txSvcClient.Simulate(ctx, &sdktx.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return sdkutilities.Simulation{}, err
	}

	if chainName == "terra" {
		coins, err := computeTax(ctx, chainName, txBytes)
		if err != nil {
			return sdkutilities.Simulation{}, err
		}

		return sdkutilities.Simulation{
			GasWanted: simRes.GasInfo.GasWanted,
			GasUsed:   simRes.GasInfo.GasUsed,
			Fees:      coins,
		}, nil
	}

	return sdkutilities.Simulation{
		GasWanted: simRes.GasInfo.GasWanted,
		GasUsed:   simRes.GasInfo.GasUsed,
	}, nil
}

type computeTaxReq struct {
	TxBytes []byte `json:"tx_bytes"`
}

type computeTaxResp struct {
	TaxAmount []struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"tax_amount"`
}

func computeTax(ctx context.Context, endpointName string, txBytes []byte) ([]*sdkutilities.Coin, error) {
	// TODO(gsora): keeping this here until we have terra on ibc-go v2
	/*terraCli := terratx.NewServiceClient(grpcConn)
	taxRes, err := terraCli.ComputeTax(context.Background(), &terratx.ComputeTaxRequest{
		TxBytes: txBytes,
	})

	if err != nil {
		return nil, err
	}

	var coins []*sdkutilities.Coin
	for _, coin := range taxRes.TaxAmount {
		coins = append(coins, &sdkutilities.Coin{
			Denom:  coin.Denom,
			Amount: coin.Amount.String(),
		})
	}

	return coins, nil*/

	const path = "/terra/tx/v1beta1/compute_tax"
	u := url.URL{}
	u.Host = fmt.Sprintf("%v:1317", endpointName)
	u.Path = path
	u.Scheme = "http"

	payload, err := json.Marshal(computeTaxReq{
		TxBytes: txBytes,
	})

	if err != nil {
		return nil, fmt.Errorf("cannot json marshal terra computeTax request, %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("cannot create http request to terra computeTax, %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	c := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http terra computeTax request returned error, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http terra computeTax request returned with code %v", resp.Status)
	}

	dec := json.NewDecoder(resp.Body)
	defer func() {
		_ = resp.Body.Close()
	}()

	rawTax := computeTaxResp{}
	if err := dec.Decode(&rawTax); err != nil {
		return nil, fmt.Errorf("cannot decode terra computeTax response, %w", err)
	}

	coins := make([]*sdkutilities.Coin, len(rawTax.TaxAmount))

	for _, coin := range rawTax.TaxAmount {
		coins = append(coins, &sdkutilities.Coin{
			Denom:  coin.Denom,
			Amount: coin.Amount,
		})
	}

	return coins, nil
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
	if port == nil {
		port = &grpcPort
	}
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, *port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.BudgetParams2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	bc := budget.NewQueryClient(grpcConn)
	resp, err := bc.Params(ctx, &budget.QueryParamsRequest{})
	if err != nil {
		return sdkutilities.BudgetParams2{}, nil
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return sdkutilities.BudgetParams2{}, fmt.Errorf("cannot json marshal response from budget params, %w", err)
	}

	return sdkutilities.BudgetParams2{
		BudgetParams: respJSON,
	}, nil
}

func OsmoPools(ctx context.Context, chainName string, port *int) (sdkutilities.OsmoPools2, error) {
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%d", chainName, port), grpc.WithInsecure())
	if err != nil {
		return sdkutilities.OsmoPools2{}, err
	}

	defer func() {
		_ = grpcConn.Close()
	}()

	gq := gamm.NewQueryClient(grpcConn)

	numpoolsres, err := gq.NumPools(ctx, &gamm.QueryNumPoolsRequest{})
	if err != nil {
		return sdkutilities.OsmoPools2{}, fmt.Errorf("cannot get number of pools, %w", err)
	}

	fmt.Println(numpoolsres.NumPools)

	res, err := gq.Pools(ctx, &gamm.QueryPoolsRequest{
		Pagination: &sdkquery.PageRequest{
			Limit: numpoolsres.NumPools,
		},
	})
	if err != nil {
		return sdkutilities.OsmoPools2{}, fmt.Errorf("cannot get pools, %w", err)
	}

	out, err := cdc.MarshalJSON(res)
	if err != nil {
		return sdkutilities.OsmoPools2{}, fmt.Errorf("failed to marshal response, %w", err)
	}

	return sdkutilities.OsmoPools2{
		OsmoPools: out,
	}, nil
}
