package common

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fil-assistant/chain"
	"fil-assistant/lib"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v6/actors/builtin"
	"github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/v6/actors/builtin/multisig"
	"golang.org/x/xerrors"
)

type Handler struct {
	process 		func(float64) error
	confidence 		uint64
	maxFee 			abi.TokenAmount
	gasFeeCap 		types.BigInt
	client 			*chain.LotusClient
	block 			cipher.Block
}

func newHandler(ctx context.Context, ipAddr, token string, maxFee abi.TokenAmount, gasFeeCap types.BigInt,
	confidence uint64, key []byte, process func(float64) error) (*Handler, error) {
	client, err := chain.NewLotusRpcClient(ctx, ipAddr, token)
	if err != nil {
		return nil, err
	}

	block, _ := aes.NewCipher(key)

	return &Handler{
		process: 	process,
		confidence: confidence,
		maxFee:     maxFee,
		gasFeeCap:  gasFeeCap,
		client:     client,
		block: 		block,
	}, nil
}

func (m *Handler) messagePush(ctx context.Context, rawMsg *types.Message, pk *types.KeyInfo, signer lib.Signer, start int) ([]byte, error) {
	m.process(float64(start + 1) / float64(start + 6))
	bal, err := m.client.GetBalance(ctx, rawMsg.From)
	if err != nil {
		return nil, err
	}
	need := types.BigAdd(m.maxFee, rawMsg.Value)
	if bal.LessThan(need) {
		return nil, xerrors.Errorf("sender balance %s is less than %s", types.FIL(bal).String(), types.FIL(need).String())
	}

	m.process(float64(start + 2) / float64(start + 6))
	rawMsg.Nonce, err = m.client.GetNonce(ctx, rawMsg.From)
	if err != nil {
		return nil, err
	}

	m.process(float64(start + 3) / float64(start + 6))
	rawMsg.GasFeeCap = m.gasFeeCap
	newMsg, err := m.client.EstimateMessageGas(ctx, m.maxFee, rawMsg)
	if err != nil {
		return nil, err
	}

	m.process(float64(start + 4) / float64(start + 6))
	c, err := m.client.SendMsg(ctx, pk.PrivateKey, newMsg, signer)
	if err != nil {
		return nil, err
	}

	m.process(float64(start + 5) / float64(start + 6))
	return m.client.WaitMessage(ctx, c, m.confidence)
}

func (m *Handler) Send(ctx context.Context, pk, toAddr, amount string, proposal *Proposal) error {
	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	to, err := address.NewFromString(toAddr)
	if err != nil {
		return err
	}

	amnt, err := types.ParseFIL(amount)
	if err != nil {
		return err
	}

	if proposal == nil {
		signer := lib.ChooseSigner(pki.Type)
		from, err := signer.ToAddress(pki.PrivateKey)
		if err != nil {
			return err
		}

		rawMsg := &types.Message{
			From:       from,
			To:         to,
			Value:      abi.TokenAmount(amnt),
			Method:     builtin.MethodSend,
		}
		_, err = m.messagePush(ctx, rawMsg, pki, signer, 0)
		return err
	} else {
		msigAddr, err := address.NewFromString(proposal.Msig)
		if err != nil {
			return err
		}

		proposal.TxnID, err = m.propose(ctx, pki, msigAddr, &multisig.ProposeParams{
			To: to,
			Value: abi.TokenAmount(amnt),
			Method: builtin.MethodSend,
		}, 0)
		return err
	}
}

func (m *Handler) ChangeOwner1(ctx context.Context, pk, newAddr, minerID string, proposal *Proposal) error {
	mID, err := address.NewFromString(minerID)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	newOwner, err := address.NewFromString(newAddr)
	if err != nil {
		return err
	}

	num := 0
	if newOwner.Protocol() != address.ID {
		num = 1
		m.process(1 / float64(num + 6))
		newOwner, err = m.client.LookupID(ctx, newOwner)
		if err != nil {
			return err
		}
	}

	enc, err := actors.SerializeParams(&newOwner)
	if err != nil {
		return err
	}

	if proposal == nil {
		signer := lib.ChooseSigner(pki.Type)
		from, err := signer.ToAddress(pki.PrivateKey)
		if err != nil {
			return err
		}

		_, err = m.messagePush(ctx, &types.Message{
			From:       from,
			To:         mID,
			Value:      abi.NewTokenAmount(0),
			Method:     builtin.MethodsMiner.ChangeOwnerAddress,
			Params:     enc,
		}, pki, signer, num)
		return err
	} else {
		msigAddr, err := address.NewFromString(proposal.Msig)
		if err != nil {
			return err
		}

		proposal.TxnID, err = m.propose(ctx, pki, msigAddr, &multisig.ProposeParams{
			To: mID,
			Value: abi.NewTokenAmount(0),
			Method: builtin.MethodsMiner.ChangeOwnerAddress,
			Params: enc,
		}, num)
		return err
	}
}

