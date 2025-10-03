package main

import (
	"fmt"
	"github.com/faerlin-developer/rate-limiter.git/limiter"
	"github.com/faerlin-developer/rate-limiter.git/store"
)

func main() {

	database, err := store.NewInMemoryStore[string, int](10)
	if err != nil {
		panic(err)
	}

	limiter := limiter.NewTokenBucket(100, database)
	fmt.Println(limiter)

	database.Put("one", 1)
	value, err := database.Get("one")
	if err != nil {
		panic(err)
	}

	fmt.Printf("one: %d\n", value)
}

/**

if err := lim.Allow(ctx, key); err != nil {
    var de *DeniedError
    if errors.As(err, &de) {
        // handle retry-after
    } else {
        // other error
    }
}

*/
