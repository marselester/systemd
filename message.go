package systemd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
)

// Unit represents a currently loaded systemd unit.
// Note that units may be known by multiple names at the same name,
// and hence there might be more unit names loaded than actual units behind them.
type Unit struct {
	// Name is the primary unit name.
	Name string
	// Description is the human readable description.
	Description string
	// LoadState is the load state, i.e., whether the unit file has been loaded successfully.
	LoadState string
	// ActiveState is the active state, i.e., whether the unit is currently started or not.
	ActiveState string
	// SubState is the sub state, i.e.,
	// a more fine-grained version of the active state that
	// is specific to the unit type, which the active state is not.
	SubState string
	// Followed is a unit that is being followed in its state by this unit,
	// if there is any, otherwise the empty string.
	Followed string
	// Path is the unit object path.
	Path string
	// JobID is the numeric job ID
	// if there is a job queued for the job unit, 0 otherwise.
	JobID uint32
	// JobType is the job type.
	JobType string
	// JobPath is the job object path.
	JobPath string
}

func newMessageDecoder() *messageDecoder {
	return &messageDecoder{
		bufConn: bufio.NewReaderSize(nil, 4096),
		dec:     newDecoder(nil),
		// With 4KB buffer, 35867B message takes 25563 B/op, 7 allocs/op.
		conv: newStringConverter(4096),
	}
}

// messageDecoder is responsible for decoding responses from dbus method calls.
type messageDecoder struct {
	// bufConn buffers the reads from a connection
	// thus reducing count of read syscalls (from 4079 to 12 in DecodeListUnits),
	// but it takes 1% longer for DecodeString, and 3% for DecodeListUnits.
	bufConn *bufio.Reader
	dec     *decoder
	conv    *stringConverter

	// The following fields are reused to reduce memory allocs.
	unit Unit
	hdr  header
}

// DecodeListUnits decodes a reply from systemd ListUnits method.
// The pointer to Unit struct in f must not be retained,
// because its fields change on each f call.
func (d *messageDecoder) DecodeListUnits(conn io.Reader, f func(*Unit)) error {
	// Reset the decoder with the buffered connection reader.
	d.bufConn.Reset(conn)
	d.dec.Reset(d.bufConn)

	// Decode the message header (16 bytes).
	//
	// Then read the message header where the body signature is stored.
	// The header usually occupies 61 bytes.
	// Since we already know the signature from the spec,
	// the header is discarded.
	//
	// Note, the length of the header must be a multiple of 8,
	// allowing the body to begin on an 8-byte boundary.
	// If the header does not naturally end on an 8-byte boundary,
	// up to 7 bytes of alignment padding is added.
	err := decodeHeader(d.dec, d.conv, &d.hdr, true)
	if err != nil {
		return fmt.Errorf("message header: %w", err)
	}

	// Read the message body limited by the body length.
	// For example, if it is 35714 bytes,
	// we should stop reading at offset 35794,
	// because the body starts at offset 80,
	// i.e., offset 35794 = 16 head + 61 header + 3 padding + 35714 body.
	body := io.LimitReader(
		d.bufConn,
		int64(d.hdr.BodyLen),
	)
	d.dec.Reset(body)

	// ListUnits has a body signature "a(ssssssouso)" which is
	// ARRAY of STRUCT of (STRING, STRING, STRING, STRING, STRING, STRING,
	// OBJECT_PATH, UINT32, STRING, OBJECT_PATH).
	//
	// Read the body starting from the array length "a" (uint32).
	// The array length is in bytes, e.g., 35706 bytes.
	if _, err = d.dec.Uint32(); err != nil {
		return fmt.Errorf("discard unit array length: %w", err)
	}

	for ; err == nil; err = decodeUnit(d.dec, d.conv, &d.unit) {
		f(&d.unit)
	}
	if err != io.EOF {
		return fmt.Errorf("message body: %w", err)
	}

	return nil
}

// decodeUnit decodes D-Bus Unit struct.
func decodeUnit(d *decoder, conv *stringConverter, unit *Unit) error {
	// The "()" symbols in the signature represent a STRUCT
	// which is always aligned to an 8-byte boundary,
	// regardless of the alignments of their contents.
	if err := d.Align(8); err != nil {
		return err
	}

	// The Unit struct's fields represent the signature "ssssssouso".
	// Here we decode all its fields sequentially.
	v := reflect.ValueOf(unit).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		switch field.Kind() {
		case reflect.String:
			s, err := d.String()
			if err != nil {
				return err
			}
			field.SetString(conv.String(s))

		case reflect.Uint32:
			u, err := d.Uint32()
			if err != nil {
				return err
			}
			field.SetUint(uint64(u))
		}
	}

	return nil
}