func (m *Handler) ChangeOwner2(ctx context.Context, pk, minerID string, proposal *Proposal) error {
	mID, err := address.NewFromString(minerID)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	if proposal == nil {
		signer := lib.ChooseSigner(pki.Type)
		from, err := signer.ToAddress(pki.PrivateKey)
		if err != nil {
			return err
		}

		m.process(1 / float64(1 + 6))
		fromId, err := m.client.LookupID(ctx, from)
		if err != nil {
			return err
		}

		enc, err := actors.SerializeParams(&fromId)
		if err != nil {
			return err
		}

		_, err = m.messagePush(ctx, &types.Message{
			From:       from,
			To:         mID,
			Value:      abi.NewTokenAmount(0),
			Method:     builtin.MethodsMiner.ChangeOwnerAddress,
			Params:     enc,
		}, pki, signer, 1)
		return err
	} else {
		msigAddr, err := address.NewFromString(proposal.Msig)
		if err != nil {
			return err
		}

		enc, err := actors.SerializeParams(&msigAddr)
		if err != nil {
			return err
		}

		proposal.TxnID, err = m.propose(ctx, pki, msigAddr, &multisig.ProposeParams{
			To: mID,
			Value: abi.NewTokenAmount(0),
			Method: builtin.MethodsMiner.ChangeOwnerAddress,
			Params: enc,
		}, 0)
		return err
	}
}

func (m *Handler) Withdraw(ctx context.Context, pk, minerID, amount string, proposal *Proposal) error {
	mID, err := address.NewFromString(minerID)
	if err != nil {
		return err
	}

	amnt, err := types.ParseFIL(amount)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	m.process(1 / float64(1 + 6))
	avail, err := m.client.GetMinerAvailableBalance(ctx, mID)
	if err != nil {
		return err
	}
	if avail.LessThan(types.BigInt(amnt)) {
		return xerrors.Errorf("avail balance %s is less than withdraw amount %s", types.FIL(avail).String(), amnt.String())
	}

	enc, err := actors.SerializeParams(&miner.WithdrawBalanceParams{
		AmountRequested: abi.TokenAmount(amnt),
	})
	if err != nil {
		return err
	}

	if proposal == nil {
		signer := lib.ChooseSigner(pki.Type)
		from, err := signer.ToAddress(pki.PrivateKey)
		if err != nil {
			return err
		}

		_, err = m.messagePush(ctx, &types.Message{
			From:       from,
			To:         mID,
			Value:      abi.NewTokenAmount(0),
			Method:     builtin.MethodsMiner.WithdrawBalance,
			Params:     enc,
		}, pki, signer, 1)
		return err
	} else {
		msigAddr, err := address.NewFromString(proposal.Msig)
		if err != nil {
			return err
		}

		proposal.TxnID, err = m.propose(ctx, pki, msigAddr, &multisig.ProposeParams{
			To: mID,
			Value: abi.NewTokenAmount(0),
			Method: builtin.MethodsMiner.WithdrawBalance,
			Params: enc,
		}, 1)
		return err
	}
}

