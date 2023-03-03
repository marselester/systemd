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
				Type:      typeMethodCall,
				Flags:     0,
				Proto:     1,
				BodyLen:   52,
				Serial:    3,
				HeaderLen: 160,
				Fields: map[uint8]headerField{
					1: {Signature: "o", S: "/org/freedesktop/systemd1/unit/dbus_2eservice", Code: 1},
					2: {Signature: "s", S: "org.freedesktop.DBus.Properties", Code: 2},
					3: {Signature: "s", S: "Get", Code: 3},
					6: {Signature: "s", S: "org.freedesktop.systemd1", Code: 6},
					8: {Signature: "g", S: "ss", Code: 8},
				},
			},
		},
		"pid response": {
			in: mainPIDResponse,
			want: header{
				ByteOrder: littleEndian,
				Type:      typeMethodReply,
				Flags:     1,
				Proto:     1,
				BodyLen:   8,
				Serial:    2263,
				HeaderLen: 45,
				Fields: map[uint8]headerField{
					5: {Signature: "u", U: uint64(3), Code: 5},
					6: {Signature: "s", S: ":1.388", Code: 6},
					7: {Signature: "s", S: ":1.0", Code: 7},
					8: {Signature: "g", S: "v", Code: 8},
				},
			},
		},
		"units request": {
			in: listUnitsRequest,
			want: header{
				ByteOrder: littleEndian,
				Type:      typeMethodCall,
				Flags:     0,
				Proto:     1,
				BodyLen:   0,
				Serial:    2,
				HeaderLen: 145,
				Fields: map[uint8]headerField{
					1: {Signature: "o", S: "/org/freedesktop/systemd1", Code: 1},
					2: {Signature: "s", S: "org.freedesktop.systemd1.Manager", Code: 2},
					3: {Signature: "s", S: "ListUnits", Code: 3},
					6: {Signature: "s", S: "org.freedesktop.systemd1", Code: 6},
				},
			},
		},
		"units response": {
			in: listUnitsResponse,
			want: header{
				ByteOrder: littleEndian,
				Type:      typeMethodReply,
				Flags:     1,
				Proto:     1,
				BodyLen:   35714,
				Serial:    1758,
				HeaderLen: 61,
				Fields: map[uint8]headerField{
					5: {Signature: "u", U: uint64(2), Code: 5},
					6: {Signature: "s", S: ":1.308", Code: 6},
					7: {Signature: "s", S: ":1.0", Code: 7},
					8: {Signature: "g", S: "a(ssssssouso)", Code: 8},
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
