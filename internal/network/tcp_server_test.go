package network

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/Novip1906/my-redis/internal/aof"
	"github.com/Novip1906/my-redis/internal/compute"
	"github.com/Novip1906/my-redis/internal/storage"
)

func TestTCPServer_Integration(t *testing.T) {
	storage := storage.NewMemoryStorage()

	parser := compute.NewParser(storage)

	aof, err := aof.NewAOF("database_test.aof")
	if err != nil {
		t.Error("Failed to init AOF", "error", err)
	}

	port := ":4000"
	server := NewTCPServer(port, parser, aof, slog.Default())

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
