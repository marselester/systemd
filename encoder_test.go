package systemd

import (
	"bytes"
	"testing"
)

func TestEscapeBusLabel(t *testing.T) {
	tt := map[string]string{
		"":                                     "_",
		"dbus":                                 "dbus",
		"dbus.service":                         "dbus_2eservice",
		"foo@bar.service":                      "foo_40bar_2eservice",
		"foo_bar@bar.service":                  "foo_5fbar_40bar_2eservice",
		"systemd-networkd-wait-online.service": "systemd_2dnetworkd_2dwait_2donline_2eservice",
		"555":                                  "_3555",
	}

	buf := &bytes.Buffer{}

	for name, want := range tt {
		buf.Reset()

		escapeBusLabel(name, buf)
		got := string(buf.Bytes())
		if want != got {
			t.Errorf("expected %q got %q", want, got)
		}
	}
}

func BenchmarkEscapeBusLabel(b *testing.B) {
	buf := &bytes.Buffer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()

		escapeBusLabel("dbus.service", buf)
		got = buf.Bytes()
	}
}
