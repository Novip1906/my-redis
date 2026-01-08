package storage

import (
	"sync"
	"testing"
	"time"
)

func TestMemoryStorage_SetGet(t *testing.T) {
	s := NewMemoryStorage()

	tests := []struct {
		name      string
		key       string
		value     string
		wantValue string
		wantOk    bool
	}{
		{"Simple Set", "user:1", "Alice", "Alice", true},
		{"Empty Value", "empty", "", "", true},
		{"Non-existent", "ghost", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantOk {
				s.Set(tt.key, tt.value)
			}

			gotValue, gotOk := s.Get(tt.key)

			if gotOk != tt.wantOk {
				t.Errorf("Get() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotOk && gotValue != tt.wantValue {
				t.Errorf("Get() val = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	s := NewMemoryStorage()
	s.Set("key", "val")
	s.Delete("key")

	_, ok := s.Get("key")
	if ok {
		t.Error("Expected key to be deleted, but it was found")
	}
}

func TestMemoryStorage_TTL(t *testing.T) {
	s := NewMemoryStorage()

	s.Set("key", "val")
	s.Set("without", "ttl")
	s.SetTTL("key", 1)

	seconds := s.GetTTL("key")
	if seconds <= 0 {
		t.Errorf("TTL() seconds = %v, want >0", seconds)
	}

	seconds = s.GetTTL("without")
	if seconds != -1 {
		t.Errorf("TTL() seconds = %v, want -1", seconds)
	}

	time.Sleep(1100 * time.Millisecond)

	_, ok := s.Get("key")
	if ok {
		t.Error("GET() ok = true, want false")
	}

	seconds = s.GetTTL("key")
	if seconds != -2 {
		t.Errorf("TTL() seconds = %v, want -2", seconds)
	}
}

func TestMemoryStorage_Increment(t *testing.T) {
	s := NewMemoryStorage()

	s.Set("one", "0")

	for i := 0; i < 3; i++ {
		res, err := s.Increment("one")
		if err != nil {
			t.Error("Increment error", "error", err)
		}
		if res != int64(i+1) {
			t.Errorf("res = %v, want %v", res, i+1)
		}
	}

	res, err := s.Increment("zero")
	if err != nil {
		t.Error("Increment error", "error", err)
	}
	if res != 1 {
		t.Errorf("res = %v, want 1", res)
	}

	s.Set("str", "string")
	_, err = s.Increment("str")
	if err == nil {
		t.Error("err is null, expected parse error")
	}
}

func TestMemoryStorage_Flush(t *testing.T) {
	s := NewMemoryStorage()

	s.Set("key", "value")

	s.Flush()

	_, ok := s.Get("key")
	if ok {
		t.Error("DB is not empty afret FLUSH command")
	}
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	s := NewMemoryStorage()
	var wg sync.WaitGroup

	iterations := 1000

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			s.Set("key", "value")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, _ = s.Get("key")
		}
	}()

	wg.Wait()
}
