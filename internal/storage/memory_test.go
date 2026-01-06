package storage

import (
	"sync"
	"testing"
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
