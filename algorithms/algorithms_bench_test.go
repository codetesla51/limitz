package algorithms

import (
	"testing"
	"time"

	"github.com/codetesla51/limitz/store"
)

// Fixed Window Benchmarks
func BenchmarkFixedWindowAllow(b *testing.B) {
	s := store.NewMemoryStore()
	fw := NewFixedWindow(100, 1*time.Second, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fw.Allow("user1")
	}
}

func BenchmarkFixedWindowMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	fw := NewFixedWindow(100, 1*time.Second, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fw.Allow("user" + string(rune(i%1000)))
	}
}

// Leaky Bucket Benchmarks
func BenchmarkLeakyBucketAllow(b *testing.B) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(100, 10, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.Allow("user1")
	}
}

func BenchmarkLeakyBucketMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(100, 10, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.Allow("user" + string(rune(i%1000)))
	}
}

// Sliding Window Counter Benchmarks
func BenchmarkSlidingWindowCounterAllow(b *testing.B) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(100, 1*time.Second, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		swc.Allow("user1")
	}
}

func BenchmarkSlidingWindowCounterMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(100, 1*time.Second, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		swc.Allow("user" + string(rune(i%1000)))
	}
}

// Sliding Window Benchmarks
func BenchmarkSlidingWindowAllow(b *testing.B) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(100, 1*time.Second, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Allow("user1")
	}
}

func BenchmarkSlidingWindowMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(100, 1*time.Second, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Allow("user" + string(rune(i%1000)))
	}
}

// Token Bucket Benchmarks
func BenchmarkTokenBucketAllow(b *testing.B) {
	s := store.NewMemoryStore()
	tb := NewTokenBucket(100, 10, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow("user1")
	}
}

func BenchmarkTokenBucketMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	tb := NewTokenBucket(100, 10, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow("user" + string(rune(i%1000)))
	}
}

// Concurrent Benchmarks
func BenchmarkFixedWindowConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	fw := NewFixedWindow(10000, 1*time.Second, s)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fw.Allow("user1")
		}
	})
}

func BenchmarkLeakyBucketConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(10000, 100, s)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.Allow("user1")
		}
	})
}

func BenchmarkSlidingWindowCounterConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(10000, 1*time.Second, s)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			swc.Allow("user1")
		}
	})
}

func BenchmarkSlidingWindowConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(10000, 1*time.Second, s)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sw.Allow("user1")
		}
	})
}

func BenchmarkTokenBucketConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	tb := NewTokenBucket(10000, 100, s)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow("user1")
		}
	})
}
