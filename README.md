# sfcache

This is a caching and cache-filling library with the ability to set
expire times on values.

``` go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/silas/sfcache"
)

func main() {
	loader := func(ctx context.Context, key interface{}) (interface{}, time.Time, error) {
		value := fmt.Sprintf("%s-%d", key, time.Now().Unix())
		expireTime := time.Now().Add(3 * time.Second)
		return value, expireTime, nil
	}

	c, err := sfcache.New(100, loader)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		v, err := c.Load(context.Background(), "foo")
		if err != nil {
			panic(err)
		}

		fmt.Println(v)

		time.Sleep(time.Second)
	}
}
```
