# sfcache

This is a caching and cache-filling library.

``` go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/silas/sfcache"
)

func main() {
	c, err := sfcache.New(&sfcache.Config{
		Load: func(ctx context.Context, key interface{}) (interface{}, error) {
			return fmt.Sprintf("%s-%d", key, time.Now().Unix()), nil
		},
		MaxAge: 3 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		v, err := c.Get(context.Background(), "foo")
		if err != nil {
			panic(err)
		}

		fmt.Println(v)

		time.Sleep(time.Second)
	}
}
```
