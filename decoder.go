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

// decodeMessageHeader reads structured binary data from conn into the message header mh.
//
// The signature of the header is "yyyyuua(yv)" which is
// BYTE, BYTE, BYTE, BYTE, UINT32, UINT32, ARRAY of STRUCT of (BYTE, VARIANT).
// Here only the fixed portion "yyyyuua" of the entire header is decoded
// where "a" is the length of the header array in bytes.
// The caller can later decode "(yv)" structs knowing how many bytes to process
// based on the header length.
func decodeMessageHeader(dec *decoder, mh *messageHead) error {
	// Read the fixed portion of the message header (16 bytes),
	// and set the position of the next byte we should be reading from.
	b, err := dec.ReadN(messageHeadSize)
	if err != nil {
		return err
	}

	mh.ByteOrder = b[0]
	order := mh.Order()
	dec.SetOrder(order)

	mh.Type = b[1]
	mh.Flags = b[2]
	mh.Proto = b[3]
	mh.BodyLen = order.Uint32(b[4:8])
	mh.Serial = order.Uint32(b[8:12])
	mh.HeaderLen = order.Uint32(b[12:])

	if mh.BodyLen > maxMessageSize {
		return fmt.Errorf("message exceeded the maximum length: %d/%d bytes", mh.BodyLen, maxMessageSize)
	}

	// Read the header fields where the body signature is stored.
	// A caller might already know the signature from the spec
	// and choose not to decode the fields as an optimization.
	if b, err = dec.ReadN(mh.HeaderLen); err != nil {
		return fmt.Errorf("message header: %w", err)
	}

	// The length of the header must be a multiple of 8,
	// allowing the body to begin on an 8-byte boundary.
	// If the header does not naturally end on an 8-byte boundary,
	// up to 7 bytes of alignment padding is added.
	// Here we're discarding the header padding.
	if err = dec.Align(8); err != nil {
		return fmt.Errorf("discard header padding: %w", err)
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
	// offset is a current position in the message
	// which is used solely to determine the alignment.
	// The offset is limited by maxMessageSize.
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

// Align advances the decoder by discarding the alignment padding.
func (d *decoder) Align(n uint32) error {
	offset, padding := nextOffset(d.offset, n)
	if padding == 0 {
		return nil
	}

	_, err := readN(d.src, d.buf, int(padding))
	d.offset = offset
	return err
}

// ReadN reads exactly n bytes without decoding.
func (d *decoder) ReadN(n uint32) ([]byte, error) {
	d.offset += n
	return readN(d.src, d.buf, int(n))
}

// Byte decodes D-Bus BYTE.
func (d *decoder) Byte() (byte, error) {
	b, err := readN(d.src, d.buf, 1)
	if err != nil {
		return 0, err
	}

	d.offset++
	return b[0], nil
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
	b, err := readN(d.src, d.buf, int(strLen))
	if err != nil {
		return nil, err
	}

	d.offset += strLen
	return b[:strLen-1], nil
}

// Signature decodes D-Bus SIGNATURE
// which is the same as STRING except the length is a single byte
// (thus signatures have a maximum length of 255).
func (d *decoder) Signature() ([]byte, error) {
	strLen, err := d.Byte()
	if err != nil {
		return nil, err
	}
	// Account for a null byte at the end of the string.
	strLen++

	// Read the string content.
	b, err := readN(d.src, d.buf, int(strLen))
	if err != nil {
		return nil, err
	}

	d.offset += uint32(strLen)
	return b[:strLen-1], nil
}

// readN reads exactly n bytes from src into the buffer.
// The buffer grows on demand.
// The objective is to reduce memory allocs.
func readN(src io.Reader, buf *bytes.Buffer, n int) ([]byte, error) {
	buf.Reset()
	buf.Grow(n)
	b := buf.Bytes()[:n]

	// Since src is buffered, a single Read call
	// doesn't guarantee that all required n bytes will be read.
	// The second Read call fetches the remaining bytes.
	//
	// If the requested n bytes don't fit into src' buffer,
	// it doesn't buffer them, so there can't be three calls.
	//
	// Reading in a loop would simplify the reasoning,
	// but it works 8.51% slower for DecodeString, and 4.23% for DecodeListUnits.
	// TODO: See if bufio.Reader can be replaced by a faster version.
	var (
		k   int
		err error
	)
	if k, err = src.Read(b); err != nil {
		return nil, err
	}
	if k != n {
		k, err = src.Read(b[k:])
	}

	return b, err
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
		buf:    make([]byte, 0, cap),
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
	buf []byte
	// offset is a buffer position where the last string was written.
	offset int
}

// String converts bytes to a string.
func (c *stringConverter) String(b []byte) string {
	n := len(b)
	if n == 0 {
		return ""
	}
	// Must allocate because a string doesn't fit into the buffer.
	if n > cap(c.buf) {
		return string(b)
	}

	if len(c.buf)+n > cap(c.buf) {
		c.buf = make([]byte, 0, cap(c.buf))
		c.offset = 0
	}
	c.buf = append(c.buf, b...)

	b = c.buf[c.offset:]
	s := unsafe.String(&b[0], n)
	c.offset += n
	return s
}
