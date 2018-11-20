package protocol

import "encoding/binary"

func writeUvarint(buf *[]byte, x uint64) {
	endPos := len(*buf)
	*buf = append(*buf, make([]byte, 16)...)
	n := binary.PutUvarint((*buf)[endPos:], x)
	*buf = (*buf)[:endPos+n]
}

func writeUint16(buf *[]byte, x uint16) {
	endPos := len(*buf)
	*buf = append(*buf, make([]byte, 2)...)
	binary.LittleEndian.PutUint16((*buf)[endPos:], x)
}

func writeUint64(buf *[]byte, x uint64) {
	endPos := len(*buf)
	*buf = append(*buf, make([]byte, 4)...)
	binary.LittleEndian.PutUint64((*buf)[endPos:], x)
}
