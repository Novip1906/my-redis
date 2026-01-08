package aof

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestAOF_WritexRead(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "database_test.aof")

	aof, err := NewAOF(dbPath)
	if err != nil {
		t.Fatalf("Failed to create AOF: %v", err)
	}

	commands := []string{
		"SET key1 value1",
		"SET key2 value2\n",
		"DEL key1",
	}

	for _, cmd := range commands {
		if err := aof.Write(cmd); err != nil {
			t.Fatalf("Failed to write command: %v", err)
		}
	}

	if err := aof.Close(); err != nil {
		t.Fatalf("Failed to close AOF: %v", err)
	}

	var recoveredCommands []string
	err = ReadAll(dbPath, func(line string) {
		recoveredCommands = append(recoveredCommands, line)
	})
	if err != nil {
		t.Fatalf("Failed to read AOF: %v", err)
	}

	if len(recoveredCommands) != len(commands) {
		t.Errorf("Expected %d commands, got %d", len(commands), len(recoveredCommands))
	}

	expected := []string{"SET key1 value1", "SET key2 value2", "DEL key1"}
	for i, cmd := range recoveredCommands {
		if cmd != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i, expected[i], cmd)
		}
	}
}

func TestAOF_ConcurrentWrite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "database_test.aof")

	aof, err := NewAOF(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer aof.Close()

	n := 100
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(val int) {
			defer wg.Done()
			err := aof.Write("SET key value")
			if err != nil {
				t.Errorf("Concurrent write failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
	aof.Close()

	linesCount := 0
	err = ReadAll(dbPath, func(line string) {
		linesCount++
		if line != "SET key value" {
			t.Errorf("Corrupted line detected: %q", line)
		}
	})

	if err != nil {
		t.Fatal(err)
	}

	if linesCount != n {
		t.Errorf("Expected %d lines, got %d. Mutex might be failing.", n, linesCount)
	}
}

func TestReadAll_NoFile(t *testing.T) {
	err := ReadAll("aopdapodspd.aof", func(line string) {
		t.Error("Callback should not be called for non-existent file")
	})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
}
