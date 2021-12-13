//go:build sdk_v44

package sdkservice

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/types"
	mint "github.com/cosmos/cosmos-sdk/x/mint/types"
	gaia "github.com/cosmos/gaia/v6/app"
	ibcTypes "github.com/cosmos/ibc-go/v2/modules/apps/transfer/types"
	liquidity "github.com/gravity-devs/liquidity/x/liquidity/types"
	"github.com/tendermint/tendermint/abci/types"
	"google.golang.org/grpc"
)

var (
	grpcPort             = 9090
	cdc      codec.Codec = nil
	cdcOnce  sync.Once
)

const (
	transferMsgType = "transfer"
)

func initCodec() {
	cfg := gaia.MakeEncodingConfig()
	cdc = cfg.Marshaler
}

func getCodec() codec.Codec {
	cdcOnce.Do(initCodec)

	return cdc
}

func QuerySupply(chainName string, port *int) (sdkutilities.Supply2, error) {
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

	suppRes, err := bankQuery.TotalSupply(context.Background(), &bank.QueryTotalSupplyRequest{})
	if err != nil {
		return sdkutilities.Supply2{}, err
	}

	ret := sdkutilities.Supply2{}

	for _, s := range suppRes.Supply {
		ret.Coins = append(ret.Coins, &sdkutilities.Coin{
			Denom:  s.Denom,
			Amount: s.Amount.String(),
		})
	}

	return ret, nil
}

func GetTxFromHash(chainName string, port *int, hash string) ([]byte, error) {
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

	grpcRes, err := txClient.GetTx(context.Background(), &sdktx.GetTxRequest{Hash: hash})
	if err != nil {
		return nil, err
	}

	return getCodec().MarshalJSON(grpcRes)
}

func BroadcastTx(chainName string, port *int, txBytes []byte) (string, error) {
	if port == nil {
		port = &grpcPort
	}

	grpcConn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", chainName, port), // Or your gRPC server address.
		grpc.WithInsecure(),                   // The SDK doesn't support any transport security mechanism.
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
		context.Background(),
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

func TxMetadata(txBytes []byte) (sdkutilities.TxMessagesMetadata, error) {
	txObj := sdktx.Tx{}

	if err := getCodec().Unmarshal(txBytes, &txObj); err != nil {
		return sdkutilities.TxMessagesMetadata{}, fmt.Errorf("cannot unmarshal transaction, %w", err)
	}

	ret := sdkutilities.TxMessagesMetadata{}

	for idx, m := range txObj.GetMsgs() {
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
	}

	return ret, nil
}

func Block(chainName string, port *int, height int64) (sdkutilities.BlockData, error) {
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
	resp, err := sc.GetBlockByHeight(context.Background(), &tmservice.GetBlockByHeightRequest{
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

func LiquidityParams(chainName string, port *int) (sdkutilities.LiquidityParams2, error) {
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

	resp, err := lq.Params(context.Background(), &liquidity.QueryParamsRequest{})

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

func LiquidityPools(chainName string, port *int) (sdkutilities.LiquidityPools2, error) {
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

	resp, err := lq.LiquidityPools(context.Background(), &liquidity.QueryLiquidityPoolsRequest{})

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

func MintInflation(chainName string, port *int) (sdkutilities.MintInflation2, error) {
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

	resp, err := mq.Inflation(context.Background(), &mint.QueryInflationRequest{})

	if err != nil {
		return sdkutilities.MintInflation2{}, nil
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

func MintParams(chainName string, port *int) (sdkutilities.MintParams2, error) {
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

	resp, err := mq.Params(context.Background(), &mint.QueryParamsRequest{})

	if err != nil {
		return sdkutilities.MintParams2{}, nil
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

func MintAnnualProvision(chainName string, port *int) (sdkutilities.MintAnnualProvision2, error) {
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

	resp, err := mq.AnnualProvisions(context.Background(), &mint.QueryAnnualProvisionsRequest{})

	if err != nil {
		return sdkutilities.MintAnnualProvision2{}, nil
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

func AccountNumbers(chainName string, port *int, hexAddress string, bech32hrp string) (sdkutilities.AccountNumbers2, error) {
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

	res, err := authQuery.Account(context.Background(), &auth.QueryAccountRequest{
		Address: addr,
	})
	if err != nil {
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

func DelegatorRewards(chainName string, port *int, hexAddress string, bech32hrp string) (sdkutilities.DelegatorRewards2, error) {
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

	res, err := distributionQuery.DelegationTotalRewards(context.Background(), &distribution.QueryDelegationTotalRewardsRequest{
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

func sdkDecCoinToUtilCoin(c sdktypes.DecCoin) *sdkutilities.Coin {
	return &sdkutilities.Coin{
		Denom:  c.Denom,
		Amount: c.Amount.String(),
	}
}
