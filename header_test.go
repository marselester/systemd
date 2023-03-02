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
					1: {Code: 1, Value: "/org/freedesktop/systemd1/unit/dbus_2eservice"},
					2: {Code: 2, Value: "org.freedesktop.DBus.Properties"},
					3: {Code: 3, Value: "Get"},
					6: {Code: 6, Value: "org.freedesktop.systemd1"},
					8: {Code: 8, Value: "ss"},
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
					5: {Code: 5, Value: uint32(3)},
					6: {Code: 6, Value: ":1.388"},
					7: {Code: 7, Value: ":1.0"},
					8: {Code: 8, Value: "v"},
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
					1: {Code: 1, Value: "/org/freedesktop/systemd1"},
					2: {Code: 2, Value: "org.freedesktop.systemd1.Manager"},
					3: {Code: 3, Value: "ListUnits"},
					6: {Code: 6, Value: "org.freedesktop.systemd1"},
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
					5: {Code: 5, Value: uint32(2)},
					6: {Code: 6, Value: ":1.308"},
					7: {Code: 7, Value: ":1.0"},
					8: {Code: 8, Value: "a(ssssssouso)"},
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
