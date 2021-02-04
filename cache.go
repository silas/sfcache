package sfcache

import (
	"context"
	"errors"
	"time"

	lru "github.com/hashicorp/golang-lru"

	"github.com/silas/sfcache/internal/singleflight"
)

type noValue struct{}

var (
	// ErrNotFound is returned when Loader returns a nil or expired value.
	ErrNotFound = errors.New("cache entry not found")

	// NoExpireTime disables expire time for a given value.
	NoExpireTime time.Time

	// NoValue is a value a Loader can return to cache a nil.
	NoValue noValue = struct{}{}
)

// Loader gets data to populate the cache, returning the value and expire time.
type Loader func(ctx context.Context, key interface{}) (interface{}, time.Time, error)

// Cache is an LRU cache with cache filling functionality.
type Cache struct {
	lru   *lru.Cache
	group *singleflight.Group
	load  Loader
}

type entry struct {
	expireTime time.Time
	value      interface{}
}

// New creates an LRU cache with the given size and loader.
func New(size int, load Loader) (*Cache, error) {
	if size < 1 {
		return nil, errors.New("size must be 1 or greater")
	}
	if load == nil {
		return nil, errors.New("loader is required")
	}

	l, err := lru.New(size)
	if err != nil {
		return nil, err
	}

	cache := &Cache{
		lru:   l,
		group: &singleflight.Group{},
		load:  load,
	}
	return cache, nil
}

// Load looks up a key's value from the cache or populates it from Loader if not found.
func (c *Cache) Load(ctx context.Context, key interface{}) (interface{}, error) {
	if v, ok := c.Get(key); ok {
		if v == nil {
			return nil, ErrNotFound
		}
		return v, nil
	}
	return c.do(ctx, key)
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	return c.filter(c.lru.Get(key))
}

// Set sets a value to the cache. Returns false if expired or value is nil.
func (c *Cache) Set(key interface{}, value interface{}, expireTime time.Time) bool {
	if value != nil && (expireTime.IsZero() || time.Now().Before(expireTime)) {
		if expireTime.IsZero() {
			c.lru.Add(key, value)
		} else {
			c.lru.Add(key, &entry{
				expireTime: expireTime,
				value:      value,
			})
		}
		return true
	}
	return false
}

// Peek returns the key's value (or nil if not found) without updating
// the "recently used"-ness of the key.
func (c *Cache) Peek(key interface{}) (interface{}, bool) {
	return c.filter(c.lru.Peek(key))
}

// Delete removes the provided key from the cache.
func (c *Cache) Delete(key interface{}) bool {
	return c.lru.Remove(key)
}

func (c *Cache) filter(v interface{}, ok bool) (interface{}, bool) {
	if v == nil || !ok {
		return nil, false
	}
	if entry, ok := v.(*entry); ok {
		if time.Now().Before(entry.expireTime) {
			if _, ok := entry.value.(noValue); ok {
				return nil, true
			}
			return entry.value, true
		}
	} else {
		if _, ok := v.(noValue); ok {
			return nil, true
		}
		return v, true
	}
	return nil, false
}

func (c *Cache) do(ctx context.Context, key interface{}) (interface{}, error) {
	v, err, _ := c.group.Do(key, func() (interface{}, error) {
		v, expireTime, err := c.load(ctx, key)
		if err != nil {
			return nil, err
		}
		if !c.Set(key, v, expireTime) {
			return nil, ErrNotFound
		}
		if _, ok := v.(noValue); ok {
			return nil, ErrNotFound
		}
		return v, nil
	})
	if err != nil {
		return nil, err
	}
	return v, err
}
