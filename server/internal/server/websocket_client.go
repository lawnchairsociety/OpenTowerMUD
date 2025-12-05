package server

import (
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketClient wraps a WebSocket connection for browser-based communication.
type WebSocketClient struct {
	conn    *websocket.Conn
	readBuf []string   // Buffer for lines when a message contains multiple lines
	mu      sync.Mutex // Protects readBuf
}

// NewWebSocketClient creates a new WebSocketClient from a WebSocket connection.
func NewWebSocketClient(conn *websocket.Conn) *WebSocketClient {
	return &WebSocketClient{
		conn:    conn,
		readBuf: make([]string, 0),
	}
}

// ReadLine reads a line from the WebSocket connection (blocking).
// If a message contains multiple lines, they are buffered and returned one at a time.
func (c *WebSocketClient) ReadLine() (string, error) {
	c.mu.Lock()
	// Check if we have buffered lines from a previous multi-line message
	if len(c.readBuf) > 0 {
		line := c.readBuf[0]
		c.readBuf = c.readBuf[1:]
		c.mu.Unlock()
		return line, nil
	}
	c.mu.Unlock()

	// Read a new message from the WebSocket
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return "", err
	}

	// Convert to string and split by newlines (in case client sends multiple lines)
	text := string(message)
	lines := strings.Split(text, "\n")

	// Filter out empty lines and trim whitespace
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}

	if len(filtered) == 0 {
		// Empty message, try again
		return c.ReadLine()
	}

	// Return first line, buffer the rest
	c.mu.Lock()
	if len(filtered) > 1 {
		c.readBuf = append(c.readBuf, filtered[1:]...)
	}
	c.mu.Unlock()

	return filtered[0], nil
}

// WriteLine writes a message to the WebSocket client.
// Unlike telnet, we don't need to add newlines - the message is self-contained.
func (c *WebSocketClient) WriteLine(message string) error {
	return c.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

// Write writes raw bytes to the WebSocket client as a text message.
func (c *WebSocketClient) Write(data []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Close closes the WebSocket connection.
func (c *WebSocketClient) Close() error {
	return c.conn.Close()
}

// RemoteAddr returns the remote address as a string.
func (c *WebSocketClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}
