package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Command struct {
	Cmd string `json:"cmd"`
}

func main() {
	dir := os.TempDir()
	dir = "/tmp/myredis-bench"
	os.MkdirAll(dir, 0777)

	rounds := 1000000
	dataCmd := "SET my_test_key value"

	fmt.Printf("Starting benchmark: %d concurrent writes per format\n\n", rounds)

	benchTextFormat(dir, rounds, dataCmd)
	benchBinaryFormat(dir, rounds, dataCmd)
	benchJSONFormat(dir, rounds, dataCmd)
	benchRESPFormat(dir, rounds, dataCmd)

	fmt.Println("Done")
}

func benchTextFormat(dir string, rounds int, data string) {
	path := filepath.Join(dir, "test_format_text.txt")
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

			dataWithNewline := data
			if !strings.HasSuffix(dataWithNewline, "\n") {
				dataWithNewline += "\n"
			}

			mu.Lock()
			file.WriteString(dataWithNewline)
			mu.Unlock()
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fileInfo, _ := file.Stat()
	fmt.Printf("[Text Format]   Time: %v \tSize: %d bytes\n", elapsed, fileInfo.Size())
}

func benchBinaryFormat(dir string, rounds int, data string) {
	path := filepath.Join(dir, "test_format_binary.bin")
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

