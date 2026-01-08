package aof

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

type AOF struct {
	file *os.File
	mu   sync.Mutex
}

func NewAOF(path string) (*AOF, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &AOF{
		file: file,
	}, nil
}

func (a *AOF) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.file.Close()
}

func (a *AOF) Write(command string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !strings.HasSuffix(command, "\n") {
		command += "\n"
	}

	_, err := a.file.WriteString(command)
	if err != nil {
		return err
	}

	return nil
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
