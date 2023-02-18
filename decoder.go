package systemd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

const (
	// maxMessageSize is the maximum length of a message (128 MiB),
	// including header, header alignment padding, and body.
	maxMessageSize = 134217728
	// messageHeadSize is the length of the fixed part of a message header.
	messageHeadSize = 16

	littleEndian = 'l'
	bigEndian    = 'B'
	u32size      = 4
)

// messageHead represents the part of the message header
// that has a constant size (16 bytes).
type messageHead struct {
	// ByteOrder is an endianness flag;
	// ASCII 'l' for little-endian or ASCII 'B' for big-endian.
	// Both header and body are in this endianness.
	ByteOrder byte
	// Type is a message type.
	Type byte
	// Flags is a bitwise OR of message flags.
	Flags byte
	// Proto is a major protocol version of the sending application.
	Proto byte
	// BodyLen is a length in bytes of the message body,
	// starting from the end of the header.
	// The header ends after its alignment padding to an 8-boundary.
	BodyLen uint32
	// Serial is the serial of this message,
	// used as a cookie by the sender to identify the reply corresponding to this request.
	// This must not be zero.
	Serial uint32
	// HeaderLen is a length of the header array in bytes.
	// The array contains structs of header fields (code, variant).
	HeaderLen uint32
}

// ByteOrder specifies how to convert byte slices into 32-bit unsigned integers.
func (mh *messageHead) Order() binary.ByteOrder {
	switch mh.ByteOrder {
	case littleEndian:
		return binary.LittleEndian
	case bigEndian:
		return binary.BigEndian
	default:
		return nil
	}
}

// decodeMessageHead reads structured binary data from conn into the message head mh.
// It always reads msgHeadSize bytes into the buffer buf.
//
// The signature of the header is "yyyyuua(yv)" which is
// BYTE, BYTE, BYTE, BYTE, UINT32, UINT32, ARRAY of STRUCT of (BYTE, VARIANT).
// Here only the fixed portion "yyyyuua" of the entire header is decoded
// where "a" is the length of the header array in bytes.
// The caller can later decode "(yv)" structs knowing how many bytes to process
// based on the header length.
func decodeMessageHead(conn io.Reader, mh *messageHead, buf *bytes.Buffer) (err error) {
	buf.Reset()
	if _, err = io.CopyN(buf, conn, messageHeadSize); err != nil {
		return err
	}

	mh.ByteOrder = buf.Bytes()[0]
	if err = binary.Read(buf, mh.Order(), mh); err != nil {
		return fmt.Errorf("binary decode: %w", err)
	}

	return nil
}

// newDecoder creates a new D-Bus decoder.
// By default it expects the little-endian byte order
// and assumes a zero offset to start counting bytes read from src.
func newDecoder(src io.Reader) *decoder {
	return &decoder{
		order:  binary.LittleEndian,
		src:    src,
		buf:    &bytes.Buffer{},
		offset: 0,
	}
}

type decoder struct {
	order binary.ByteOrder
	src   io.Reader
	buf   *bytes.Buffer
	// offset is limited by maxMessageSize.
	offset uint32
}

// Reset resets the decoder to be reading from src
// with zero offset.
func (d *decoder) Reset(src io.Reader) {
	d.src = src
	d.offset = 0
}

// SetOrder sets a byte order used in decoding.
func (d *decoder) SetOrder(order binary.ByteOrder) {
	d.order = order
}

// SetOffset sets the tracked offset that is used for alignment.
// Note, it does not act like a Seek.
func (d *decoder) SetOffset(offset uint32) {
	d.offset = offset
}

// Align advances the decoder by discarding the alignment padding.
func (d *decoder) Align(n uint32) error {
	offset, padding := nextOffset(d.offset, n)
	if padding == 0 {
		return nil
	}

	_, err := readN(d.src, d.buf, padding)
	d.offset = offset
	return err
}

// Uint32 decodes D-Bus UINT32.
func (d *decoder) Uint32() (uint32, error) {
	err := d.Align(u32size)
	if err != nil {
		return 0, err
	}

	b, err := readN(d.src, d.buf, u32size)
	if err != nil {
		return 0, err
	}

	u := d.order.Uint32(b)
	// 4 bytes were read because uint32 takes 4 bytes.
	d.offset += u32size
	return u, nil
}

// String decodes D-Bus STRING or OBJECT_PATH.
// A caller must not retain the returned byte slice.
// The string conversion is not done here to avoid allocations.
func (d *decoder) String() ([]byte, error) {
	strLen, err := d.Uint32()
	if err != nil {
		return nil, err
	}
	// Account for a null byte at the end of the string.
	strLen++

	// Read the string content.
	b, err := readN(d.src, d.buf, strLen)
	if err != nil {
		return nil, err
	}

	d.offset += strLen
	return b[:strLen-1], nil
}

// readN reads exactly n bytes from src into the buffer.
// The buffer grows on demand.
// The objective is to reduce memory allocs.
func readN(src io.Reader, buf *bytes.Buffer, n uint32) ([]byte, error) {
	buf.Reset()
	buf.Grow(int(n))
	b := buf.Bytes()[:n]
	if _, err := src.Read(b); err != nil {
		return nil, err
	}

	return b, nil
}

// nextOffset returns the next byte position and the padding
// according to the current offset and alignment requirement.
func nextOffset(current, align uint32) (next, padding uint32) {
	if current%align == 0 {
		return current, 0
	}

	next = (current + align - 1) & ^(align - 1)
	padding = next - current
	return next, padding
}

func newStringConverter(cap int) *stringConverter {
	return &stringConverter{
		buf:    bytes.NewBuffer(make([]byte, 0, cap)),
		cap:    cap,
		offset: 0,
	}
}

// stringConverter converts bytes to strings with less allocs.
// The idea is to accumulate bytes in a buffer with specified capacity
// and create strings with unsafe.String using bytes from a buffer.
// For example, 10 "fizz" strings written to a 40-byte buffer
// will result in 1 alloc instead of 10.
//
// Once a buffer is filled, a new one is created with the same capacity.
// Old buffers will be eventually GC-ed
// with no side effects to the returned strings.
type stringConverter struct {
	// buf is a temporary buffer where decoded strings are batched.
	buf *bytes.Buffer
	// cap is a buffer capacity.
	cap int
	// offset is a buffer position where the last string was written.
	offset int
}

// String converts bytes to a string.
func (c *stringConverter) String(b []byte) string {
	if c.buf.Len() > c.cap {
		c.buf = bytes.NewBuffer(make([]byte, 0, c.cap))
		c.offset = 0
	}

	// Buffer always returns nil error.
	n, _ := c.buf.Write(b)
	if n == 0 {
		return ""
	}

	b = c.buf.Bytes()[c.offset:]
	s := unsafe.String(&b[0], len(b))
	c.offset += n
	return s
}
