package kcache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestCache_GetReturnsSetValue(t *testing.T) {
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

func TestCache_DeleteRemovesKey(t *testing.T) {
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

func TestCache_LenCountsExpiredUntouchedEntries(t *testing.T) {
	c := NewCache[string, int]()

	c.Set("a", 1, 15*time.Millisecond)
	time.Sleep(40 * time.Millisecond)

	// Nothing has called Get or Delete on "a" since it expired, so lazy
	// removal hasn't happened yet — Len must still count it, per its own
	// documented behavior.
	if n := c.Len(); n != 1 {
		t.Fatalf("Len() = %d, want 1 (expired but untouched, still counted)", n)
	}
}

func TestCache_LenCountsEntries(t *testing.T) {
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

// Documented, not a kcache bug, for all three of Get/Set/Delete: the
// panic is centralized in shardFor, which every one of them calls first.
// comparable permits interface types, but hashing/comparing a
// non-comparable dynamic value inside one still panics at runtime.

func TestCache_SetWithNonComparableDynamicKeyPanics(t *testing.T) {
	c := NewCache[any, string]()

	defer func() {
		if recover() == nil {
			t.Fatal("expected Set with a non-comparable dynamic value to panic")
		}
	}()
	c.Set([]int{1, 2, 3}, "boom", 0)
}

func TestCache_GetWithNonComparableDynamicKeyPanics(t *testing.T) {
	c := NewCache[any, string]()

	defer func() {
		if recover() == nil {
			t.Fatal("expected Get with a non-comparable dynamic value to panic")
		}
	}()
	c.Get([]int{1, 2, 3})
}

func TestCache_DeleteWithNonComparableDynamicKeyPanics(t *testing.T) {
	c := NewCache[any, string]()

	defer func() {
		if recover() == nil {
			t.Fatal("expected Delete with a non-comparable dynamic value to panic")
		}
	}()
	c.Delete([]int{1, 2, 3})
}

func TestCache_ConcurrentGetSetDelete(t *testing.T) {
	c := NewCache[int, int]()

	var wg sync.WaitGroup
	for g := range 50 {
		wg.Go(func() { c.Set(g, g, 0) })
		wg.Go(func() { c.Get(g) })
		wg.Go(func() { c.Delete(g) })
	}
	wg.Wait()
}
