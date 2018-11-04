package network

import "encoding/binary"

func writeUvarint(buf *[]byte, x uint64) {
	endPos := len(*buf)
	*buf = append(*buf, make([]byte, 16)...)
	n := binary.PutUvarint((*buf)[endPos:], x)
	*buf = (*buf)[:endPos+n]
}
