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
	var h header
	if err := decodeHeader(dec, &h); err != nil {
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
	}
	if diff := cmp.Diff(want, h); diff != "" {
		t.Errorf(diff)
	}
}

func BenchmarkDecodeHeader(b *testing.B) {
	conn := bytes.NewReader(mainPIDResponse)
	dec := newDecoder(conn)
	var h header

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.Seek(0, io.SeekStart)

		if err := decodeHeader(dec, &h); err != nil {
			b.Error(err)
		}
	}
}
