package systemd

import (
	"bytes"
	"encoding/binary"
)

// newEncoder creates a new D-Bus encoder.
// By default it uses the little-endian byte order
// and assumes a zero offset to start counting written bytes.
func newEncoder(dst *bytes.Buffer) *encoder {
	return &encoder{
		order:  binary.LittleEndian,
		dst:    dst,
		offset: 0,
	}
}

type encoder struct {
	order binary.ByteOrder
	dst   *bytes.Buffer
	// pad must always contain zeroes to add padding to dst.
	pad [8]byte
	// buf is a buffer that is used to encode integers.
	buf [4]byte
	// offset is a current position in the encoded message
	// which is used solely to determine the alignment.
	// The offset is limited by maxMessageSize.
	offset uint32
}

// Align adds the alignment padding.
func (e *encoder) Align(n uint32) {
	offset, padding := nextOffset(e.offset, n)
	if padding == 0 {
		return
	}

	e.dst.Write(e.pad[:padding])
	e.offset = offset
}

// Byte encodes D-Bus BYTE.
func (e *encoder) Byte(b byte) {
	e.dst.WriteByte(b)
	e.offset++
}

// Uint32 encodes D-Bus UINT32.
func (e *encoder) Uint32(u uint32) {
	const u32size = 4
	e.Align(u32size)

	b := e.buf[:u32size]
	e.order.PutUint32(b, u)
	e.dst.Write(b)
	// 4 bytes were written because uint32 takes 4 bytes.
	e.offset += u32size
}

// String encodes D-Bus STRING or OBJECT_PATH.
func (e *encoder) String(s string) {
	strLen := len(s)
	e.Uint32(uint32(strLen))

	e.dst.WriteString(s)
	// Account for a null byte at the end of the string.
	e.dst.WriteByte(0)
	e.offset += uint32(strLen + 1)
}

// Signature encodes D-Bus SIGNATURE
// which is the same as STRING except the length is a single byte
// (thus signatures have a maximum length of 255).
func (e *encoder) Signature(s string) {
	strLen := len(s)
	e.Byte(byte(strLen))

	e.dst.WriteString(s)
	// Account for a null byte at the end of the string.
	e.dst.WriteByte(0)
	e.offset += uint32(strLen + 1)
}
