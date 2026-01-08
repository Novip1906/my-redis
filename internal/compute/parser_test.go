package compute

import (
	"testing"

	"github.com/Novip1906/my-redis/internal/storage"
)

func TestParser(t *testing.T) {
	storage := storage.NewMemoryStorage()

	parser := NewParser(storage)

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
		{"FLUSH", "OK"},
	}

	for _, tt := range tests {

		response, _ := parser.ProcessCommand(tt.command)

		if response != tt.expected {
			t.Errorf("Command: %q, got: %q, want: %q", tt.command, response, tt.expected+"\n")
		}
	}
}
