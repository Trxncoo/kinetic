package kcache

import (
	"math/rand/v2"
	"strconv"
	"testing"
)

const benchKeyCount = 10_000

func benchKeys() []string {
	keys := make([]string, benchKeyCount)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	return keys
}

func BenchmarkCache_Get(b *testing.B) {
	c := NewCache[string, int]()
	keys := benchKeys()
	for i, k := range keys {
		c.Set(k, i, 0)
	}

	b.ReportAllocs()
	i := 0
	for b.Loop() {
		c.Get(keys[i%len(keys)])
		i++
	}
}

func BenchmarkCache_Get_Parallel(b *testing.B) {
	c := NewCache[string, int]()
	keys := benchKeys()
	for i, k := range keys {
		c.Set(k, i, 0)
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Get(keys[rand.IntN(len(keys))])
		}
	})
}

func BenchmarkCache_Set(b *testing.B) {
	c := NewCache[string, int]()
	keys := benchKeys()

	b.ReportAllocs()
	i := 0
	for b.Loop() {
		c.Set(keys[i%len(keys)], i, 0)
		i++
	}
}

func BenchmarkCache_Set_Parallel(b *testing.B) {
	c := NewCache[string, int]()
	keys := benchKeys()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set(keys[rand.IntN(len(keys))], 1, 0)
		}
	})
}
