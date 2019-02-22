package identity

import "fmt"

type Keypair interface {
	fmt.Stringer

	ID() []byte
	PublicKey() []byte
	PrivateKey() []byte
}
