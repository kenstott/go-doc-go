package cache

import (
	"testing"
	"time"
)

func TestNewLRUCache(t *testing.T) {
	c := NewLRUCache(10, 60)
	if c == nil {
		t.Fatal("Failed to create cache")
	}
	if c.Size() != 0 {
		t.Errorf("New cache should be empty, got size %d", c.Size())
	}
}

func TestSetAndGet(t *testing.T) {
	c := NewLRUCache(10, 60)

	// Test basic set and get
	c.Set("key1", "value1")
	val, found := c.Get("key1")
	if !found {
		t.Error("Key should exist")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	// Test non-existent key
	_, found = c.Get("nonexistent")
	if found {
		t.Error("Non-existent key should not be found")
	}
}

func TestLRUEviction(t *testing.T) {
	c := NewLRUCache(3, 60)

	// Fill cache
	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	// Access key1 to make it recently used
	c.Get("key1")

	// Add new item, should evict key2 (least recently used)
	c.Set("key4", "value4")

	// key1 should still exist (recently accessed)
	_, found := c.Get("key1")
	if !found {
		t.Error("key1 should still exist")
	}

	// key2 should be evicted
	_, found = c.Get("key2")
	if found {
		t.Error("key2 should have been evicted")
	}

	// key3 and key4 should exist
	_, found = c.Get("key3")
	if !found {
		t.Error("key3 should exist")
	}
	_, found = c.Get("key4")
	if !found {
		t.Error("key4 should exist")
	}
}

func TestTTLExpiration(t *testing.T) {
	c := NewLRUCache(10, 1) // 1 second TTL

	c.Set("key1", "value1")

	// Should exist immediately
	_, found := c.Get("key1")
	if !found {
		t.Error("Key should exist immediately after setting")
	}

	// Wait for expiration
	time.Sleep(1100 * time.Millisecond)

	// Should be expired now
	_, found = c.Get("key1")
	if found {
		t.Error("Key should have expired")
	}
}

func TestUpdateExistingKey(t *testing.T) {
	c := NewLRUCache(10, 60)

	c.Set("key1", "value1")
	c.Set("key1", "value2")

	val, found := c.Get("key1")
	if !found {
		t.Error("Key should exist")
	}
	if val != "value2" {
		t.Errorf("Expected updated value 'value2', got %v", val)
	}

	// Size should still be 1
	if c.Size() != 1 {
		t.Errorf("Cache size should be 1, got %d", c.Size())
	}
}

func TestClear(t *testing.T) {
	c := NewLRUCache(10, 60)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	if c.Size() != 3 {
		t.Errorf("Cache should have 3 items, got %d", c.Size())
	}

	c.Clear()

	if c.Size() != 0 {
		t.Errorf("Cache should be empty after clear, got size %d", c.Size())
	}

	_, found := c.Get("key1")
	if found {
		t.Error("Cache should not contain any keys after clear")
	}
}

func TestCleanupExpired(t *testing.T) {
	c := NewLRUCache(10, 1) // 1 second TTL

	// Add items at different times
	c.Set("key1", "value1")
	time.Sleep(500 * time.Millisecond)
	c.Set("key2", "value2")

	// Wait for key1 to expire but not key2
	time.Sleep(600 * time.Millisecond)

	// Clean up expired entries
	c.CleanupExpired()

	// key1 should be gone
	_, found := c.Get("key1")
	if found {
		t.Error("key1 should have been cleaned up")
	}

	// key2 should still exist
	_, found = c.Get("key2")
	if !found {
		t.Error("key2 should still exist")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := NewLRUCache(100, 60)
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			c.Set(string(rune(i)), i)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			c.Get(string(rune(i)))
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	// Cache should still be functional
	c.Set("test", "value")
	val, found := c.Get("test")
	if !found {
		t.Error("Cache should still be functional after concurrent access")
	}
	if val != "value" {
		t.Errorf("Expected 'value', got %v", val)
	}
}

func TestVariousDataTypes(t *testing.T) {
	c := NewLRUCache(10, 60)

	// Test with different types
	c.Set("string", "hello")
	c.Set("int", 42)
	c.Set("float", 3.14)
	c.Set("bool", true)
	c.Set("map", map[string]interface{}{"nested": "value"})
	c.Set("slice", []int{1, 2, 3})

	// Verify all types can be retrieved
	val, _ := c.Get("string")
	if val != "hello" {
		t.Errorf("String value mismatch")
	}

	val, _ = c.Get("int")
	if val != 42 {
		t.Errorf("Int value mismatch")
	}

	val, _ = c.Get("float")
	if val != 3.14 {
		t.Errorf("Float value mismatch")
	}

	val, _ = c.Get("bool")
	if val != true {
		t.Errorf("Bool value mismatch")
	}

	val, _ = c.Get("map")
	if m, ok := val.(map[string]interface{}); ok {
		if m["nested"] != "value" {
			t.Errorf("Map value mismatch")
		}
	} else {
		t.Error("Failed to retrieve map")
	}

	val, _ = c.Get("slice")
	if s, ok := val.([]int); ok {
		if len(s) != 3 || s[0] != 1 {
			t.Errorf("Slice value mismatch")
		}
	} else {
		// JSON unmarshaling might return []interface{}
		if s, ok := val.([]interface{}); ok {
			if len(s) != 3 {
				t.Errorf("Slice length mismatch")
			}
		} else {
			t.Error("Failed to retrieve slice")
		}
	}
}