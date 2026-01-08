package storage

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Item struct {
	Value     string
	ExpiresAt int64
}

type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string]Item
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]Item),
	}
}

func (s *MemoryStorage) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = Item{
		Value:     value,
		ExpiresAt: -1,
	}
}

func (s *MemoryStorage) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.data[key]
	if !ok {
		return "", false
	}

	if item.ExpiresAt > 0 && time.Now().Unix() >= item.ExpiresAt {
		delete(s.data, key)
		return "", false
	}

	return item.Value, ok
}

func (s *MemoryStorage) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *MemoryStorage) Expire(key string, seconds int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.data[key]
	if !ok {
		return false
	}

	if item.ExpiresAt > 0 && time.Now().Unix() >= item.ExpiresAt {
		delete(s.data, key)
		return false
	}

	item.ExpiresAt = time.Now().Add(time.Duration(seconds) * time.Second).Unix()
	s.data[key] = item
	return true
}
func (s *MemoryStorage) TTL(key string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.data[key]
	if !ok {
		return -2
	}

	if item.ExpiresAt == -1 {
		return -1
	}

	now := time.Now().Unix()

	if now >= item.ExpiresAt {
		delete(s.data, key)
		return -2
	}

	return item.ExpiresAt - now
}

func (s *MemoryStorage) Increment(key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.data[key]
	if !ok || item.ExpiresAt > 0 && time.Now().Unix() >= item.ExpiresAt {
		s.data[key] = Item{
			Value:     "1",
			ExpiresAt: -1,
		}
		return 1, nil
	}

	value, err := strconv.ParseInt(item.Value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer or out of range")
	}

	item.Value = strconv.FormatInt(value, 10)
	s.data[key] = item

	return int64(value + 1), nil
}
