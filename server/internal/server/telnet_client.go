package server

import (
	"bufio"
	"net"
)

// TelnetClient wraps a raw TCP connection for telnet-style communication.
type TelnetClient struct {
	conn    net.Conn
	scanner *bufio.Scanner
	writer  *bufio.Writer
}

// NewTelnetClient creates a new TelnetClient from a TCP connection.
func NewTelnetClient(conn net.Conn) *TelnetClient {
	return &TelnetClient{
		conn:    conn,
		scanner: bufio.NewScanner(conn),
		writer:  bufio.NewWriter(conn),
	}
}

// ReadLine reads a line from the connection (blocking).
// Returns the line without the trailing newline.
func (c *TelnetClient) ReadLine() (string, error) {
	if c.scanner.Scan() {
		return c.scanner.Text(), nil
	}
	if err := c.scanner.Err(); err != nil {
		return "", err
	}
	// Scanner finished without error means EOF/connection closed
	return "", net.ErrClosed
}

// WriteLine writes a message followed by a newline to the client.
func (c *TelnetClient) WriteLine(message string) error {
	if _, err := c.writer.WriteString(message); err != nil {
		return err
	}
	return c.writer.Flush()
}

// Write writes raw bytes to the client.
func (c *TelnetClient) Write(data []byte) error {
	if _, err := c.writer.Write(data); err != nil {
		return err
	}
	return c.writer.Flush()
}

// Close closes the underlying connection.
func (c *TelnetClient) Close() error {
	return c.conn.Close()
}

// RemoteAddr returns the remote address as a string.
func (c *TelnetClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

// GetConn returns the underlying net.Conn for cases where direct access is needed
// (e.g., for IP extraction in auth).
func (c *TelnetClient) GetConn() net.Conn {
	return c.conn
}
