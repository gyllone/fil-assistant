package lib

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-crypto"
	crypto2 "github.com/filecoin-project/go-state-types/crypto"
	"github.com/minio/blake2b-simd"
	"golang.org/x/xerrors"
)

type Secp256Signer struct {}

func (s *Secp256Signer) GenPriKey() ([]byte, error) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		return nil, xerrors.Errorf("GenPriKey error: %w", err)
	}
	return priv, nil
}

func (s *Secp256Signer) ToAddress(pk []byte) (address.Address, error) {
	pubKey := crypto.PublicKey(pk)
	addr, err := address.NewSecp256k1Address(pubKey)
	if err != nil {
		return address.Undef, xerrors.Errorf("convert private key to address error: %w", err)
	} else {
		return addr, nil
	}
}

func (s *Secp256Signer) Sign(pk, msg []byte) ([]byte, error) {
	b2sum := blake2b.Sum256(msg)
	sig, err := crypto.Sign(pk, b2sum[:])
	if err != nil {
		return nil, xerrors.Errorf("secp256k1 signing error: %w", err)
	} else {
		return sig, nil
	}
}

func (s *Secp256Signer) Type() crypto2.SigType {
	return crypto2.SigTypeSecp256k1
}