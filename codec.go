package noise

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
)

// Serializable attributes whether or not a type has a byte representation that it may be serialized into.
type Serializable interface {
	// Marshal converts this type into it's byte representation as a slice.
	Marshal() []byte
}

type codec struct {
	sync.RWMutex

	counter uint16
	ser     map[reflect.Type]uint16
	de      map[uint16]reflect.Value
}

func newCodec() *codec {
	return &codec{
		ser: make(map[reflect.Type]uint16, math.MaxUint16),
		de:  make(map[uint16]reflect.Value, math.MaxUint16),
	}
}

func (c *codec) register(ser Serializable, de interface{}) uint16 {
	c.Lock()
	defer c.Unlock()

	t := reflect.TypeOf(ser)
	d := reflect.ValueOf(de)

	if opcode, registered := c.ser[t]; registered {
		panic(fmt.Errorf("attempted to register type %+v which is already registered under opcode %d", t, opcode))
	}

	expected := reflect.FuncOf([]reflect.Type{reflect.TypeOf(([]byte)(nil))}, []reflect.Type{t, reflect.TypeOf((*error)(nil)).Elem()}, false)

	if d.Type() != expected {
		panic(fmt.Errorf("provided decoder for message type %+v is %s, but expected %s", t, d, expected))
	}

	c.ser[t] = c.counter
	c.de[c.counter] = d

	c.counter++

	return c.counter - 1
}

func (c *codec) encode(msg Serializable) ([]byte, error) {
	c.RLock()
	defer c.RUnlock()

	t := reflect.TypeOf(msg)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	opcode, registered := c.ser[t]
	if !registered {
		return nil, fmt.Errorf("opcode not registered for message type %+v", t)
	}

	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf[:2], opcode)

	return append(buf, msg.Marshal()...), nil
}

func (c *codec) decode(data []byte) (Serializable, error) {
	if len(data) < 2 {
		return nil, io.ErrUnexpectedEOF
	}

	opcode := binary.BigEndian.Uint16(data[:2])
	data = data[2:]

	c.RLock()
	defer c.RUnlock()

	decoder, registered := c.de[opcode]
	if !registered {
		return nil, fmt.Errorf("opcode %d is not registered", opcode)
	}

	results := decoder.Call([]reflect.Value{reflect.ValueOf(data)})

	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	return results[0].Interface().(Serializable), nil
}
