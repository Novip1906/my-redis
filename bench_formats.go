package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Command struct {
	Cmd string `json:"cmd"`
}

type SerializeFunc func(w io.Writer, data string) error

type DeserializeFunc func(br *bufio.Reader) (string, error)

func main() {
	dir := "bench_data"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory %s: %v\n", dir, err)
		os.Exit(1)
	}

	rounds := 1_000_000
	largeValue := strings.Repeat("This is a large value to test how different formats handle bigger payloads. ", 20)
	testData := []string{
		"SET key:1 value:100",
		"SET ключ:1 значение:100",
		"SET long_key:123 \"Very long string with spaces and many words to test length prefixing\"",
		"SET big_key:1 \"" + largeValue + "\"",
		"SET huge_key:2 \"" + strings.Repeat("DATA_", 100) + "\"",
		"GET key:1",
		"DEL ключ:1",
	}

	benchmarks := []struct {
		name        string
		filename    string
		serialize   SerializeFunc
		deserialize DeserializeFunc
	}{
		{"Text", "test_format_text.txt", serializeText, deserializeText},
		{"Binary", "test_format_binary.bin", serializeBinary, deserializeBinary},
		{"JSON", "test_format_json.json", serializeJSON, deserializeJSON},
		{"RESP", "test_format_resp.resp", serializeRESP, deserializeRESP},
	}

	fmt.Printf("=== WRITE: %d records per format ===\n\n", rounds)
	for _, b := range benchmarks {
		path := filepath.Join(dir, b.filename)
		if err := runWriteBench(b.name, path, rounds, testData, b.serialize); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] write benchmark failed: %v\n", b.name, err)
		}
	}

	fmt.Printf("\n=== READ (AOF restore): %d records per format ===\n\n", rounds)
	for _, b := range benchmarks {
		path := filepath.Join(dir, b.filename)
		if err := runReadBench(b.name, path, b.deserialize); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] read benchmark failed: %v\n", b.name, err)
		}
	}

	fmt.Println("\nDone")
}

func runWriteBench(name, path string, rounds int, data []string, serialize SerializeFunc) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old file: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	bw := bufio.NewWriter(file)

	start := time.Now()
	for i := 0; i < rounds; i++ {
		if err := serialize(bw, data[i%len(data)]); err != nil {
			return fmt.Errorf("serialize at round %d: %w", i, err)
		}
	}

	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
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

func runReadBench(name, path string, deserialize DeserializeFunc) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	br := bufio.NewReader(file)
	count := 0
	start := time.Now()
	for {
		_, err := deserialize(br)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("deserialize at record %d: %w", count, err)
		}
		count++
	}
	elapsed := time.Since(start)

	fmt.Printf("[%-6s Format]   Time: %-14v Records: %d\n", name, elapsed, count)
	return nil
}

func serializeText(w io.Writer, data string) error {
	if _, err := io.WriteString(w, data); err != nil {
		return err
	}
	_, err := w.Write([]byte{'\n'})
	return err
}

func serializeBinary(w io.Writer, data string) error {
	cmdLen := uint32(len(data))
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], cmdLen)
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := io.WriteString(w, data)
	return err
}

func serializeJSON(w io.Writer, data string) error {
	cmd := Command{Cmd: data}
	jsonBytes, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	if _, err := w.Write(jsonBytes); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}

func serializeRESP(w io.Writer, data string) error {
	parts := parseCommand(data)
	if _, err := io.WriteString(w, "*"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, strconv.Itoa(len(parts))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	for _, p := range parts {
		if _, err := io.WriteString(w, "$"); err != nil {
			return err
		}
		if _, err := io.WriteString(w, strconv.Itoa(len(p))); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\r\n"); err != nil {
			return err
		}
		if _, err := io.WriteString(w, p); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\r\n"); err != nil {
			return err
		}
	}
	return nil
}

func parseCommand(data string) []string {
	var parts []string
	var b strings.Builder
	inQuotes := false

	for i := 0; i < len(data); i++ {
		c := data[i]
		switch {
		case c == '"':
			inQuotes = !inQuotes
		case c == ' ' && !inQuotes:
			if b.Len() > 0 {
				parts = append(parts, b.String())
				b.Reset()
			}
		default:
			b.WriteByte(c)
		}
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	return parts
}

func deserializeText(br *bufio.Reader) (string, error) {
	line, err := br.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\n"), nil
}

func deserializeBinary(br *bufio.Reader) (string, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(br, lenBuf[:]); err != nil {
		return "", err
	}
	size := binary.LittleEndian.Uint32(lenBuf[:])

	data := make([]byte, size)

	if _, err := io.ReadFull(br, data); err != nil {
		return "", err
	}

	return string(data), nil
}

func deserializeJSON(br *bufio.Reader) (string, error) {
	line, err := br.ReadString('\n')
	if err != nil {
		return "", err
	}
	var cmd Command
	if err := json.Unmarshal([]byte(line), &cmd); err != nil {
		return "", err
	}
	return cmd.Cmd, nil
}

func deserializeRESP(br *bufio.Reader) (string, error) {
	line, err := br.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if len(line) == 0 || line[0] != '*' {
		return "", fmt.Errorf("expected array header, got: %q", line)
	}
	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return "", fmt.Errorf("parse array length: %w", err)
	}

	parts := make([]string, 0, count)
	for i := 0; i < count; i++ {
		hdr, err := br.ReadString('\n')
		if err != nil {
			return "", err
		}
		hdr = strings.TrimSuffix(strings.TrimSuffix(hdr, "\n"), "\r")
		if len(hdr) == 0 || hdr[0] != '$' {
			return "", fmt.Errorf("expected bulk header, got: %q", hdr)
		}
		size, err := strconv.Atoi(hdr[1:])
		if err != nil {
			return "", fmt.Errorf("parse bulk length: %w", err)
		}

		buf := make([]byte, size)

		if _, err := io.ReadFull(br, buf); err != nil {
			return "", err
		}
		if _, err := br.Discard(2); err != nil {
			return "", err
		}
		parts = append(parts, string(buf))
	}

	return strings.Join(parts, " "), nil
}
