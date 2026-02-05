package main

import (
	"github.com/codetesla51/limitz/algorithms"
)

func main() {
	tb := &algorithms.TokenBucket{
		Capacity:   3,
		RefillRate: 0,
		Buckets:    map[string]*algorithms.Buckets{},
	}
	for i := 0; i < 5; i++ {
		allowed := tb.Allow("user123")
		if allowed {
			println("Request", i+1, "allowed for user123")
		} else {
			println("Request", i+1, "denied for user123")
		}
		err := tb.Reset("user123")
		if err != nil {
			println("Error resetting token bucket:", err.Error())
		} else {
			println("Token bucket reset for user123")
		}
	}

}
