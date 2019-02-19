package callbacks

import (
	"github.com/pkg/errors"
)

var Deregister = errors.New("callback deregistered")

type Callback func(params ...interface{}) error
type ReduceCallback func(in interface{}, params ...interface{}) (interface{}, error)
