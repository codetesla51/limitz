package main

import (
	"fmt"
	"time"

	"github.com/codetesla51/limitz/algorithms"
	"github.com/codetesla51/limitz/store"
)

func main() {
	m := store.NewMemoryStore()
	sw := algorithms.NewSlidingWindow(5, 10*time.Second, m)

	userID := "user123"

	for i := 1; i <= 20; i++ {
		allowed := sw.Allow(userID)
		if allowed {
			fmt.Printf("Request %d for %s: Allowed\n", i, userID)
		} else {
			fmt.Printf("Request %d for %s: Denied\n", i, userID)
		}
		time.Sleep(200 * time.Millisecond)
		if i == 5 {
			time.Sleep(11 * time.Second) // Wait to reset the window
		}
	}

}
