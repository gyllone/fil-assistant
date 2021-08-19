package chain

import (
	"context"
	"fil-assistant/lib"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

type msgSendSpec struct {
	MaxFee 	abi.TokenAmount
}

type MsigTransaction struct {
	ID     int64
	To     address.Address
	Value  abi.TokenAmount
	Method abi.MethodNum
	Params []byte

	Approved []address.Address
}

type MsgLookup struct {
	Message   cid.Cid // Can be different than requested, in case it was replaced, but only gas values changed
	Receipt   types.MessageReceipt
	ReturnDec interface{}
	TipSet    types.TipSetKey
	Height    abi.ChainEpoch
}

type LotusClient struct {
	client 				*rpc.Client
}

func NewLotusRpcClient(ctx context.Context, ipAddr, token string) (*LotusClient, error) {
	if client, err := rpc.DialContext(ctx, ipAddr); err != nil {
		return nil, err
	} else {
		client.SetHeader("Authorization", token)
		return &LotusClient{
			client: client,
		}, nil
	}
}

func (l *LotusClient) GetNonce(ctx context.Context, addr address.Address) (uint64, error) {
	var nonce uint64
	err := l.client.CallContext(ctx, &nonce, "Filecoin.MpoolGetNonce", addr)
	if err != nil {
		return 0, xerrors.Errorf("GetNonce error: %w", err)
	} else {
		return nonce, nil
	}
}

func (l *LotusClient) EstimateMessageGas(ctx context.Context, maxFee abi.TokenAmount, msg *types.Message) (*types.Message, error) {
	newMsg := new(types.Message)
	err := l.client.CallContext(ctx, newMsg, "Filecoin.GasEstimateMessageGas", msg, msgSendSpec{MaxFee: maxFee}, types.EmptyTSK)
	if err != nil {
		return nil, xerrors.Errorf("EstimateMessageGas error: %w", err)
	} else {
		return newMsg, nil
	}
}

func (l *LotusClient) SendMsg(ctx context.Context, pk []byte, msg *types.Message, signer lib.Signer) (cid.Cid, error) {
	mb, err := msg.ToStorageBlock()
	if err != nil {
		return cid.Undef, xerrors.Errorf("SendMsg ToStorageBlock error: %w", err)
	}

	sig, err := signer.Sign(pk, mb.Cid().Bytes())
	if err != nil {
		return cid.Undef, xerrors.Errorf("SendMsg Sign error: %w", err)
	}

	signedMsg := &types.SignedMessage{
		Message:	*msg,
		Signature: 	crypto.Signature{
			Type: signer.Type(),
			Data: sig,
		},
	}

	var c cid.Cid
	err = l.client.CallContext(ctx, &c, "Filecoin.MpoolPush", signedMsg)
	if err != nil {
		return cid.Undef, xerrors.Errorf("SendMsg Call error: %w", err)
	} else {
		return c, nil
	}
}

func (l *LotusClient) GetBalance(ctx context.Context, addr address.Address) (types.BigInt, error) {
	var balance types.BigInt
	err := l.client.CallContext(ctx, &balance, "Filecoin.WalletBalance", addr)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("GetBalance of %s error: %w", addr.String(), err)
	} else {
		return balance, nil
	}
}

func (l *LotusClient) LookupMessage(ctx context.Context, c cid.Cid) (*types.Message, error) {
	msg := new(types.Message)
	err := l.client.CallContext(ctx, msg, "Filecoin.ChainGetMessage", c)
	if err != nil {
		return nil, xerrors.Errorf("LookupMessage for %s error: %w", c.String(), err)
	} else {
		return msg, nil
	}
}

func (l *LotusClient) WaitMessage(ctx context.Context, c cid.Cid, confidence uint64) ([]byte, error) {
	wait := new(MsgLookup)
	err := l.client.CallContext(ctx, wait, "Filecoin.StateWaitMsg", c, confidence)
	if err != nil {
		return nil, xerrors.Errorf("WaitMessage for %s error: %w", c.String(), err)
	} else if wait.Receipt.ExitCode != 0 {
		return nil, xerrors.Errorf("WaitMessage executed, exit code %d", wait.Receipt.ExitCode)
	} else {
		return wait.Receipt.Return, nil
	}
}

func (l *LotusClient) LookupID(ctx context.Context, addr address.Address) (address.Address, error) {
	var addrId address.Address
	err := l.client.CallContext(ctx, &addrId, "Filecoin.StateLookupID", addr, types.EmptyTSK)
	if err != nil {
		return address.Undef, xerrors.Errorf("LookupID error: %w", err)
	} else {
		return addrId, nil
	}
}

func (l *LotusClient) GetMinerAvailableBalance(ctx context.Context, minerID address.Address) (types.BigInt, error) {
	var bal types.BigInt
	err := l.client.CallContext(ctx, &bal, "Filecoin.StateMinerAvailableBalance", minerID, types.EmptyTSK)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("GetMinerInfo error: %w", err)
	} else {
		return bal, nil
	}
}

func (l *LotusClient) GetPendingMsigTrxs(ctx context.Context, msigAddr address.Address) ([]MsigTransaction, error) {
	var trxs []MsigTransaction
	err := l.client.CallContext(ctx, &trxs, "Filecoin.MsigGetPending", msigAddr, types.EmptyTSK)
	if err != nil {
		return nil, xerrors.Errorf("GetPendingMsigTrxs error: %w", err)
	} else {
		return trxs, nil
	}
}

func (l *LotusClient) StateGetActorCode(ctx context.Context, actor address.Address) (cid.Cid, error) {
	var act types.Actor
	err := l.client.CallContext(ctx, &act, "Filecoin.StateGetActor", actor, types.EmptyTSK)
	if err != nil {
		return cid.Cid{}, xerrors.Errorf("StateGetActorCode error: %w", err)
	} else {
		return act.Code, nil
	}
}

func (l *LotusClient) Close() {
	l.client.Close()
}