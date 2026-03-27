package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Command struct {
	Cmd string `json:"cmd"`
}

type SerializeFunc func(data string) ([]byte, error)

func main() {
	dir := filepath.Join(os.TempDir(), "myredis-bench")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory %s: %v\n", dir, err)
		os.Exit(1)
	}

	rounds := 1_000_000
	dataCmd := "SET my_test_key value"

	fmt.Printf("Starting benchmark: %d writes per format\n\n", rounds)

	benchmarks := []struct {
		name     string
		filename string
		fn       SerializeFunc
	}{
		{"Text", "test_format_text.txt", serializeText},
		{"Binary", "test_format_binary.bin", serializeBinary},
		{"JSON", "test_format_json.json", serializeJSON},
		{"RESP", "test_format_resp.resp", serializeRESP},
	}

	for _, b := range benchmarks {
		if err := runBench(b.name, filepath.Join(dir, b.filename), rounds, dataCmd, b.fn); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] benchmark failed: %v\n", b.name, err)
		}
	}

	fmt.Println("Done")
}

func runBench(name, path string, rounds int, data string, serialize SerializeFunc) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old file: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	start := time.Now()
	for i := 0; i < rounds; i++ {
		payload, err := serialize(data)
		if err != nil {
			return fmt.Errorf("serialize at round %d: %w", i, err)
		}

		if _, err := file.Write(payload); err != nil {
			return fmt.Errorf("write at round %d: %w", i, err)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	elapsed := time.Since(start)

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	fmt.Printf("[%-6s Format]   Time: %-14v Size: %d bytes\n", name, elapsed, fileInfo.Size())
	return nil
}

func serializeText(data string) ([]byte, error) {
	return append([]byte(data), '\n'), nil
}

func serializeBinary(data string) ([]byte, error) {
	dataBytes := []byte(data)
	cmdLen := uint32(len(dataBytes))
	buf := make([]byte, 4+int(cmdLen))
	binary.LittleEndian.PutUint32(buf[0:4], cmdLen)
	copy(buf[4:], dataBytes)
	return buf, nil
}

func serializeJSON(data string) ([]byte, error) {
	cmd := Command{Cmd: data}
	jsonBytes, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	return append(jsonBytes, '\n'), nil
}

func serializeRESP(data string) ([]byte, error) {
	parts := strings.Split(data, " ")
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(parts)))
	b.WriteString("\r\n")
	for _, p := range parts {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(p)))
		b.WriteString("\r\n")
		b.WriteString(p)
		b.WriteString("\r\n")
	}
	return []byte(b.String()), nil
}
