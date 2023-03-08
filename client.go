// Package systemd provides access to systemd via dbus
// using Unix domain sockets as a transport.
// The objective of this package is to list processes
// with low overhead for the caller.
package systemd

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Dial connects to dbus via a Unix domain socket
// specified by a bus address,
// for example, "unix:path=/run/user/1000/bus".
func Dial(busAddr string) (*net.UnixConn, error) {
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
//
// The address of the system message bus is given in
// the DBUS_SYSTEM_BUS_ADDRESS environment variable.
// If that variable is not set,
// the Client will try to connect to the well-known address
// unix:path=/var/run/dbus/system_bus_socket, see
// https://dbus.freedesktop.org/doc/dbus-specification.html.
func New(opts ...Option) (*Client, error) {
	conf := Config{
		connReadSize:         DefaultConnectionReadSize,
		strConvSize:          DefaultStringConverterSize,
		isSerialCheckEnabled: false,
	}
	for _, opt := range opts {
		opt(&conf)
	}

	// Establish a connection if a caller hasn't provided one.
	var err error
	if conf.conn == nil {
		addr := os.Getenv("DBUS_SYSTEM_BUS_ADDRESS")
		if addr == "" {
			addr = "unix:path=/var/run/dbus/system_bus_socket"
		}

		conf.conn, err = Dial(addr)
		if err != nil {
			return nil, err
		}
	}

	if err = authExternal(conf.conn); err != nil {
		return nil, fmt.Errorf("dbus auth failed: %w", err)
	}

	if err = sendHello(conf.conn); err != nil {
		return nil, fmt.Errorf("dbus hello failed: %w", err)
	}

	strConv := newStringConverter(conf.strConvSize)
	msgEnc := messageEncoder{
		Enc:  newEncoder(nil),
		Conv: strConv,
	}
	msgDec := messageDecoder{
		Dec:              newDecoder(nil),
		Conv:             strConv,
		SkipHeaderFields: true,
	}
	if conf.isSerialCheckEnabled {
		msgDec.SkipHeaderFields = false
	}

	c := Client{
		conf:    conf,
		bufConn: bufio.NewReaderSize(conf.conn, conf.connReadSize),
		msgEnc:  &msgEnc,
		msgDec:  &msgDec,
	}

	return &c, nil
}

// Client provides access to systemd via dbus.
// A caller shouldn't use Client concurrently.
type Client struct {
	conf Config
	// bufConn buffers the reads from a connection
	// thus reducing count of read syscalls.
	bufConn *bufio.Reader
	msgEnc  *messageEncoder
	msgDec  *messageDecoder

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

// Close closes the connection.
func (c *Client) Close() error {
	return c.conf.conn.Close()
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

// verifyMsgSerial verifies that the message serial sent
// in the request matches the reply serial found in the header field.
func verifyMsgSerial(h *header, wantSerial uint32) error {
	var replySerial uint32
	for _, f := range h.Fields {
		if f.Code == fieldReplySerial {
			replySerial = uint32(f.U)
			break
		}
	}

	if wantSerial != replySerial {
		return fmt.Errorf("message reply serial mismatch: want %d got %d", wantSerial, replySerial)
	}
	return nil
}

// ListUnits fetches systemd units,
// optionally filters them with a given predicate, and calls f.
// The pointer to Unit struct in f must not be retained,
// because its fields change on each f call.
//
// Note, don't call any Client's methods within f,
// because concurrent reading from the same underlying connection
// is not supported.
func (c *Client) ListUnits(p Predicate, f func(*Unit)) error {
	if !c.mu.TryLock() {
		return fmt.Errorf("must be called serially")
	}
	defer c.mu.Unlock()

	serial := c.nextMsgSerial()
	// Send a dbus message that calls
	// org.freedesktop.systemd1.Manager.ListUnits method
	// to get an array of all currently loaded systemd units.
	err := c.msgEnc.EncodeListUnits(c.conf.conn, serial)
	if err != nil {
		return fmt.Errorf("encode ListUnits: %w", err)
	}

	err = c.msgDec.DecodeListUnits(c.bufConn, p, f)
	if err != nil {
		return fmt.Errorf("decode ListUnits: %w", err)
	}

	if c.conf.isSerialCheckEnabled {
		err = verifyMsgSerial(c.msgDec.Header(), serial)
	}

	return err
}

// MainPID fetches the main PID of the service.
// If a service is inactive (see Unit.ActiveState),
// the returned PID will be zero.
//
// Note, you can't call this method within ListUnits's f func,
// because that would imply concurrent reading from the same underlying connection.
// Simply waiting on a lock won't help, because ListUnits won't be able to
// finish waiting for MainPID, thus creating a deadlock.
func (c *Client) MainPID(service string) (pid uint32, err error) {
	if !c.mu.TryLock() {
		return 0, fmt.Errorf("must be called serially")
	}
	defer c.mu.Unlock()

	serial := c.nextMsgSerial()
	// Send a dbus message that calls
	// org.freedesktop.DBus.Properties.Get method
	// to retrieve MainPID property from
	// org.freedesktop.systemd1.Service interface.
	err = c.msgEnc.EncodeMainPID(c.conf.conn, service, serial)
	if err != nil {
		return 0, fmt.Errorf("encode MainPID: %w", err)
	}

	pid, err = c.msgDec.DecodeMainPID(c.bufConn)
	if err != nil {
		return pid, fmt.Errorf("decode MainPID: %w", err)
	}

	if c.conf.isSerialCheckEnabled {
		err = verifyMsgSerial(c.msgDec.Header(), serial)
	}

	return pid, err
}
