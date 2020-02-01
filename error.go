package noise

import "errors"

var (
	// ErrMessageTooLarge is reported by a client when it receives a message from a peer that exceeds the max
	// receivable message size limit configured on a node.
	ErrMessageTooLarge = errors.New("msg from peer is too large")
)
