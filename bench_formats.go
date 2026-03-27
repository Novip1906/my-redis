package main

import (
	"bufio"
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
	dir := "/tmp/myredis-bench"
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
	os.Remove(path)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var mu sync.Mutex

	start := time.Now()
	for i := 0; i < rounds; i++ {
		go func() {
			defer wg.Done()

			dataWithNewline := data
			if !strings.HasSuffix(dataWithNewline, "\n") {
				dataWithNewline += "\n"
			}

			mu.Lock()
			file.WriteString(dataWithNewline)
			mu.Unlock()
		}()
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	elapsed := time.Since(start)

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	defer file.Close()

	var mu sync.Mutex

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(rounds)
	for i := 0; i < rounds; i++ {
		go func() {
			defer wg.Done()

			dataBytes := []byte(data)
			cmdLen := uint32(len(dataBytes))
			buf := make([]byte, 4+int(cmdLen))
			binary.LittleEndian.PutUint32(buf[0:4], cmdLen)
			copy(buf[4:], dataBytes)

			mu.Lock()
			file.Write(buf)
			mu.Unlock()
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fileInfo, _ := file.Stat()
	fmt.Printf("[Binary Format] Time: %v \tSize: %d bytes\n", elapsed, fileInfo.Size())
}

func benchJSONFormat(dir string, rounds int, data string) {
	path := filepath.Join(dir, "test_format_json.json")
	os.Remove(path)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var mu sync.Mutex

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(rounds)
	for i := 0; i < rounds; i++ {
		go func() {
			defer wg.Done()

			cmd := Command{Cmd: data}
			jsonBytes, _ := json.Marshal(cmd)
			jsonBytes = append(jsonBytes, '\n')

			mu.Lock()
			file.Write(jsonBytes)
			mu.Unlock()
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fileInfo, _ := file.Stat()
	fmt.Printf("[JSON Format]   Time: %v \tSize: %d bytes\n", elapsed, fileInfo.Size())
}

func benchRESPFormat(dir string, rounds int, data string) {
	path := filepath.Join(dir, "test_format_resp.resp")
	os.Remove(path)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}
	jsonBytes = append(jsonBytes, '\n')
	return jsonBytes, nil
}

	var mu sync.Mutex

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(rounds)
	for i := 0; i < rounds; i++ {
		go func() {
			defer wg.Done()

			parts := strings.Split(data, " ")
			respData := "*" + strconv.Itoa(len(parts)) + "\r\n"
			for _, p := range parts {
				respData += "$" + strconv.Itoa(len(p)) + "\r\n" + p + "\r\n"
			}
			respBytes := []byte(respData)

			mu.Lock()
			file.Write(respBytes)
			mu.Unlock()
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fileInfo, _ := file.Stat()
	fmt.Printf("[RESP Format]   Time: %v \tSize: %d bytes\n", elapsed, fileInfo.Size())
}

