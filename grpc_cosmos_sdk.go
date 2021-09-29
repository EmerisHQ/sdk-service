package sdkservicev42

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/tx"

	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"
)

var grpcPort = 9090

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

func GetTxFromHash(chainName string, port *int, hash string, cdc codec.Marshaler) ([]byte, error) {
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

	txClient := tx.NewServiceClient(grpcConn)

	grpcRes, err := txClient.GetTx(context.Background(), &tx.GetTxRequest{Hash: hash})
	if err != nil {
		return nil, err
	}

	return cdc.MarshalJSON(grpcRes)
}
