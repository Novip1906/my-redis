package network

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/Novip1906/my-redis/internal/storage"
)

func TestTCPServer_Integration(t *testing.T) {
	store := storage.NewMemoryStorage()

	port := ":4000"
	server := NewTCPServer(port, store, slog.Default())

	go func() {
		if err := server.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost"+port)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	tests := []struct {
		command  string
		expected string
	}{
		{"SET mykey myvalue", "OK"},
		{"GET mykey", "myvalue"},
		{"GET unknown", "(nil)"},
		{"DEL mykey", "OK"},
		{"GET mykey", "(nil)"},
		{"SET with ttl", "OK"},
		{"EXPIRE with 2", "1"},
		{"TTL with", "2"},
		{"SET without ttl", "OK"},
		{"TTL without", "-1"},
		{"INCR testIncr", "1"},
		{"SET testIncr 2", "OK"},
		{"INCR testIncr", "3"},
	}

	reader := bufio.NewReader(conn)

	for _, tt := range tests {
		fmt.Fprint(conn, tt.command+"\n")

		response, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		if response != tt.expected+"\n" {
			t.Errorf("Command: %q, got: %q, want: %q", tt.command, response, tt.expected+"\n")
		}
	}
}
