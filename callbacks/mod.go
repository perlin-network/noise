package callbacks

import (
	"github.com/pkg/errors"
)

var DeregisterCallback = errors.New("callback deregistered")

type callback func(params ...interface{}) error
type reduceCallback func(in interface{}, params ...interface{}) (interface{}, error)

type LogMetadata struct {
	NodeID              string
	PeerID              string
	RuntimeCallerOffset int
}
