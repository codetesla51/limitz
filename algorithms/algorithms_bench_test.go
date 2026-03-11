package algorithms

import (
	"context"
	"testing"
	"time"

	"github.com/codetesla51/limitz/store"
)

func BenchmarkFixedWindowAllow(b *testing.B) {
	s := store.NewMemoryStore()
	fw := NewFixedWindow(100, 1*time.Second, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fw.Allow(ctx, "user1")
	}
}

func BenchmarkFixedWindowMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	fw := NewFixedWindow(100, 1*time.Second, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fw.Allow(ctx, "user"+string(rune(i%1000)))
	}
}

func BenchmarkLeakyBucketAllow(b *testing.B) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(100, 10, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.Allow(ctx, "user1")
	}
}

func BenchmarkLeakyBucketMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(100, 10, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.Allow(ctx, "user"+string(rune(i%1000)))
	}
}

func BenchmarkSlidingWindowCounterAllow(b *testing.B) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(100, 1*time.Second, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		swc.Allow(ctx, "user1")
	}
}

func BenchmarkSlidingWindowCounterMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(100, 1*time.Second, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		swc.Allow(ctx, "user"+string(rune(i%1000)))
	}
}

func BenchmarkSlidingWindowAllow(b *testing.B) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(100, 1*time.Second, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Allow(ctx, "user1")
	}
}

func BenchmarkSlidingWindowMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(100, 1*time.Second, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Allow(ctx, "user"+string(rune(i%1000)))
	}
}

func BenchmarkTokenBucketAllow(b *testing.B) {
	s := store.NewMemoryStore()
	tb := NewTokenBucket(100, 10, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow(ctx, "user1")
	}
}

func BenchmarkTokenBucketMultipleUsers(b *testing.B) {
	s := store.NewMemoryStore()
	tb := NewTokenBucket(100, 10, s)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow(ctx, "user"+string(rune(i%1000)))
	}
}

func BenchmarkFixedWindowConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	fw := NewFixedWindow(10000, 1*time.Second, s)
	ctx := context.Background()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fw.Allow(ctx, "user1")
		}
	})
}

func BenchmarkLeakyBucketConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(10000, 100, s)
	ctx := context.Background()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lb.Allow(ctx, "user1")
		}
	})
}

func BenchmarkSlidingWindowCounterConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(10000, 1*time.Second, s)
	ctx := context.Background()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			swc.Allow(ctx, "user1")
		}
	})
}

func BenchmarkSlidingWindowConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(10000, 1*time.Second, s)
	ctx := context.Background()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sw.Allow(ctx, "user1")
		}
	})
}

func BenchmarkTokenBucketConcurrent(b *testing.B) {
	s := store.NewMemoryStore()
	tb := NewTokenBucket(10000, 100, s)
	ctx := context.Background()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow(ctx, "user1")
		}
	})
}
