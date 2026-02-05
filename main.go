package main

import (
	"fmt"

	"github.com/codetesla51/limitz/algorithms"
)

func main() {
	lb := &algorithms.LeakyBucket{
		Capacity: 5,
		Rate:     5,
		Buckets:  make(map[string]*algorithms.LeakyBucketUser),
	}
	for i := 0; i < 10; i++ {
		if lb.Allow("user1") {
			fmt.Printf("allow request for %+v\n", lb.Buckets["user1"])
		} else {
			fmt.Println("deny request")
		}

	}
}
