package testclient

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// TestClient represents a test client connection to the MUD server
type TestClient struct {
	Name     string
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
	messages []string
	mu       sync.Mutex
	done     chan struct{}
}

// Credentials holds login/registration information
type Credentials struct {
	Username      string
	Password      string
	CharacterName string
}

// newClientConnection creates a basic client connection without authentication
func newClientConnection(address string) (*TestClient, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client := &TestClient{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		writer:   bufio.NewWriter(conn),
		messages: make([]string, 0),
		done:     make(chan struct{}),
	}

	// Start reading messages in background
	go client.readMessages()

	return client, nil
}

// NewTestClient creates a new test client by registering a new account.
// This is the primary way to create test clients - each gets a unique account.
func NewTestClient(name string, address string) (*TestClient, error) {
	client, err := newClientConnection(address)
	if err != nil {
		return nil, err
	}
	client.Name = name

	// Wait for welcome/login prompt
	time.Sleep(200 * time.Millisecond)

	// Choose register
	if err := client.SendCommand("r"); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to select register: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send username (use the name as username for simplicity)
	if err := client.SendCommand(name); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to send username: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send password
	password := name + "pass123"
	if err := client.SendCommand(password); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to send password: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Confirm password
	if err := client.SendCommand(password); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to confirm password: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send character name (same as username for tests)
	if err := client.SendCommand(name); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to send character name: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Select class (3 = Cleric for tests - has heal spell for magic tests)
	if err := client.SendCommand("3"); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to select class: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Confirm class selection (Y/N prompt)
	if err := client.SendCommand("Y"); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to confirm class: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send ability scores (standard array: 15, 14, 13, 12, 10, 8)
	// Assign in order: Strength, Dexterity, Constitution, Intelligence, Wisdom, Charisma
	abilityScores := []string{"15", "14", "13", "12", "10", "8"}
	for _, score := range abilityScores {
		if err := client.SendCommand(score); err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to send ability score: %w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)

	// Verify we're in the game (should see room description)
	if !client.WaitForMessage("Town Square", 2*time.Second) {
		// Check if we got an error message
		messages := client.GetMessages()
		client.Close()
		return nil, fmt.Errorf("failed to enter game, messages: %v", messages)
	}

	return client, nil
}

// NewTestClientWithLogin creates a test client by logging into an existing account
func NewTestClientWithLogin(creds Credentials, address string) (*TestClient, error) {
	client, err := newClientConnection(address)
	if err != nil {
		return nil, err
	}
	client.Name = creds.CharacterName

	// Wait for welcome/login prompt
	time.Sleep(200 * time.Millisecond)

	// Choose login
	if err := client.SendCommand("l"); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to select login: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send username
	if err := client.SendCommand(creds.Username); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to send username: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send password
	if err := client.SendCommand(creds.Password); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to send password: %w", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Select character (enter "1" for first character)
	if err := client.SendCommand("1"); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to select character: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	return client, nil
}

// NewTestClientRaw creates a raw client connection without any authentication.
// Use this for testing the auth flow itself.
func NewTestClientRaw(address string) (*TestClient, error) {
	client, err := newClientConnection(address)
	if err != nil {
		return nil, err
	}
	client.Name = "RawClient"

	// Wait for welcome prompt
	time.Sleep(200 * time.Millisecond)

	return client, nil
}

// readMessages continuously reads messages from the server
func (c *TestClient) readMessages() {
	for {
		select {
		case <-c.done:
			return
		default:
			line, err := c.reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line != "" {
				c.mu.Lock()
				c.messages = append(c.messages, line)
				c.mu.Unlock()
			}
		}
	}
}

// SendCommand sends a command to the server
func (c *TestClient) SendCommand(cmd string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.writer.WriteString(cmd + "\n")
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

// GetMessages returns all messages received so far
func (c *TestClient) GetMessages() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return a copy
	result := make([]string, len(c.messages))
	copy(result, c.messages)
	return result
}

// GetLastMessages returns the last N messages
func (c *TestClient) GetLastMessages(n int) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if n > len(c.messages) {
		n = len(c.messages)
	}

	start := len(c.messages) - n
	result := make([]string, n)
	copy(result, c.messages[start:])
	return result
}

// ClearMessages clears the message buffer
func (c *TestClient) ClearMessages() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = make([]string, 0)
}

// WaitForMessage waits for a message containing the specified text (with timeout)
func (c *TestClient) WaitForMessage(text string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		messages := c.GetMessages()
		for _, msg := range messages {
			if strings.Contains(msg, text) {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	return false
}

// WaitForAnyMessage waits for any of the specified texts (with timeout)
func (c *TestClient) WaitForAnyMessage(texts []string, timeout time.Duration) (string, bool) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		messages := c.GetMessages()
		for _, msg := range messages {
			for _, text := range texts {
				if strings.Contains(msg, text) {
					return text, true
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	return "", false
}

// Close closes the client connection
func (c *TestClient) Close() error {
	close(c.done)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetLastMessage returns the most recent message
func (c *TestClient) GetLastMessage() string {
	messages := c.GetLastMessages(1)
	if len(messages) > 0 {
		return messages[0]
	}
	return ""
}

// PrintMessages prints all messages (for debugging)
func (c *TestClient) PrintMessages() {
	messages := c.GetMessages()
	fmt.Printf("\n=== Messages for %s ===\n", c.Name)
	for i, msg := range messages {
		fmt.Printf("[%d] %s\n", i, msg)
	}
	fmt.Println("======================")
}

// HasMessage checks if any message contains the specified text
func (c *TestClient) HasMessage(text string) bool {
	messages := c.GetMessages()
	for _, msg := range messages {
		if strings.Contains(msg, text) {
			return true
		}
	}
	return false
}
