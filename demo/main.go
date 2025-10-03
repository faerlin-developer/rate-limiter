package main

import (
	"fmt"
	"github.com/faerlin-developer/rate-limiter.git/db"
	"github.com/faerlin-developer/rate-limiter.git/ratelimit"
)

func main() {

	database, err := db.NewInMemoryStore[string, ratelimit.Bucket](10)
	if err != nil {
		panic(err)
	}

	var limiter ratelimit.Limiter
	limiter, err = ratelimit.NewTBLimiter(100, database)

	fmt.Printf("%v\n", limiter)

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
