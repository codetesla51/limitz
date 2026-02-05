package main

import (
	"fmt"

	"github.com/codetesla51/limitz/algorithms"
	"github.com/codetesla51/limitz/store"
)

func main() {
	m := store.NewMemoryStore()
	for i := 0; i < 7; i++ {
		result, err := algorithms.NewTokenBucket(5, 1, m).Allow("user1")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Allowed: %v, Limit: %d, Remaining: %d, RetryAfter: %v\n",
			result.Allowed, result.Limit, result.Remaining, result.RetryAfter)
	}
}
