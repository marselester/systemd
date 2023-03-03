package systemd

import (
	"encoding/binary"
	"fmt"
)

// Message types that can appear in the second byte of the header.
const (
	// typeMethodCall is a method call.
	// This message type may prompt a reply.
	typeMethodCall byte = 1 + iota
	// typeMethodReply is a method reply with returned data.
	typeMethodReply
	// typeError is an error reply.
	// If the first argument exists and is a string, it is an error message.
	typeError
	// typeSignal is a signal emission.
	typeSignal
)

// header represents a message header.
type header struct {
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

	// Fields contain header fields if a caller chose to decode them.
	// A header must contain the required header fields for its message type,
	// and zero or more of any optional header fields.
	Fields map[byte]headerField
}

const (
	littleEndian = 'l'
	bigEndian    = 'B'
)

// ByteOrder specifies how to convert byte slices into 32-bit unsigned integers.
// Both header and body are in this endianness.
func (h *header) Order() binary.ByteOrder {
	switch h.ByteOrder {
	case littleEndian:
		return binary.LittleEndian
	case bigEndian:
		return binary.BigEndian
	default:
		return nil
	}
}

const (
	// messageHeadSize is the length of the fixed part of a message header.
	messageHeadSize = 16
	// maxMessageSize is the maximum length of a message (128 MiB),
	// including header, header alignment padding, and body.
	maxMessageSize = 134217728
)

