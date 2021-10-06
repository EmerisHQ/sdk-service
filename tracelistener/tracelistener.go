package tracelistener

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/allinbits/sdk-service-meta/gen/log"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	tracemeta "github.com/allinbits/sdk-service-meta/tracelistener"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	connectionTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	ibcchanneltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	tmIBCTypes "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/07-tendermint/types"
	delegationtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type Processor struct {
	cdc codec.Marshaler
	l   *log.Logger
}

func NewProcessor(
	cdc codec.Marshaler,
	logger *log.Logger,
) Processor {
	return Processor{
		cdc: cdc,
		l:   logger,
	}
}

// Rule of thumb: for each entry that returns error, add a error string to the sdkutilities.ProcessingError Errors slice.
// Each handling function must declare a sdkutilities.ProcessingError struct, and defer the assign assign statement of
// a pointer to said variable to err.
// No `return' statement must be executed on error handling, only `continue'.

func (p Processor) AuthEndpoint(ctx context.Context, payload *sdkutilities.AuthPayload) (res []*sdkutilities.Auth, err error) {
	perrs := newPe()
	defer func() {
		if perrs.Errors != nil {
			err = &perrs
		}
	}()

	for idx, pl := range payload.Payload {
		p.l.Debugw("auth processor entered", "key", string(pl.Key), "value", string(pl.Value))
		if len(pl.Key) != sdk.AddrLen+1 {
			p.l.Debugw("auth got key that isn't supposed to")
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        "cannot process key, not of auth type",
				PayloadIndex: idx,
			})
			// key len must be len(account bytes) + 1
			continue
		}

		var acc authTypes.AccountI

		if err := p.cdc.UnmarshalInterface(pl.Value, &acc); err != nil {
			// HACK: since slashing and auth use the same prefix for two different things,
			// let's ignore "no concrete type registered for type URL *" errors.
			// This is ugly, but frankly this is the only way to do it.
			// Frojdi please bless us with the new SDK ASAP.

			if strings.HasPrefix(err.Error(), "no concrete type registered for type URL") {
				perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
					Value:        "value is not AccountI",
					PayloadIndex: idx,
				})
				p.l.Debugw("exiting because value isnt accountI")
				continue
			}

			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        err.Error(),
				PayloadIndex: idx,
			})
			continue
		}

		if _, ok := acc.(*types.ModuleAccount); ok {
			// ignore moduleaccounts
			p.l.Debugw("detected moduleaccount, ignoring")
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        "detected moduleaccount, ignoring",
				PayloadIndex: idx,
			})
			continue
		}

		baseAcc, ok := acc.(*types.BaseAccount)
		if !ok {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Sprintf("cannot cast account to BaseAccount, type %T, account object type %T", baseAcc, acc),
				PayloadIndex: idx,
			})
			continue
		}

		if err := baseAcc.Validate(); err != nil {
			p.l.Debugw("found invalid base account", "account", baseAcc, "error", err)
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Errorf("non compliant auth account, %w", err).Error(),
				PayloadIndex: idx,
			})
			continue
		}

		_, bz, err := bech32.DecodeAndConvert(baseAcc.Address)
		if err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Errorf("cannot parse %s as bech32, %w", baseAcc.Address, err).Error(),
				PayloadIndex: idx,
			})
			continue
		}

		hAddr := hex.EncodeToString(bz)
		seq := acc.GetSequence()
		accN := acc.GetAccountNumber()

		res = append(res, &sdkutilities.Auth{
			Address:        hAddr,
			SequenceNumber: seq,
			AccountNumber:  accN,
		})
	}

	return
}

func (p Processor) Bank(ctx context.Context, payload *sdkutilities.BankPayload) (res []*sdkutilities.Balance, err error) {
	perrs := newPe()
	defer func() {
		if perrs.Errors != nil {
			err = &perrs
		}
	}()

	for idx, pl := range payload.Payload {
		addrBytes := pl.Key
		pLen := len(bankTypes.BalancesPrefix)

		if len(addrBytes) < pLen+20 {
			p.l.Debugw("found bank entry which doesn't respect balance prefix bounds check, ignoring")
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        "found bank entry which doesn't respect balance prefix bounds check, ignoring",
				PayloadIndex: idx,
			})
			continue
		}

		addr := addrBytes[pLen : pLen+20]

		coin := sdk.Coin{
			Amount: sdk.NewInt(0),
		}

		if err := p.cdc.UnmarshalBinaryBare(pl.Value, &coin); err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        err.Error(),
				PayloadIndex: idx,
			})
			continue
		}

		if !coin.IsValid() {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        "detected invalid coin, ignoring",
				PayloadIndex: idx,
			})
			continue
		}

		hAddr := hex.EncodeToString(addr)

		res = append(res, &sdkutilities.Balance{
			Address: hAddr,
			Amount:  coin.Amount.String(),
			Denom:   coin.Denom,
		})
	}

	return
}

