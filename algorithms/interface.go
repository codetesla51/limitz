package algorithms

type RateLimiter interface {
	Allow(key string) bool
	Reset(key string) error
}
