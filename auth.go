package systemd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
)

/*
authExternal performs EXTERNAL authentication,
see https://dbus.freedesktop.org/doc/dbus-specification.html#auth-protocol.
The protocol is a line-based, where each line ends with \r\n.

	client: AUTH EXTERNAL 31303030
	server: OK bde8d2222a9e966420ee8c1a63e972b4
	client: BEGIN

The client is authenticating as Unix uid 1000 in this example,
where 31303030 is ASCII decimal 1000 represented in hex.
*/
func authExternal(rw io.ReadWriter) error {
	// Send null byte as required by the protocol.
	if _, err := rw.Write([]byte{0}); err != nil {
		return fmt.Errorf("send null failed: %w", err)
	}

	uid := strconv.Itoa(os.Geteuid())
	var buf bytes.Buffer
	buf.WriteString("AUTH EXTERNAL ")
	buf.WriteString(hex.EncodeToString([]byte(uid)))
	buf.WriteString("\r\n")
	_, err := rw.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("AUTH EXTERNAL uid: %w", err)
	}

	// TODO: decode and handle the reply, but skip them for now.
	buf.Reset()
	buf.Grow(4096)
	b := buf.Bytes()[:buf.Cap()]
	if _, err = rw.Read(b); err != nil {
		return err
	}

	if !bytes.HasPrefix(b, []byte("OK")) {
		return fmt.Errorf("expected OK, got %s", b)
	}

	if _, err = rw.Write([]byte("BEGIN\r\n")); err != nil {
		return fmt.Errorf("BEGIN: %w", err)
	}

	return nil
}

func sendHello(conn io.ReadWriter) error {
	_, err := conn.Write(helloMsg)
	if err != nil {
		return err
	}

	// TODO: decode and handle the reply, but skip them for now.
	// Usually reply is 261 bytes, read them all with big enough buffer in one Read call.
	b := make([]byte, 300)
	if _, err = conn.Read(b); err != nil {
		return err
	}

	return nil
}

var helloMsg = []byte{108, 1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 110, 0, 0, 0, 6, 1, 115, 0, 20, 0, 0, 0, 111, 114, 103, 46, 102, 114, 101, 101, 100, 101, 115, 107, 116, 111, 112, 46, 68, 66, 117, 115, 0, 0, 0, 0, 3, 1, 115, 0, 5, 0, 0, 0, 72, 101, 108, 108, 111, 0, 0, 0, 2, 1, 115, 0, 20, 0, 0, 0, 111, 114, 103, 46, 102, 114, 101, 101, 100, 101, 115, 107, 116, 111, 112, 46, 68, 66, 117, 115, 0, 0, 0, 0, 1, 1, 111, 0, 21, 0, 0, 0, 47, 111, 114, 103, 47, 102, 114, 101, 101, 100, 101, 115, 107, 116, 111, 112, 47, 68, 66, 117, 115, 0, 0, 0}
