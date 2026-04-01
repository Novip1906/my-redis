package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

type Command struct {
	Cmd string `json:"cmd"`
}

type SerializeFunc func(w io.Writer, data string) error

type DeserializeFunc func(br *bufio.Reader) (string, error)

var testData = []string{
	"SET key:1 value:100",
	"SET ключ:1 значение:100",
	"SET long_key:123 \"Very long string with spaces and many words to test length prefixing\"",
	"SET big_key:1 \"" + strings.Repeat("This is a large value to test how different formats handle bigger payloads. ", 20) + "\"",
	"SET huge_key:2 \"" + strings.Repeat("DATA_", 100) + "\"",
	"GET key:1",
	"DEL ключ:1",
}

type formatBench struct {
	name        string
	wrapWriter  func(io.Writer) io.WriteCloser
	wrapReader  func(io.Reader) (io.ReadCloser, error)
	serialize   SerializeFunc
	deserialize DeserializeFunc
}

var formats = []formatBench{
	{"Text", nil, nil, serializeText, deserializeText},
	{"Binary", nil, nil, serializeBinary, deserializeBinary},
	{"JSON", nil, nil, serializeJSON, deserializeJSON},
	{"RESP", nil, nil, serializeRESP, deserializeRESP},
	{"GzipBinary",
		func(w io.Writer) io.WriteCloser { return gzip.NewWriter(w) },
		func(r io.Reader) (io.ReadCloser, error) { return gzip.NewReader(r) },
		serializeBinary, deserializeBinary},
}

func BenchmarkFormats_Write(b *testing.B) {
	for _, f := range formats {
		b.Run(f.name, func(b *testing.B) {
			file, err := os.CreateTemp("", "bench-format-write-*.dat")
			if err != nil {
				b.Fatal(err)
			}
			path := file.Name()
			defer os.Remove(path)

			bw := bufio.NewWriter(file)
			var w io.Writer = bw
			var wc io.Closer
			if f.wrapWriter != nil {
				wr := f.wrapWriter(bw)
				w = wr
				wc = wr
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				data := testData[i%len(testData)]
				if err := f.serialize(w, data); err != nil {
					b.Fatalf("serialize error at %d: %v", i, err)
				}
			}

			if wc != nil {
				wc.Close()
			}
			bw.Flush()
			file.Sync()

			b.StopTimer()

			info, err := file.Stat()
			if err == nil {
				b.ReportMetric(float64(info.Size())/float64(b.N), "disk_bytes/op")
			}
			file.Close()
		})
	}
}

func BenchmarkFormats_Read(b *testing.B) {
	for _, f := range formats {
		b.Run(f.name, func(b *testing.B) {
			b.StopTimer()

			wfile, err := os.CreateTemp("", "bench-format-read-*.dat")
			if err != nil {
				b.Fatal(err)
			}
			path := wfile.Name()
			defer os.Remove(path)

			bw := bufio.NewWriter(wfile)
			var w io.Writer = bw
			var wc io.Closer
			if f.wrapWriter != nil {
				wr := f.wrapWriter(bw)
				w = wr
				wc = wr
			}

			for i := 0; i < b.N; i++ {
				data := testData[i%len(testData)]
				if err := f.serialize(w, data); err != nil {
					b.Fatalf("serialize setup error at %d: %v", i, err)
				}
			}

			if wc != nil {
				wc.Close()
			}
			bw.Flush()
			wfile.Sync()
			wfile.Close()

			b.StartTimer()

			rfile, err := os.Open(path)
			if err != nil {
				b.Fatal(err)
			}
			defer rfile.Close()

			br := bufio.NewReader(rfile)
			var r io.Reader = br
			var rc io.Closer
			if f.wrapReader != nil {
				wr, err := f.wrapReader(br)
				if err != nil {
					b.Fatalf("wrap reader error: %v", err)
				}
				r = wr
				rc = wr
			}
			brTarget := bufio.NewReader(r)

			for i := 0; i < b.N; i++ {
				_, err := f.deserialize(brTarget)
				if err != nil {
					b.Fatalf("deserialize error at %d: %v", i, err)
				}
			}

			if rc != nil {
				rc.Close()
			}
		})
	}
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
