package lib

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
)

type Signer interface {
	GenPriKey() ([]byte, error)
	ToAddress(pk []byte) (address.Address, error)
	Sign(pk, msg []byte) ([]byte, error)
	Type() crypto.SigType
}

func ChooseSigner(t types.KeyType) Signer {
	switch t {
	case types.KTSecp256k1:
		return new(Secp256Signer)
	case types.KTBLS:
		return new(BlsSigner)
	default:
		panic("key type is not supported")
	}
}