// DecodeMainPID decodes MainPID property reply from systemd
// org.freedesktop.DBus.Properties.Get method.
func (d *messageDecoder) DecodeMainPID(conn io.Reader) (uint32, error) {
	d.bufConn.Reset(conn)
	d.dec.Reset(d.bufConn)

	err := decodeHeader(d.dec, d.conv, &d.hdr, true)
	if err != nil {
		return 0, fmt.Errorf("message header: %w", err)
	}

	body := io.LimitReader(
		d.bufConn,
		int64(d.hdr.BodyLen),
	)
	d.dec.Reset(body)

	// Discard known signature "u".
	if _, err = d.dec.Signature(); err != nil {
		return 0, err
	}

	return d.dec.Uint32()
}

func newMessageEncoder() *messageEncoder {
	return &messageEncoder{
		buf:  &bytes.Buffer{},
		enc:  newEncoder(nil),
		conv: newStringConverter(4096),
	}
}

// messageEncoder is responsible for encoding and sending messages to dbus.
type messageEncoder struct {
	// buf is a buffer where an encoder writes the message.
	buf  *bytes.Buffer
	enc  *encoder
	conv *stringConverter
}

// EncodeListUnits encodes a request to systemd ListUnits method.
func (e *messageEncoder) EncodeListUnits(conn io.Writer) error {
	// Reset the encoder to encode the header.
	e.buf.Reset()
	e.enc.Reset(e.buf)

	h := header{
		ByteOrder: littleEndian,
		Type:      msgTypeMethodCall,
		Proto:     1,
		Serial:    2,
		Fields: []headerField{
			{Signature: "s", S: "ListUnits", Code: fieldMember},
			{Signature: "s", S: "org.freedesktop.systemd1.Manager", Code: fieldInterface},
			{Signature: "o", S: "/org/freedesktop/systemd1", Code: fieldPath},
			{Signature: "s", S: "org.freedesktop.systemd1", Code: fieldDestination},
		},
	}
	err := encodeHeader(e.enc, &h)
	if err != nil {
		return fmt.Errorf("message header: %w", err)
	}

	if _, err = conn.Write(e.buf.Bytes()); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// EncodeMainPID encodes MainPID property request for the given unit name,
// e.g., "dbus.service".
func (e *messageEncoder) EncodeMainPID(conn io.Writer, unitName string) error {
	// Escape an object path to send a call to,
	// e.g., /org/freedesktop/systemd1/unit/dbus_2eservice.
	e.buf.Reset()
	e.buf.WriteString("/org/freedesktop/systemd1/unit/")
	escapeBusLabel(unitName, e.buf)
	objPath := e.conv.String(e.buf.Bytes())

	// Reset the encoder to encode the header and the body.
	e.buf.Reset()
	e.enc.Reset(e.buf)

	h := header{
		ByteOrder: littleEndian,
		Type:      msgTypeMethodCall,
		Proto:     1,
		Serial:    3,
		Fields: []headerField{
			{Signature: "o", S: objPath, Code: fieldPath},
			{Signature: "s", S: "org.freedesktop.systemd1", Code: fieldDestination},
			{Signature: "s", S: "Get", Code: fieldMember},
			{Signature: "s", S: "org.freedesktop.DBus.Properties", Code: fieldInterface},
			{Signature: "g", S: "ss", Code: fieldSignature},
		},
	}
	err := encodeHeader(e.enc, &h)
	if err != nil {
		return fmt.Errorf("message header: %w", err)
	}

	// Encode message body with a known signature "ss".
	const (
		iface    = "org.freedesktop.systemd1.Service"
		propName = "MainPID"
	)
	bodyOffset := e.enc.Offset()
	e.enc.String(iface)
	e.enc.String(propName)

	// Overwrite the h.BodyLen with an actual length of the message body.
	const headerBodyLenOffset = 4
	bodyLen := e.enc.Offset() - bodyOffset
	if err = e.enc.Uint32At(bodyLen, headerBodyLenOffset); err != nil {
		return fmt.Errorf("encode header BodyLen: %w", err)
	}

	if _, err = conn.Write(e.buf.Bytes()); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}