func (p Processor) DelegationEndpoint(ctx context.Context, payload *sdkutilities.DelegationPayload) (res []*sdkutilities.Delegation, err error) {
	perrs := newPe()
	defer func() {
		if perrs.Errors != nil {
			err = &perrs
		}
	}()

	for idx, pl := range payload.Payload {
		if *pl.OperationType == string(tracemeta.DeleteOp) {
			if len(pl.Key) < 41 { // 20 bytes by address, 1 prefix = 2*20 + 1
				perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
					Value:        "detected invalid delegation row, ignoring",
					PayloadIndex: idx,
				})
				continue // found probably liquidity stuff being deleted
			}

			delegatorAddr := hex.EncodeToString(pl.Key[1:21])
			validatorAddr := hex.EncodeToString(pl.Key[21:41])
			p.l.Debugw("new delegation delete", "delegatorAddr", delegatorAddr, "validatorAddr", validatorAddr)

			delType := tracemeta.TypeDeleteDelegation
			res = append(res, &sdkutilities.Delegation{
				Delegator: delegatorAddr,
				Validator: validatorAddr,
				Amount:    "",
				Type:      delType,
			})

			continue
		}

		delegation := delegationtypes.Delegation{}

		if err := p.cdc.UnmarshalBinaryBare(pl.Value, &delegation); err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Errorf("found delegation object, but cannot unmarshal, %w", err).Error(),
				PayloadIndex: idx,
			})
			continue
		}

		delegator, err := b32Hex(delegation.DelegatorAddress)
		if err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Errorf("cannot convert delegator address from bech32 to hex, %w", err).Error(),
				PayloadIndex: idx,
			})
			continue
		}

		validator, err := b32Hex(delegation.ValidatorAddress)
		if err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Errorf("cannot convert validator address from bech32 to hex, %w", err).Error(),
				PayloadIndex: idx,
			})
			continue
		}

		delAmount := delegation.Shares.String()
		creatType := tracemeta.TypeCreateDelegation

		res = append(res, &sdkutilities.Delegation{
			Delegator: delegator,
			Validator: validator,
			Amount:    delAmount,
			Type:      creatType,
		})
	}

	return
}

func (p Processor) IbcChannel(ctx context.Context, payload *sdkutilities.IbcChannelPayload) (res []*sdkutilities.IBCChannel, err error) {
	perrs := newPe()
	defer func() {
		if perrs.Errors != nil {
			err = &perrs
		}
	}()

	for idx, pl := range payload.Payload {
		p.l.Debugw("ibc channel key", "key", string(pl.Key), "raw value", string(pl.Value))
		var result ibcchanneltypes.Channel
		if err := p.cdc.UnmarshalBinaryBare(pl.Value, &result); err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        err.Error(),
				PayloadIndex: idx,
			})
			continue
		}

		if err := result.ValidateBasic(); err != nil {
			p.l.Debugw("found non-compliant channel", "channel", result, "error", err)
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Sprintf("found non-compliant channel %s: %s", result, err.Error()),
				PayloadIndex: idx,
			})
			continue
		}

		if result.Ordering != ibcchanneltypes.UNORDERED {
			continue
		}

		p.l.Debugw("ibc channel data", "result", result)

		portID, channelID, err := host.ParseChannelPath(string(pl.Key))
		if err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        err.Error(),
				PayloadIndex: idx,
			})
			continue
		}

		state := int32(result.State)

		res = append(res, &sdkutilities.IBCChannel{
			ChannelID:        channelID,
			CounterChannelID: result.Counterparty.ChannelId,
			Hops:             result.GetConnectionHops(),
			Port:             portID,
			State:            state,
		})
	}

	return
}