// decodeHeader decodes a message header from conn into h.
// The string converter conv helps to reduce allocs when decoding header fields.
// A caller can ignore the header fields with the skipFields flag.
//
// The signature of the header is "yyyyuua(yv)" which is
// BYTE, BYTE, BYTE, BYTE, UINT32, UINT32, ARRAY of STRUCT of (BYTE, VARIANT).
// Here only the fixed portion "yyyyuua" of the entire header is decoded
// where "a" is the length of the header array in bytes.
// The caller can later decode "(yv)" structs knowing how many bytes to process
// based on the header length.
func decodeHeader(dec *decoder, conv *stringConverter, h *header, skipFields bool) error {
	// Read the fixed portion of the message header (16 bytes),
	// and set the position of the next byte we should be reading from.
	b, err := dec.ReadN(messageHeadSize)
	if err != nil {
		return err
	}

	h.ByteOrder = b[0]
	order := h.Order()
	dec.SetOrder(order)

	h.Type = b[1]
	h.Flags = b[2]
	h.Proto = b[3]
	h.BodyLen = order.Uint32(b[4:8])
	h.Serial = order.Uint32(b[8:12])
	h.HeaderLen = order.Uint32(b[12:])

	if h.BodyLen > maxMessageSize {
		return fmt.Errorf("message exceeded the maximum length: %d/%d bytes", h.BodyLen, maxMessageSize)
	}

	// Read the header fields where the body signature is stored.
	// A caller might already know the signature from the spec
	// and choose not to decode the fields as an optimization.
	if skipFields {
		if b, err = dec.ReadN(h.HeaderLen); err != nil {
			return fmt.Errorf("message header: %w", err)
		}
	} else {
		var (
			ff        = make(map[byte]headerField)
			f         headerField
			hdrArrEnd = dec.offset + h.HeaderLen
		)
		for dec.offset < hdrArrEnd {
			if f, err = decodeHeaderField(dec, conv); err != nil {
				break
			}

			ff[f.Code] = f
		}
		if len(ff) > 0 {
			h.Fields = ff
		}
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

// Header fields.
const (
	// fieldPath is the object to send a call to,
	// or the object a signal is emitted from.
	// This header field is controlled by the message sender.
	fieldPath byte = 1 + iota
	// fieldInterface is the interface to invoke a method call on,
	// or that a signal is emitted from.
	// Optional for method calls, required for signals.
	// This header field is controlled by the message sender.
	fieldInterface
	// fieldMember is the member, either the method name or signal name.
	// This header field is controlled by the message sender.
	fieldMember
	// fieldErrorName is the name of the error that occurred.
	fieldErrorName
	// fieldReplySerial is the serial number of the message this message is a reply to.
	// The serial number is the second UINT32 in the header.
	// This header field is controlled by the message sender.
	fieldReplySerial
	// fieldDestination represents the name of the connection the message is intended for.
	// This field is usually only meaningful in combination with the message bus,
	// but other servers may define their own meanings for it.
	// This header field is controlled by the message sender.
	fieldDestination
	// fieldSender is a unique name of the sending connection.
	// This field is usually only meaningful in combination with the message bus,
	// but other servers may define their own meanings for it.
	// On a message bus, this header field is controlled by the message bus,
	// so it is as reliable and trustworthy as the message bus itself.
	// Otherwise, this header field is controlled by the message sender,
	// unless there is out-of-band information that indicates otherwise.
	fieldSender
	// fieldSignature is the signature of the message body.
	// If omitted, it is assumed to be the empty signature "",
	// i.e., the body must be 0-length.
	// This header field is controlled by the message sender.
	fieldSignature
	// fieldUnixFDs is the number of Unix file descriptors that accompany the message.
	// If omitted, it is assumed that no Unix file descriptors accompany the message.
	// The actual file descriptors need to be transferred via platform specific mechanism out-of-band.
	// They must be sent at the same time as part of the message itself.
	// They may not be sent before the first byte of the message itself is transferred
	// or after the last byte of the message itself.
	// This header field is controlled by the message sender.
	fieldUnixFDs
)

// headerField represents a header field.
// The array at the end of the header contains header fields,
// where each field is a 1-byte field code followed by a field value.
// For example, REPLY_SERIAL code and UINT32 value 3
// which is the serial number of the message this message is a reply to.
type headerField struct {
	// Signature is a signature (single complete type) of the value.
	Signature string
	// The following fields contain a header field value
	// depending on signature.
	// The decision was made against an interface{} to reduce allocs.
	U uint64
	S string

	// Code is a header field code, e.g., fieldPath.
	Code byte
}

func (f *headerField) String() string {
	var name string
	switch f.Code {
	case fieldPath:
		name = "PATH"
	case fieldInterface:
		name = "INTERFACE"
	case fieldMember:
		name = "MEMBER"
	case fieldErrorName:
		name = "ERROR_NAME"
	case fieldReplySerial:
		name = "REPLY_SERIAL"
	case fieldDestination:
		name = "DESTINATION"
	case fieldSender:
		name = "SENDER"
	case fieldSignature:
		name = "SIGNATURE"
	case fieldUnixFDs:
		name = "UNIX_FDS"
	default:
		name = "INVALID"
	}

	return name
}

// decodeHeaderField decodes a header field.
func decodeHeaderField(d *decoder, conv *stringConverter) (f headerField, err error) {
	// Since "(yv)" struct is being decoded, an alignment must be discarded.
	if err = d.Align(8); err != nil {
		return
	}

	// Decode "y" (a byte) which is a field code.
	if f.Code, err = d.Byte(); err != nil {
		return
	}

	// Decode "v" (variant) which is a field value.
	// Variants are marshalled as the SIGNATURE of the contents
	// (which must be a single complete type),
	// followed by a marshalled value with the type given by that signature.
	var sign []byte
	if sign, err = d.Signature(); err != nil {
		return
	}
	// Container types are not supported yet.
	// Because there is no need in the scope of this library.
	if len(sign) != 1 {
		return f, fmt.Errorf("container type is not supported: %s", sign)
	}
	f.Signature = conv.String(sign)

	var (
		u uint32
		s []byte
	)
	switch sign[0] {
	// UINT32 type.
	case 'u':
		if u, err = d.Uint32(); err != nil {
			return
		}
		f.U = uint64(u)
	// STRING, OBJECT_PATH types.
	case 's', 'o':
		if s, err = d.String(); err != nil {
			return
		}
		f.S = conv.String(s)
	// SIGNATURE type.
	case 'g':
		if s, err = d.Signature(); err != nil {
			return
		}
		f.S = conv.String(s)
	default:
		return f, fmt.Errorf("unknown type: %s", sign)
	}

	return
}
