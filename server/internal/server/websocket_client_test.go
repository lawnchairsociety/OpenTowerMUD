package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestWebSocketClient_ReadLine_EmptyMessages tests that empty messages are skipped
// without causing stack overflow (tests the loop-based implementation)
func TestWebSocketClient_ReadLine_EmptyMessages(t *testing.T) {
	// Create a test WebSocket server
	upgrader := websocket.Upgrader{}
	messagesSent := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
		}
		defer conn.Close()

		// Send several empty messages followed by a valid message
		conn.WriteMessage(websocket.TextMessage, []byte(""))
		conn.WriteMessage(websocket.TextMessage, []byte("   "))
		conn.WriteMessage(websocket.TextMessage, []byte("\n\n\n"))
		conn.WriteMessage(websocket.TextMessage, []byte("valid message"))
		close(messagesSent)

		// Keep connection open briefly
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	// Connect to the test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := NewWebSocketClient(conn)

	// Wait for messages to be sent
	<-messagesSent

	// ReadLine should skip empty messages and return the valid one
	line, err := client.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine failed: %v", err)
	}

	if line != "valid message" {
		t.Errorf("Expected 'valid message', got '%s'", line)
	}
}

// TestWebSocketClient_ReadLine_MultiLineMessage tests that multi-line messages
// are split and buffered correctly
func TestWebSocketClient_ReadLine_MultiLineMessage(t *testing.T) {
	upgrader := websocket.Upgrader{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
		}
		defer conn.Close()

		// Send a multi-line message
		conn.WriteMessage(websocket.TextMessage, []byte("line1\nline2\nline3"))

		// Keep connection open briefly
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := NewWebSocketClient(conn)

	// Should get each line separately
	line1, _ := client.ReadLine()
	if line1 != "line1" {
		t.Errorf("Expected 'line1', got '%s'", line1)
	}

	line2, _ := client.ReadLine()
	if line2 != "line2" {
		t.Errorf("Expected 'line2', got '%s'", line2)
	}

	line3, _ := client.ReadLine()
	if line3 != "line3" {
		t.Errorf("Expected 'line3', got '%s'", line3)
	}
}

// TestWebSocketClient_WriteLine tests writing messages to the client
func TestWebSocketClient_WriteLine(t *testing.T) {
	upgrader := websocket.Upgrader{}
	received := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
		}
		defer conn.Close()

		// Read the message sent by client
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		received <- string(msg)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := NewWebSocketClient(conn)

	// Write a message
	err = client.WriteLine("Hello, World!")
	if err != nil {
		t.Fatalf("WriteLine failed: %v", err)
	}

	// Verify it was received
	select {
	case msg := <-received:
		if msg != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got '%s'", msg)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message")
	}
}

// TestWebSocketClient_RemoteAddr tests the RemoteAddr method
func TestWebSocketClient_RemoteAddr(t *testing.T) {
	upgrader := websocket.Upgrader{}
	done := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
		}
		defer conn.Close()
		<-done
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	defer close(done)

	client := NewWebSocketClient(conn)

	addr := client.RemoteAddr()
	if addr == "" {
		t.Error("RemoteAddr should not be empty")
	}
}