func (p Processor) IbcClientState(ctx context.Context, payload *sdkutilities.IbcClientStatePayload) (res []*sdkutilities.IBCClientState, err error) {
	perrs := newPe()
	defer func() {
		if perrs.Errors != nil {
			err = &perrs
		}
	}()

	for idx, pl := range payload.Payload {
		p.l.Debugw("ibc client key", "key", string(pl.Key), "raw value", string(pl.Value))
		var result exported.ClientState
		var dest *tmIBCTypes.ClientState
		if err := p.cdc.UnmarshalInterface(pl.Value, &result); err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        err.Error(),
				PayloadIndex: idx,
			})
			continue
		}

		if castRes, ok := result.(*tmIBCTypes.ClientState); !ok {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        "detected ibc client state not of tendermint type, ignoring",
				PayloadIndex: idx,
			})
			continue
		} else {
			dest = castRes
		}

		if err := result.Validate(); err != nil {
			p.l.Debugw("found non-compliant ibc connection", "connection", dest, "error", err)
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Errorf("find non-compliant ibc connection, %w", err).Error(),
				PayloadIndex: idx,
			})
			continue
		}

		keySplit := strings.Split(string(pl.Key), "/")
		clientID := keySplit[1]
		tp := int64(dest.TrustingPeriod)

		res = append(res, &sdkutilities.IBCClientState{
			ChainID:        dest.ChainId,
			ClientID:       clientID,
			LatestHeight:   dest.LatestHeight.RevisionHeight,
			TrustingPeriod: tp,
		})
	}

	return
}

func (p Processor) IbcConnection(ctx context.Context, payload *sdkutilities.IbcConnectionPayload) (res []*sdkutilities.IBCConnection, err error) {
	perrs := newPe()
	defer func() {
		if perrs.Errors != nil {
			err = &perrs
		}
	}()

	for idx, pl := range payload.Payload {
		keyFields := strings.FieldsFunc(string(pl.Key), func(r rune) bool {
			return r == '/'
		})

		p.l.Debugw("ibc store key", "fields", keyFields, "raw key", string(pl.Key))

		// IBC keys are mostly strings
		switch len(keyFields) {
		case 2:
			if keyFields[0] == host.KeyConnectionPrefix { // this is a ConnectionEnd
				ce := connectionTypes.ConnectionEnd{}
				if err := p.cdc.UnmarshalBinaryBare(pl.Value, &ce); err != nil {
					perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
						Value:        err.Error(),
						PayloadIndex: idx,
					})
					continue
				}

				if err := ce.ValidateBasic(); err != nil {
					p.l.Debugw("found non-compliant connection end", "connection end", ce, "error", err)
					perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
						Value:        fmt.Errorf("found non-compliant connection end, %w", err).Error(),
						PayloadIndex: idx,
					})
					continue
				}

				p.l.Debugw("connection end", "data", ce)

				state := ce.State.String()
				res = append(res, &sdkutilities.IBCConnection{
					ConnectionID:        keyFields[1],
					ClientID:            ce.ClientId,
					State:               state,
					CounterConnectionID: ce.Counterparty.ConnectionId,
					CounterClientID:     ce.Counterparty.ClientId,
				})
			}
		}
	}

	return
}

func (p Processor) IbcDenomTrace(ctx context.Context, payload *sdkutilities.IbcDenomTracePayload) (res []*sdkutilities.IBCDenomTrace, err error) {
	perrs := newPe()
	defer func() {
		if perrs.Errors != nil {
			err = &perrs
		}
	}()

	for idx, pl := range payload.Payload {
		p.l.Debugw("beginning denom trace processor", "key", string(pl.Key), "value", string(pl.Value))

		dt := transferTypes.DenomTrace{}
		if err := p.cdc.UnmarshalBinaryBare(pl.Value, &dt); err != nil {
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        err.Error(),
				PayloadIndex: idx,
			})
			continue
		}

		if err := dt.Validate(); err != nil {
			p.l.Debugw("found a denom trace that isn't ICS20 compliant", "denom trace", dt, "error", err)
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        fmt.Errorf("found a denom trace that isn't ICS20 compliant, %w", err).Error(),
				PayloadIndex: idx,
			})
			continue
		}

		if dt.BaseDenom == "" {
			p.l.Debugw("ignoring since it's not a denom trace")
			perrs.Errors = append(perrs.Errors, &sdkutilities.ErrorObject{
				Value:        "ignoring since it's not a denom trace",
				PayloadIndex: idx,
			})
			continue
		}

		hash := hex.EncodeToString(dt.Hash())

		res = append(res, &sdkutilities.IBCDenomTrace{
			Path:      dt.Path,
			BaseDenom: dt.BaseDenom,
			Hash:      hash,
		})
	}

	return
}

func b32Hex(s string) (string, error) {
	_, b, err := bech32.DecodeAndConvert(s)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func newPe() sdkutilities.ProcessingError {
	n := "ProcessingError"
	return sdkutilities.ProcessingError{
		Name: &n,
	}
}
