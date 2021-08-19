package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fil-assistant/lib"
	"fil-assistant/utils"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v6/actors/builtin"
	init_ "github.com/filecoin-project/specs-actors/v6/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v6/actors/builtin/multisig"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"strconv"
)

type Proposal struct {
	Msig string
	TxnID string
}

type msigInfo struct {
	ID 			int64 				`json:"提案号"`
	To     		address.Address		`json:"接收方"`
	Value  		abi.TokenAmount		`json:"金额"`
	Method 		string				`json:"方法"`
	Params 		cbg.CBORUnmarshaler	`json:"参数"`
	Approved 	[]address.Address	`json:"已赞成signer"`
}

func (m *Handler) CreateMultisig(ctx context.Context, addresses []string, pk, threshold, duration,
	initAmount string) (string, error) {
	pki, err := parsePrivateKey(pk)
	if err != nil {
		return "", err
	}

	signer := lib.ChooseSigner(pki.Type)
	from, err := signer.ToAddress(pki.PrivateKey)
	if err != nil {
		return "", err
	}

	thresholdNum, err := strconv.ParseUint(threshold, 10, 64)
	if err != nil {
		return "", err
	}

	durationNum, err := strconv.ParseInt(duration, 10, 64)
	if err != nil {
		return "", err
	}

	amount, err := types.ParseFIL(initAmount)
	if err != nil {
		return "", err
	}

	if uint64(len(addresses)) < thresholdNum {
		return "", xerrors.New("threshold can not be greater than signers length")
	} else if len(addresses) == 0 {
		return "", xerrors.New("provided signers are none")
	}

	signers := make([]address.Address, 0, len(addresses))
	for _, addr := range addresses {
		signer, err := address.NewFromString(addr)
		if err != nil {
			return "", err
		} else {
			signers = append(signers, signer)
		}
	}

	msigParams := &multisig.ConstructorParams{
		Signers:               signers,
		NumApprovalsThreshold: thresholdNum,
		UnlockDuration:        abi.ChainEpoch(durationNum),
		StartEpoch:            abi.ChainEpoch(0),
	}

	enc, err := actors.SerializeParams(msigParams)
	if err != nil {
		return "", err
	}

	execParams := &init_.ExecParams{
		CodeCID:           builtin.MultisigActorCodeID,
		ConstructorParams: enc,
	}

	enc, err = actors.SerializeParams(execParams)
	if err != nil {
		return "", err
	}

	rawMsg := &types.Message{
		From:       from,
		To:         builtin.InitActorAddr,
		Value:      abi.TokenAmount(amount),
		Method: 	builtin.MethodsInit.Exec,
		Params: 	enc,
	}
	res, err := m.messagePush(ctx, rawMsg, pki, signer, 0)
	execreturn := new(init_.ExecReturn)
	if err = execreturn.UnmarshalCBOR(bytes.NewReader(res)); err != nil {
		return "", err
	} else {
		return execreturn.IDAddress.String(), nil
	}
}

func (m *Handler) ProposeAddSigner(ctx context.Context, pk, newAddr string, increase bool, proposal *Proposal) error {
	if proposal == nil {
		return xerrors.New("proposal is nil")
	}
	msig, err := address.NewFromString(proposal.Msig)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	newSigner, err := address.NewFromString(newAddr)
	if err != nil {
		return err
	}

	enc, err := actors.SerializeParams(&multisig.AddSignerParams{
		Signer:   newSigner,
		Increase: increase,
	})
	if err != nil {
		return err
	}

	proposal.TxnID, err = m.propose(ctx, pki, msig, &multisig.ProposeParams{
		To: msig,
		Value: abi.NewTokenAmount(0),
		Method: builtin.MethodsMultisig.AddSigner,
		Params: enc,
	}, 0)
	return err
}

func (m *Handler) ProposeSwapSigner(ctx context.Context, pk, old, new string, proposal *Proposal) error {
	if proposal == nil {
		return xerrors.New("proposal is nil")
	}
	msig, err := address.NewFromString(proposal.Msig)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	oldSigner, err := address.NewFromString(old)
	if err != nil {
		return err
	}

	newSigner, err := address.NewFromString(new)
	if err != nil {
		return err
	}

	enc, err := actors.SerializeParams(&multisig.SwapSignerParams{
		From: oldSigner,
		To: newSigner,
	})
	if err != nil {
		return err
	}

	proposal.TxnID, err = m.propose(ctx, pki, msig, &multisig.ProposeParams{
		To: msig,
		Value: abi.NewTokenAmount(0),
		Method: builtin.MethodsMultisig.SwapSigner,
		Params: enc,
	}, 0)
	return err
}

func (m *Handler) ProposeRemoveSigner(ctx context.Context, pk, toRemove string, decrease bool, proposal *Proposal) error {
	if proposal == nil {
		return xerrors.New("proposal is nil")
	}
	msig, err := address.NewFromString(proposal.Msig)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	removeSigner, err := address.NewFromString(toRemove)
	if err != nil {
		return err
	}

	enc, err := actors.SerializeParams(&multisig.RemoveSignerParams{
		Signer:   removeSigner,
		Decrease: decrease,
	})
	if err != nil {
		return err
	}

	proposal.TxnID, err = m.propose(ctx, pki, msig, &multisig.ProposeParams{
		To: msig,
		Value: abi.NewTokenAmount(0),
		Method: builtin.MethodsMultisig.RemoveSigner,
		Params: enc,
	}, 0)
	return err
}

