package aof

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"time"
)

type AOF struct {
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
	quit   chan struct{}
	closed bool
}

func NewAOF(path string) (*AOF, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	a := &AOF{
		file:   file,
		writer: bufio.NewWriter(file),
		quit:   make(chan struct{}),
	}

	go a.syncLoop()

	return a, nil
}

func (a *AOF) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return nil
	}
	a.closed = true
	close(a.quit)

	a.writer.Flush()
	a.file.Sync()
	return a.file.Close()
}

func (a *AOF) Write(command string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return os.ErrClosed
	}

	if !strings.HasSuffix(command, "\n") {
		command += "\n"
	}

	_, err := a.writer.WriteString(command)
	return err
}

func (a *AOF) syncLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.mu.Lock()
			if !a.closed {
				a.writer.Flush()
				a.file.Sync()
			}
			a.mu.Unlock()
		case <-a.quit:
			return
		}
	}
}

func ReadAll(path string, callback func(line string)) error {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		callback(scanner.Text())
	}

	return scanner.Err()
}
