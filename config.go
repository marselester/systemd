package systemd

const (
	// DefaultConnectionReadSize is the default size (in bytes)
	// of the buffer which is used for reading from a connection.
	// Buffering reduces count of read syscalls,
	// e.g., ListUnits makes 12 read syscalls when decoding 35KB message
	// using 4KB buffer.
	// It takes over 4K syscalls without buffering to decode the same message.
	DefaultConnectionReadSize = 4096
	// DefaultStringConverterSize is the default buffer size (in bytes)
	// of the string converter that is used to convert bytes to strings
	// with less allocs.
	//
	// After trying various buffer sizes on ListUnits,
	// a 4KB buffer showed 24.96 KB/op and 7 allocs/op
	// in a benchmark when decoding 35KB message.
	DefaultStringConverterSize = 4096
)

// Config represents a Client config.
type Config struct {
	// connReadSize defines the length of a buffer to read from
	// a D-Bus connection.
	connReadSize int
	// strConvSize defines the length of a buffer of a string converter.
	strConvSize int
	// isSerialCheckEnabled when set will check whether message serials match.
	isSerialCheckEnabled bool
}

// Option sets up a Config.
type Option func(*Config)

// WithConnectionReadSize sets a size of a buffer
// which is used for reading from a D-Bus connection.
// Bigger the buffer, less read syscalls will be made.
func WithConnectionReadSize(size int) Option {
	return func(c *Config) {
		c.connReadSize = size
	}
}

// WithStringConverterSize sets a buffer size of the string converter
// to reduce allocs.
func WithStringConverterSize(size int) Option {
	return func(c *Config) {
		c.strConvSize = size
	}
}

// WithSerialCheck when true enables checking of message serials,
// i.e., the Client will compare the serial number sent within a message to D-Bus
// with the serial received in the reply.
//
// Note, this requires decoding of header fields which incurs extra allocs.
// There shouldn't be any request/reply mishmash because
// the Client guarantees that the underlying D-Bus connection is accessed sequentially.
func WithSerialCheck(enable bool) Option {
	return func(c *Config) {
		c.isSerialCheckEnabled = enable
	}
}