func (m *Handler) ProposeChangeWorker(ctx context.Context, pk, minerID, newWorker string, controls []string,
	proposal *Proposal) error {
	mID, err := address.NewFromString(minerID)
	if err != nil {
		return err
	}

	var num int
	var cs []address.Address
	for _, control := range controls {
		c, err := address.NewFromString(control)
		if err != nil {
			return err
		} else if c.Protocol() != address.ID {
			num++
		}
		cs = append(cs, c)
	}

	nw, err := address.NewFromString(newWorker)
	if err != nil {
		return err
	}
	if nw.Protocol() != address.ID {
		num++
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	i := 0
	if nw.Protocol() != address.ID {
		i++
		m.process(float64(i) / float64(num + 6))
		nw, err = m.client.LookupID(ctx, nw)
		if err != nil {
			return err
		}
	}

	for k, c := range cs {
		if c.Protocol() != address.ID {
			i++
			m.process(float64(i) / float64(num + 6))
			c, err = m.client.LookupID(ctx, c)
			if err != nil {
				return err
			} else {
				cs[k] = c
			}
		}
	}

	enc, err := actors.SerializeParams(&miner.ChangeWorkerAddressParams{
		NewWorker:       nw,
		NewControlAddrs: cs,
	})
	if err != nil {
		return err
	}

	if proposal == nil {
		signer := lib.ChooseSigner(pki.Type)
		from, err := signer.ToAddress(pki.PrivateKey)
		if err != nil {
			return err
		}

		_, err = m.messagePush(ctx, &types.Message{
			From:       from,
			To:         mID,
			Value:      abi.NewTokenAmount(0),
			Method:     builtin.MethodsMiner.ChangeWorkerAddress,
			Params:     enc,
		}, pki, signer, num)
		return err
	} else {
		msigAddr, err := address.NewFromString(proposal.Msig)
		if err != nil {
			return err
		}

		proposal.TxnID, err = m.propose(ctx, pki, msigAddr, &multisig.ProposeParams{
			To: mID,
			Value: abi.NewTokenAmount(0),
			Method: builtin.MethodsMiner.ChangeWorkerAddress,
			Params: enc,
		}, num)
		return err
	}
}

func (m *Handler) ConfirmChangeWorker(ctx context.Context, pk, minerID string, proposal *Proposal) error {
	mID, err := address.NewFromString(minerID)
	if err != nil {
		return err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return err
	}

	if proposal == nil {
		signer := lib.ChooseSigner(pki.Type)
		from, err := signer.ToAddress(pki.PrivateKey)
		if err != nil {
			return err
		}

		_, err = m.messagePush(ctx, &types.Message{
			From:       from,
			To:         mID,
			Value:      abi.NewTokenAmount(0),
			Method:     builtin.MethodsMiner.ConfirmUpdateWorkerKey,
		}, pki, signer, 0)
		return err
	} else {
		msigAddr, err := address.NewFromString(proposal.Msig)
		if err != nil {
			return err
		}

		proposal.TxnID, err = m.propose(ctx, pki, msigAddr, &multisig.ProposeParams{
			To: mID,
			Value: abi.NewTokenAmount(0),
			Method: builtin.MethodsMiner.ConfirmUpdateWorkerKey,
		}, 0)
		return err
	}
}

func (m *Handler) Encrypt(pk string) (string, string, error) {
	if m.block == nil {
		return "", "", xerrors.New("no aes key for encryption")
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return "", "", err
	}
	if len(pki.PrivateKey) != 32 {
		return "", "", xerrors.Errorf("invalid private key size %d, should be 32", len(pki.PrivateKey))
	}

	newKi := &types.KeyInfo{
		Type:       pki.Type,
		PrivateKey: make([]byte, 32),
	}
	m.block.Encrypt(newKi.PrivateKey, pki.PrivateKey)
	m.block.Encrypt(newKi.PrivateKey[16:], pki.PrivateKey[16:])

	addr, err := lib.ChooseSigner(pki.Type).ToAddress(pki.PrivateKey)
	if err != nil {
		return "", "", err
	}

	val, err := json.Marshal(newKi)
	if err != nil {
		return "", "", err
	} else {
		return addr.String(), hex.EncodeToString(val), nil
	}
}

func (m *Handler) Decrypt(pk string) (string, string, error) {
	if m.block == nil {
		return "", "", xerrors.New("no aes key for decryption")
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return "", "", err
	}

	if len(pki.PrivateKey) != 32 {
		return "", "", xerrors.Errorf("invalid private key size %d, should be 32", len(pki.PrivateKey))
	}

	rawKi := &types.KeyInfo{
		Type:       pki.Type,
		PrivateKey: make([]byte, 32),
	}
	m.block.Decrypt(rawKi.PrivateKey, pki.PrivateKey)
	m.block.Decrypt(rawKi.PrivateKey[16:], pki.PrivateKey[16:])

	addr, err := lib.ChooseSigner(rawKi.Type).ToAddress(rawKi.PrivateKey)
	if err != nil {
		return "", "", err
	}

	val, err := json.Marshal(rawKi)
	if err != nil {
		return "", "", err
	} else {
		return addr.String(), hex.EncodeToString(val), nil
	}
}

func (m *Handler) Sign(pk, msg string) (string, error) {
	raw, err := hex.DecodeString(msg)
	if err != nil {
		return "", err
	}

	pki, err := parsePrivateKey(pk)
	if err != nil {
		return "", err
	}

	sig, err := lib.ChooseSigner(pki.Type).Sign(pki.PrivateKey, raw)
	if err != nil {
		return "", err
	} else {
		var t crypto.SigType
		switch pki.Type {
		case types.KTBLS:
			t = crypto.SigTypeBLS
		case types.KTSecp256k1:
			t = crypto.SigTypeSecp256k1
		default:
			t = crypto.SigTypeUnknown
		}

		sigBytes := append([]byte{byte(t)}, sig...)
		return hex.EncodeToString(sigBytes), nil
	}
}

func (m *Handler) Close() {
	m.client.Close()
}

func parsePrivateKey(pk string) (*types.KeyInfo, error) {
	p, err := hex.DecodeString(pk)
	if err != nil {
		return nil, err
	}
	pki := new(types.KeyInfo)
	return pki, json.Unmarshal(p, pki)
}