func (m *Handler) ProposeChangeThreshold(ctx context.Context, pk, threshold string, proposal *Proposal) error {
	if proposal == nil {
		return xerrors.New("proposal is nil")
	}
	msig, err := address.NewFromString(proposal.Msig)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	thresholdNum, err := strconv.ParseUint(threshold, 10, 64)
	if err != nil {
		return err
	}

	enc, err := actors.SerializeParams(&multisig.ChangeNumApprovalsThresholdParams{
		NewThreshold: thresholdNum,
	})
	if err != nil {
		return err
	}

	proposal.TxnID, err = m.propose(ctx, pki, msig, &multisig.ProposeParams{
		To: msig,
		Value: abi.NewTokenAmount(0),
		Method: builtin.MethodsMultisig.ChangeNumApprovalsThreshold,
		Params: enc,
	}, 0)
	return err
}

func (m *Handler) ProposeLockBalance(ctx context.Context, pk, start, duration, amount string, proposal *Proposal) error {
	if proposal == nil {
		return xerrors.New("proposal is nil")
	}
	msig, err := address.NewFromString(proposal.Msig)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	startNum, err := strconv.ParseInt(start, 10, 64)
	if err != nil {
		return err
	}

	durationNum, err := strconv.ParseInt(duration, 10, 64)
	if err != nil {
		return err
	}

	amnt, err := types.ParseFIL(amount)
	if err != nil {
		return err
	}

	enc, err := actors.SerializeParams(&multisig.LockBalanceParams{
		StartEpoch: abi.ChainEpoch(startNum),
		UnlockDuration: abi.ChainEpoch(durationNum),
		Amount: abi.TokenAmount(amnt),
	})
	if err != nil {
		return err
	}

	proposal.TxnID, err = m.propose(ctx, pki, msig, &multisig.ProposeParams{
		To: msig,
		Value: abi.NewTokenAmount(0),
		Method: builtin.MethodsMultisig.LockBalance,
		Params: enc,
	}, 0)
	return err
}

func (m *Handler) ApproveOrCancel(ctx context.Context, pk string, approve bool, proposal Proposal) error {
	msigAddr, err := address.NewFromString(proposal.Msig)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	signer := lib.ChooseSigner(pki.Type)
	from, err := signer.ToAddress(pki.PrivateKey)
	if err != nil {
		return err
	}

	txnid, err := strconv.ParseInt(proposal.TxnID, 10, 64)
	if err != nil {
		return err
	}

	enc, err := actors.SerializeParams(&multisig.TxnIDParams{ID: multisig.TxnID(txnid)})
	if err != nil {
		return err
	}

	var method abi.MethodNum
	if approve {
		method = builtin.MethodsMultisig.Approve
	} else {
		method = builtin.MethodsMultisig.Cancel
	}
	rawMsg := &types.Message{
		To:     msigAddr,
		From:   from,
		Value:  abi.NewTokenAmount(0),
		Method: method,
		Params: enc,
	}
	_, err = m.messagePush(ctx, rawMsg, pki, signer, 0)
	return err
}

func (m *Handler) propose(ctx context.Context, pki *types.KeyInfo, msig address.Address, params *multisig.ProposeParams,
	start int) (string, error) {
	signer := lib.ChooseSigner(pki.Type)
	from, err := signer.ToAddress(pki.PrivateKey)
	if err != nil {
		return "", err
	}

	enc, err := actors.SerializeParams(params)
	if err != nil {
		return "", err
	}
	rawMsg := &types.Message{
		To:     msig,
		From:   from,
		Value:  abi.NewTokenAmount(0),
		Method: builtin.MethodsMultisig.Propose,
		Params: enc,
	}
	ret, err := m.messagePush(ctx, rawMsg, pki, signer, start)
	retval := new(multisig.ProposeReturn)
	if err = retval.UnmarshalCBOR(bytes.NewReader(ret)); err != nil {
		return "", xerrors.Errorf("failed to unmarshal propose return value: %w", err)
	}
	if retval.Applied {
		return "", xerrors.Errorf("transaction was executed during propose, exit code: %d", retval.Code)
	} else {
		return strconv.FormatInt(int64(retval.TxnID), 10), nil
	}
}

func (m *Handler) GetPendingProposals(ctx context.Context, addr string) ([]byte, error) {
	msigAddr, err := address.NewFromString(addr)
	if err != nil {
		return nil, err
	}

	m.process(0.1)
	trxs, err := m.client.GetPendingMsigTrxs(ctx, msigAddr)
	if err != nil {
		return nil, err
	}

	base := 0.5 / float64(len(trxs))
	var res bytes.Buffer
	for i, trx := range trxs {
		m.process(0.5 + float64(i) * base)
		code, err := m.client.StateGetActorCode(ctx, trx.To)
		if err != nil {
			return nil, err
		}

		m, found := utils.MethodsMap[code][trx.Method]
		if !found {
			return nil, xerrors.Errorf("unknown method %d for actor %s", trx.Method, code)
		}

		if err = m.Params.UnmarshalCBOR(bytes.NewReader(trx.Params)); err != nil {
			return nil, err
		}

		mi, err := json.Marshal(msigInfo{
			ID: trx.ID,
			To: trx.To,
			Value: trx.Value,
			Method: m.Name,
			Params: m.Params,
			Approved: trx.Approved,
		})
		if err != nil {
			return nil, err
		}
		err = json.Indent(&res, mi, "", "\t")
		if err != nil {
			return nil, err
		}
		res.WriteByte('\n')
	}

	return res.Bytes(), nil
}