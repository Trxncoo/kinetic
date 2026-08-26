package kcache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestCache_SetGet(t *testing.T) {
	c := NewCache[string, int]()

	if _, ok := c.Get("a"); ok {
		t.Fatal("Get on empty cache: ok = true, want false")
	}

	c.Set("a", 42, 0)
	got, ok := c.Get("a")
	if !ok {
		t.Fatal("Get: ok = false, want true")
	}
	if got != 42 {
		t.Fatalf("Get: got %d, want 42", got)
	}
}

func TestCache_MissReturnsZeroValue(t *testing.T) {
	c := NewCache[string, int]()

	got, ok := c.Get("missing")
	if ok {
		t.Fatal("Get: ok = true, want false")
	}
	if got != 0 {
		t.Fatalf("Get: got %d, want zero value 0", got)
	}
}

func TestCache_SetOverwritesExistingKey(t *testing.T) {
	c := NewCache[string, int]()

	c.Set("a", 1, 0)
	c.Set("a", 2, 0)

	got, ok := c.Get("a")
	if !ok || got != 2 {
		t.Fatalf("Get: got (%d, %v), want (2, true)", got, ok)
	}
	if n := c.Len(); n != 1 {
		t.Fatalf("Len() = %d, want 1 (overwrite, not a second entry)", n)
	}
}

func TestCache_Delete(t *testing.T) {
	c := NewCache[string, int]()

	c.Set("a", 1, 0)
	c.Delete("a")

	if _, ok := c.Get("a"); ok {
		t.Fatal("Get after Delete: ok = true, want false")
	}

	c.Delete("never-existed") // must not panic
}

func TestCache_NonPositiveTTLNeverExpires(t *testing.T) {
	c := NewCache[string, int]()

	c.Set("zero", 1, 0)
	c.Set("negative", 2, -time.Second)
	time.Sleep(20 * time.Millisecond)

	if _, ok := c.Get("zero"); !ok {
		t.Fatal("Get(zero): ok = false, want true (ttl<=0 means permanent)")
	}
	if _, ok := c.Get("negative"); !ok {
		t.Fatal("Get(negative): ok = false, want true (ttl<=0 means permanent)")
	}
}

func TestCache_EntryExpiresAndIsRemoved(t *testing.T) {
	c := NewCache[string, int]()

	c.Set("a", 1, 15*time.Millisecond)
	if _, ok := c.Get("a"); !ok {
		t.Fatal("Get before ttl elapses: ok = false, want true")
	}

	time.Sleep(40 * time.Millisecond)

	if _, ok := c.Get("a"); ok {
		t.Fatal("Get after ttl elapses: ok = true, want false")
	}
	if n := c.Len(); n != 0 {
		t.Fatalf("Len() after expiry+Get = %d, want 0 (Get lazily removes expired entries)", n)
	}
}

func TestCache_Len(t *testing.T) {
	c := NewCache[string, int]()

	for i := range 50 {
		c.Set(strconv.Itoa(i), i, 0)
	}
	if n := c.Len(); n != 50 {
		t.Fatalf("Len() = %d, want 50", n)
	}

	c.Delete("0")
	if n := c.Len(); n != 49 {
		t.Fatalf("Len() after delete = %d, want 49", n)
	}
}

func TestCache_GenericOverStructKeys(t *testing.T) {
	type key struct {
		tenant string
		id     int
	}

	c := NewCache[key, string]()
	c.Set(key{"acme", 1}, "widget", 0)

	got, ok := c.Get(key{"acme", 1})
	if !ok || got != "widget" {
		t.Fatalf("Get: got (%q, %v), want (\"widget\", true)", got, ok)
	}
	if _, ok := c.Get(key{"acme", 2}); ok {
		t.Fatal("Get with a different key: ok = true, want false")
	}
}

func TestCache_ConcurrentGetSetDelete(t *testing.T) {
	c := NewCache[int, int]()

	var wg sync.WaitGroup
	for g := range 50 {
		wg.Add(3)
		go func(g int) {
			defer wg.Done()
			c.Set(g, g, 0)
		}(g)
		go func(g int) {
			defer wg.Done()
			c.Get(g)
		}(g)
		go func(g int) {
			defer wg.Done()
			c.Delete(g)
		}(g)
	}
	wg.Wait()
}
