package server

// Client abstracts the connection layer for both telnet and WebSocket connections.
// This allows the server to handle both protocols transparently.
type Client interface {
	// ReadLine blocks until a complete line is received (without newline).
	// Returns the line and any error encountered.
	ReadLine() (string, error)

	// WriteLine sends a line to the client.
	// For telnet, this appends a newline. For WebSocket, it sends as a message.
	WriteLine(message string) error

	// Write sends raw bytes to the client.
	Write(data []byte) error

	// Close closes the connection.
	Close() error

	// RemoteAddr returns the client's address for logging.
	RemoteAddr() string
}
