package lib

import (
	"crypto/rand"
	"fmt"
	"github.com/filecoin-project/go-address"
	crypto2 "github.com/filecoin-project/go-state-types/crypto"
	blst "github.com/supranational/blst/bindings/go"
	"golang.org/x/xerrors"
)

const DST = string("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_")

type BlsSigner struct {}

func (b *BlsSigner) GenPriKey() ([]byte, error) {
	// Generate 32 bytes of randomness
	var ikm [32]byte
	_, err := rand.Read(ikm[:])
	if err != nil {
		return nil, xerrors.Errorf("bls signature error generating random data")
	}
	// Note private keys seem to be serialized little-endian!
	return blst.KeyGen(ikm[:]).ToLEndian(), nil
}

func (b *BlsSigner) ToAddress(pk []byte) (address.Address, error) {
	pri := new(blst.SecretKey).FromLEndian(pk)
	if pri == nil || !pri.Valid() {
		return address.Undef, xerrors.Errorf("bls signature invalid private key")
	}
	pub := new(blst.P1Affine).From(pri).Compress()
	return address.NewBLSAddress(pub)
}

func (b *BlsSigner) Sign(pk, msg []byte) ([]byte, error) {
	pri := new(blst.SecretKey).FromLEndian(pk)
	if pri == nil || !pri.Valid() {
		return nil, fmt.Errorf("bls signature invalid private key")
	}
	return new(blst.P2Affine).Sign(pri, msg, []byte(DST)).Compress(), nil
}

func (b *BlsSigner) Type() crypto2.SigType {
	return crypto2.SigTypeBLS
}