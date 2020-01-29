package noise

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
)

type Serializable interface {
	Marshal() []byte
}

type Codec struct {
	sync.RWMutex

	counter uint16
	ser     map[reflect.Type]uint16
	de      map[uint16]reflect.Value
}

func NewCodec() *Codec {
	return &Codec{
		ser: make(map[reflect.Type]uint16, math.MaxUint16),
		de:  make(map[uint16]reflect.Value, math.MaxUint16),
	}
}

func (c *Codec) Register(ser Serializable, de interface{}) uint16 {
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

func (c *Codec) Encode(msg Serializable) ([]byte, error) {
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

func (c *Codec) Decode(data []byte) (Serializable, error) {
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
