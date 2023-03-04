package systemd

import (
	"bytes"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeHeader(t *testing.T) {
	tt := map[string]struct {
		in   []byte
		want header
	}{
		"pid request": {
			in: mainPIDRequest,
			want: header{
				ByteOrder: littleEndian,
				Type:      msgTypeMethodCall,
				Flags:     0,
				Proto:     1,
				BodyLen:   52,
				Serial:    3,
				FieldsLen: 160,
				Fields: []headerField{
					{Signature: "o", S: "/org/freedesktop/systemd1/unit/dbus_2eservice", Code: fieldPath},
					{Signature: "s", S: "org.freedesktop.systemd1", Code: fieldDestination},
					{Signature: "s", S: "Get", Code: fieldMember},
					{Signature: "s", S: "org.freedesktop.DBus.Properties", Code: fieldInterface},
					{Signature: "g", S: "ss", Code: fieldSignature},
				},
			},
		},
		"pid response": {
			in: mainPIDResponse,
			want: header{
				ByteOrder: littleEndian,
				Type:      msgTypeMethodReply,
				Flags:     1,
				Proto:     1,
				BodyLen:   8,
				Serial:    2263,
				FieldsLen: 45,
				Fields: []headerField{
					{Signature: "u", U: uint64(3), Code: fieldReplySerial},
					{Signature: "s", S: ":1.388", Code: fieldDestination},
					{Signature: "g", S: "v", Code: fieldSignature},
					{Signature: "s", S: ":1.0", Code: fieldSender},
				},
			},
		},
		"units request": {
			in: listUnitsRequest,
			want: header{
				ByteOrder: littleEndian,
				Type:      msgTypeMethodCall,
				Flags:     0,
				Proto:     1,
				BodyLen:   0,
				Serial:    2,
				FieldsLen: 145,
				Fields: []headerField{
					{Signature: "s", S: "ListUnits", Code: fieldMember},
					{Signature: "s", S: "org.freedesktop.systemd1.Manager", Code: fieldInterface},
					{Signature: "o", S: "/org/freedesktop/systemd1", Code: fieldPath},
					{Signature: "s", S: "org.freedesktop.systemd1", Code: fieldDestination},
				},
			},
		},
		"units response": {
			in: listUnitsResponse,
			want: header{
				ByteOrder: littleEndian,
				Type:      msgTypeMethodReply,
				Flags:     1,
				Proto:     1,
				BodyLen:   35714,
				Serial:    1758,
				FieldsLen: 61,
				Fields: []headerField{
					{Signature: "u", U: uint64(2), Code: fieldReplySerial},
					{Signature: "s", S: ":1.308", Code: fieldDestination},
					{Signature: "g", S: "a(ssssssouso)", Code: fieldSignature},
					{Signature: "s", S: ":1.0", Code: fieldSender},
				},
			},
		},
	}

	conv := newStringConverter(4096)

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			dec := newDecoder(bytes.NewReader(tc.in))

			var h header
			if err := decodeHeader(dec, conv, &h, false); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, h); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func BenchmarkDecodeHeader(b *testing.B) {
	conn := bytes.NewReader(mainPIDResponse)
	dec := newDecoder(conn)
	conv := newStringConverter(4096)
	var h header

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.Seek(0, io.SeekStart)

		if err := decodeHeader(dec, conv, &h, false); err != nil {
			b.Error(err)
		}
	}
}

func TestEncodeHeader(t *testing.T) {
	tt := map[string]struct {
		want []byte
		h    header
	}{
		"pid request": {
			want: mainPIDRequest,
			h: header{
				ByteOrder: littleEndian,
				Type:      msgTypeMethodCall,
				Flags:     0,
				Proto:     1,
				BodyLen:   52,
				Serial:    3,
				FieldsLen: 160,
				Fields: []headerField{
					{Signature: "o", S: "/org/freedesktop/systemd1/unit/dbus_2eservice", Code: fieldPath},
					{Signature: "s", S: "org.freedesktop.systemd1", Code: fieldDestination},
					{Signature: "s", S: "Get", Code: fieldMember},
					{Signature: "s", S: "org.freedesktop.DBus.Properties", Code: fieldInterface},
					{Signature: "g", S: "ss", Code: fieldSignature},
				},
			},
		},
		"pid response": {
			want: mainPIDResponse,
			h: header{
				ByteOrder: littleEndian,
				Type:      msgTypeMethodReply,
				Flags:     1,
				Proto:     1,
				BodyLen:   8,
				Serial:    2263,
				FieldsLen: 45,
				Fields: []headerField{
					{Signature: "u", U: uint64(3), Code: fieldReplySerial},
					{Signature: "s", S: ":1.388", Code: fieldDestination},
					{Signature: "g", S: "v", Code: fieldSignature},
					{Signature: "s", S: ":1.0", Code: fieldSender},
				},
			},
		},
		"units request": {
			want: listUnitsRequest,
			h: header{
				ByteOrder: littleEndian,
				Type:      msgTypeMethodCall,
				Flags:     0,
				Proto:     1,
				BodyLen:   0,
				Serial:    2,
				FieldsLen: 145,
				Fields: []headerField{
					{Signature: "s", S: "ListUnits", Code: fieldMember},
					{Signature: "s", S: "org.freedesktop.systemd1.Manager", Code: fieldInterface},
					{Signature: "o", S: "/org/freedesktop/systemd1", Code: fieldPath},
					{Signature: "s", S: "org.freedesktop.systemd1", Code: fieldDestination},
				},
			},
		},
		"units response": {
			want: listUnitsResponse,
			h: header{
				ByteOrder: littleEndian,
				Type:      msgTypeMethodReply,
				Flags:     1,
				Proto:     1,
				BodyLen:   35714,
				Serial:    1758,
				FieldsLen: 61,
				Fields: []headerField{
					{Signature: "u", U: uint64(2), Code: fieldReplySerial},
					{Signature: "s", S: ":1.308", Code: fieldDestination},
					{Signature: "g", S: "a(ssssssouso)", Code: fieldSignature},
					{Signature: "s", S: ":1.0", Code: fieldSender},
				},
			},
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			dst := bytes.Buffer{}
			enc := newEncoder(&dst)

			if err := encodeHeader(enc, &tc.h); err != nil {
				t.Fatal(err)
			}

			wantHdrLen := tc.h.Len()
			if int(wantHdrLen) != dst.Len() {
				t.Errorf("expected header len %d, got %d", wantHdrLen, dst.Len())
			}

			want := tc.want[:wantHdrLen]
			if diff := cmp.Diff(want, dst.Bytes()); diff != "" {
				t.Error(diff, want, dst.Bytes())
			}
		})
	}
}

func BenchmarkEncodeHeader(b *testing.B) {
	dst := &bytes.Buffer{}
	enc := newEncoder(dst)
	h := header{
		ByteOrder: littleEndian,
		Type:      msgTypeMethodReply,
		Flags:     1,
		Proto:     1,
		BodyLen:   8,
		Serial:    2263,
		FieldsLen: 45,
		Fields: []headerField{
			{Signature: "u", U: uint64(3), Code: fieldReplySerial},
			{Signature: "s", S: ":1.388", Code: fieldDestination},
			{Signature: "g", S: "v", Code: fieldSignature},
			{Signature: "s", S: ":1.0", Code: fieldSender},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst.Reset()
		enc.Reset(dst)

		if err := encodeHeader(enc, &h); err != nil {
			b.Error(err)
		}
	}
}
