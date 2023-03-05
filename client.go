// Package systemd provides access to systemd via dbus
// using Unix domain sockets as a transport.
// The objective of this package is to list processes
// with low overhead for the caller.
package systemd

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Dial connects to dbus via a Unix domain socket
// specified by DBUS_SESSION_BUS_ADDRESS env var,
// for example, "unix:path=/run/user/1000/bus".
func Dial() (*net.UnixConn, error) {
	busAddr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	prefix := "unix:path="
	if !strings.HasPrefix(busAddr, prefix) {
		return nil, fmt.Errorf("dbus address not found")
	}
	path := busAddr[len(prefix):]

	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{
		Name: path,
		Net:  "unix",
	})
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// New creates a new Client to access systemd via dbus.
// By default, the external auth is used.
func New(conn *net.UnixConn) (*Client, error) {
	var err error

	// Send null byte as required by the protocol.
	if _, err := conn.Write([]byte{0}); err != nil {
		return nil, fmt.Errorf("dbus send null failed: %w", err)
	}

	if err = authExternal(conn); err != nil {
		return nil, fmt.Errorf("dbus auth failed: %w", err)
	}

	if err = sendHello(conn); err != nil {
		return nil, fmt.Errorf("dbus hello failed: %w", err)
	}

	c := Client{
		conn:   conn,
		msgDec: newMessageDecoder(),
		msgEnc: newMessageEncoder(),
	}
	return &c, nil
}

// Client provides access to systemd via dbus.
// A caller shouldn't use Client concurrently.
type Client struct {
	conn   *net.UnixConn
	msgDec *messageDecoder
	msgEnc *messageEncoder

	// According to https://dbus.freedesktop.org/doc/dbus-specification.html
	// D-Bus connection receives messages serially.
	// The client doesn't have to wait for replies before sending more messages.
	// The client can match the replies with a serial number it included in a request.
	//
	// This Client implementation doesn't allow to call its methods concurrently,
	// because a caller could send multiple messages,
	// and the Client would read message fragments from the same connection.
	mu sync.Mutex
	// The serial of this message,
	// used as a cookie by the sender to identify the reply corresponding to this request.
	// This must not be zero.
	msgSerial uint32
}

// nextMsgSerial returns the next message number.
// It resets the serial to 1 after overflowing.
func (c *Client) nextMsgSerial() uint32 {
	c.msgSerial++
	// Start over when the serial overflows 4,294,967,295.
	if c.msgSerial == 0 {
		c.msgSerial++
	}
	return c.msgSerial
}

// ListUnits fetches systemd units and calls f.
// The pointer to Unit struct in f must not be retained,
// because its fields change on each f call.
//
// Note, don't call any Client's methods within f,
// because concurrent reading from the same underlying connection
// is not supported.
func (c *Client) ListUnits(f func(*Unit)) error {
	if !c.mu.TryLock() {
		return fmt.Errorf("must be called serially")
	}
	defer c.mu.Unlock()

	serial := c.nextMsgSerial()
	// Send a dbus message that calls
	// org.freedesktop.systemd1.Manager.ListUnits method
	// to get an array of all currently loaded systemd units.
	err := c.msgEnc.EncodeListUnits(c.conn, serial)
	if err != nil {
		return err
	}

	return c.msgDec.DecodeListUnits(c.conn, f)
}

// MainPID fetches the main PID of the service or 0 if there was an error.
//
// Note, you can't call this method within ListUnits's f func,
// because that would imply concurrent reading from the same underlying connection.
// Simply waiting on a lock won't help, because ListUnits won't be able to
// finish waiting for MainPID, thus creating a deadlock.
func (c *Client) MainPID(service string) (uint32, error) {
	if !c.mu.TryLock() {
		return 0, fmt.Errorf("must be called serially")
	}
	defer c.mu.Unlock()

	serial := c.nextMsgSerial()
	// Send a dbus message that calls
	// org.freedesktop.DBus.Properties.Get method
	// to retrieve MainPID property from
	// org.freedesktop.systemd1.Service interface.
	err := c.msgEnc.EncodeMainPID(c.conn, service, serial)
	if err != nil {
		return 0, err
	}

	return c.msgDec.DecodeMainPID(c.conn)
}
