package wire

type State struct {
	strings map[byte]string
	slices  map[byte][]byte
	bytes   map[byte]byte
	bools   map[byte]bool
	uint16s map[byte]uint16
	uint32s map[byte]uint32
	uint64s map[byte]uint64

	msg []byte
}

func (p State) Message() []byte {
	return p.msg
}

func (p State) String(key byte) string {
	return p.strings[key]
}

func (p State) Bytes(key byte) []byte {
	return p.slices[key]
}

func (p State) Byte(key byte) byte {
	return p.bytes[key]
}

func (p State) Bool(key byte) bool {
	return p.bools[key]
}

func (p State) Uint16(key byte) uint16 {
	return p.uint16s[key]
}

func (p State) Uint32(key byte) uint32 {
	return p.uint32s[key]
}

func (p State) Uint64(key byte) uint64 {
	return p.uint64s[key]
}

func (p *State) SetMessage(val []byte) {
	p.msg = val
}

func (p *State) Set(key byte, val string) {
	p.strings[key] = val
}

func (p *State) SetBytes(key byte, val []byte) {
	p.slices[key] = val
}

func (p *State) SetByte(key, val byte) {
	p.bytes[key] = val
}

func (p *State) SetBool(key byte, val bool) {
	p.bools[key] = val
}

func (p *State) SetUint16(key byte, val uint16) {
	p.uint16s[key] = val
}

func (p *State) SetUint32(key byte, val uint32) {
	p.uint32s[key] = val
}

func (p *State) SetUint64(key byte, val uint64) {
	p.uint64s[key] = val
}
