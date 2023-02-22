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
		buf:     &bytes.Buffer{},
		dec:     newDecoder(nil),
		// With 4KB buffer, 35867B message takes 25603 B/op, 9 allocs/op.
		conv: newStringConverter(4096),
	}
}

// messageDecoder is responsible for decoding responses from dbus method calls.
type messageDecoder struct {
	// bufConn buffers the reads from a connection
	// thus reducing count of read syscalls (from 4079 to 12 in DecodeListUnits),
	// but it takes 1% longer for DecodeString, and 3% for DecodeListUnits.
	bufConn *bufio.Reader
	buf     *bytes.Buffer
	dec     *decoder
	conv    *stringConverter

	// The following fields are reused to reduce memory allocs.
	unit Unit
	msgh messageHead
}

// ListUnits decodes a reply from systemd ListUnits method.
// The pointer to Unit struct in f must not be retained,
// because its fields change on each f call.
func (d *messageDecoder) ListUnits(conn io.Reader, f func(*Unit)) error {
	d.bufConn.Reset(conn)

	// Read the fixed portion of the message header (16 bytes),
	// and set the position of the next byte we should be reading from.
	err := decodeMessageHead(d.bufConn, &d.msgh, d.buf)
	if err != nil {
		return fmt.Errorf("message head: %w", err)
	}
	var offset uint32 = messageHeadSize

	if d.msgh.BodyLen > maxMessageSize {
		return fmt.Errorf("message exceeded the maximum length: %d/%d bytes", d.msgh.BodyLen, maxMessageSize)
	}

	// Read the message header where the body signature is stored.
	// The header usually occupies 61 bytes.
	// Since we already know the signature from the spec,
	// the header is discarded.
	if _, err = readN(d.bufConn, d.buf, int(d.msgh.HeaderLen)); err != nil {
		return fmt.Errorf("message header: %w", err)
	}
	offset += d.msgh.HeaderLen

	// The length of the header must be a multiple of 8,
	// allowing the body to begin on an 8-byte boundary.
	// If the header does not naturally end on an 8-byte boundary,
	// up to 7 bytes of alignment padding is added.
	// Here we're discarding the header padding.
	offset, padding := nextOffset(offset, 8)
	if _, err = readN(d.bufConn, d.buf, int(padding)); err != nil {
		return fmt.Errorf("discard header padding: %w", err)
	}

	// Read the message body limited by the body length.
	// For example, if it is 35714 bytes,
	// we should stop reading at offset 35794,
	// because the body starts at offset 80,
	// i.e., offset 35794 = 16 head + 61 header + 3 padding + 35714 body.
	body := io.LimitReader(
		d.bufConn,
		int64(d.msgh.BodyLen),
	)
	d.dec.Reset(body)
	d.dec.SetOffset(offset)
	if d.msgh.ByteOrder != littleEndian {
		d.dec.SetOrder(d.msgh.Order())
	}

	// ListUnits has a body signature "a(ssssssouso)" which is
	// ARRAY of STRUCT of (STRING, STRING, STRING, STRING, STRING, STRING,
	// OBJECT_PATH, UINT32, STRING, OBJECT_PATH).
	//
	// Read the body starting from the array length "a" (uint32).
	// The array length is in bytes, e.g., 35706 bytes.
	if _, err = d.dec.Uint32(); err != nil {
		return fmt.Errorf("discard array length: %w", err)
	}

	for ; err == nil; err = decodeUnit(d.dec, d.conv, &d.unit) {
		f(&d.unit)
	}
	if err != io.EOF {
		return err
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

var listUnitsMsg = []byte{108, 1, 0, 1, 0, 0, 0, 0, 2, 0, 0, 0, 145, 0, 0, 0, 3, 1, 115, 0, 9, 0, 0, 0, 76, 105, 115, 116, 85, 110, 105, 116, 115, 0, 0, 0, 0, 0, 0, 0, 2, 1, 115, 0, 32, 0, 0, 0, 111, 114, 103, 46, 102, 114, 101, 101, 100, 101, 115, 107, 116, 111, 112, 46, 115, 121, 115, 116, 101, 109, 100, 49, 46, 77, 97, 110, 97, 103, 101, 114, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 111, 0, 25, 0, 0, 0, 47, 111, 114, 103, 47, 102, 114, 101, 101, 100, 101, 115, 107, 116, 111, 112, 47, 115, 121, 115, 116, 101, 109, 100, 49, 0, 0, 0, 0, 0, 0, 0, 6, 1, 115, 0, 24, 0, 0, 0, 111, 114, 103, 46, 102, 114, 101, 101, 100, 101, 115, 107, 116, 111, 112, 46, 115, 121, 115, 116, 101, 109, 100, 49, 0, 0, 0, 0, 0, 0, 0, 0}
