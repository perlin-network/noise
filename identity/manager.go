package identity

import "fmt"

type Manager interface {
	fmt.Stringer

	PublicID() []byte

	Sign(buf []byte) ([]byte, error)
	Verify(publicKeyBuf []byte, buf []byte, signature []byte) error
}
