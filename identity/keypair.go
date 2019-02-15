package identity

import "fmt"

type Keypair interface {
	fmt.Stringer

	ID() []byte
	PublicKey() []byte
	PrivateKey() []byte

	Sign(buf []byte) ([]byte, error)
	Verify(publicKeyBuf []byte, buf []byte, signature []byte) error
}
