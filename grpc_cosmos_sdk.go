package sdkservicev42

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/tx"

	ibcTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"

	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"
)

var grpcPort = 9090

const (
	transferMsgType = "transfer"
)

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

func TxMetadata(txBytes []byte, cdc codec.Marshaler) (sdkutilities.TxMessagesMetadata, error) {
	txObj := sdktx.Tx{}

	if err := cdc.UnmarshalBinaryBare(txBytes, &txObj); err != nil {
		return sdkutilities.TxMessagesMetadata{}, fmt.Errorf("cannot unmarshal transaction, %w", err)
	}

	ret := sdkutilities.TxMessagesMetadata{}

	for idx, m := range txObj.GetMsgs() {
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
	}

	return ret, nil
}
