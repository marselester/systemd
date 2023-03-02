package systemd

import (
	"bytes"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeHeader(t *testing.T) {
	conn := bytes.NewReader(mainPIDResponse)
	dec := newDecoder(conn)
	conv := newStringConverter(4096)
	var h header
	if err := decodeHeader(dec, conv, &h, false); err != nil {
		t.Fatal(err)
	}

	want := header{
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
	}
	if diff := cmp.Diff(want, h); diff != "" {
		t.Errorf(diff)
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
