package eddsa

import (
	"github.com/Yayg/noise/internal/edwards25519"
	"github.com/Yayg/noise/signature"
	"github.com/pkg/errors"
)

var _ signature.Scheme = (*policy)(nil)

type policy struct{}

func (p policy) Sign(privateKey, messageBuf []byte) ([]byte, error) {
	return Sign(privateKey, messageBuf)
}

func (p policy) Verify(publicKeyBuf, messageBuf, signatureBuf []byte) error {
	return Verify(publicKeyBuf, messageBuf, signatureBuf)
}

func New() *policy {
	return new(policy)
}

func Sign(privateKeyBuf, messageBuf []byte) ([]byte, error) {
	if len(privateKeyBuf) != edwards25519.PrivateKeySize {
		return nil, errors.Errorf("edwards25519: private key expected to be %d bytes, but is %d bytes", edwards25519.PrivateKeySize, len(privateKeyBuf))
	}

	return edwards25519.Sign(privateKeyBuf, messageBuf), nil
}

func Verify(publicKeyBuf, messageBuf, signature []byte) error {
	if len(publicKeyBuf) != edwards25519.PublicKeySize {
		return errors.Errorf("edwards25519: public key expected to be %d bytes, but is %d bytes", edwards25519.PublicKeySize, len(publicKeyBuf))
	}

	if edwards25519.Verify(publicKeyBuf, messageBuf, signature) {
		return nil
	} else {
		return errors.New("unable to verify signature")
	}
}
