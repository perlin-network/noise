package protocol

type DoneAction byte

const (
	DoneAction_Invalid DoneAction = iota
	DoneAction_NotDone
	DoneAction_SendMessage
	DoneAction_DoNothing
)

type HandshakeProcessor interface {
	ActivelyInitHandshake() ([]byte, interface{}, error)                                   // (message, state, err)
	PassivelyInitHandshake() (interface{}, error)                                          // (state, err)
	ProcessHandshakeMessage(state interface{}, payload []byte) ([]byte, DoneAction, error) // (message, doneAction, err)
}
