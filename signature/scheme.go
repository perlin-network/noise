package signature

type Scheme interface {
	Sign(privateKey, messageBuf []byte) ([]byte, error)
	Verify(publicKeyBuf, messageBuf, signatureBuf []byte) error
}
