package main

import (
	"fmt"
	"time"

	"github.com/codetesla51/limitz/algorithms"
	"github.com/codetesla51/limitz/store"
)

func main() {
	m := store.NewMemoryStore()
	fw := algorithms.NewFixedWindow(5, 1*time.Second, m)
	for i := 0; i < 10; i++ {
		if fw.Allow("user1") {
			fmt.Println("Request", i+1, "allowed")
		} else {
			fmt.Println("Request", i+1, "denied")
		}
	}

}
