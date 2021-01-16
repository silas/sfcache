package sfcache

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/golang-lru"
)

var (
	// ErrNotFound is returned when Config.Load returns a nil value.
	ErrNotFound = errors.New("cache entry not found")
)

// Cache is an LRU cache with the ability to populate the value if not found.
type Cache struct {
	maxAge time.Duration

	lru   *lru.Cache
	group *Group
	load  func(ctx context.Context, key interface{}) (interface{}, error)
}

type entry struct {
	time  time.Time
	value interface{}
}

// Config is the settings for the LRU.
type Config struct {
	// Load is the function to get the data to populate the value.
	Load func(ctx context.Context, key interface{}) (interface{}, error)

	// Capacity is the maximum size of the cache.
	Capacity int

	// MaxAge is the maximum age of a value.
	MaxAge time.Duration
}

// New creates an LRU with the given Config.
func New(config *Config) (*Cache, error) {
	if config == nil {
		return nil, errors.New("config required")
	}
	if config.Load == nil {
		return nil, errors.New("config.Load required")
	}
	if config.Capacity == 0 {
		config.Capacity = 1000
	} else if config.Capacity < 1 {
		return nil, errors.New("config.Capacity must be positive")
	}
	if config.MaxAge < 0 {
		return nil, errors.New("config.MaxAge must be positive")
	}

	l, err := lru.New(config.Capacity)
	if err != nil {
		return nil, err
	}

	cache := &Cache{
		maxAge: config.MaxAge,
		lru:    l,
		group:  &Group{},
		load:   config.Load,
	}

	return cache, nil
}

// Get looks up a key's value from the cache or populates it if not found.
func (c *Cache) Get(ctx context.Context, key interface{}) (interface{}, error) {
	if v, ok := c.unexpired(c.lru.Get(key)); ok {
		return v, nil
	}
	return c.do(ctx, key)
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *Cache) Peek(key interface{}) (interface{}, bool) {
	return c.unexpired(c.lru.Peek(key))
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key interface{}) bool {
	return c.lru.Remove(key)
}

func (c *Cache) unexpired(v interface{}, ok bool) (interface{}, bool) {
	if ok && v != nil {
		if c.maxAge == 0 {
			return v, ok
		}
		if entry := v.(*entry); time.Since(entry.time) < c.maxAge {
			return entry.value, ok
		}
	}
	return nil, false
}

func (c *Cache) do(ctx context.Context, key interface{}) (interface{}, error) {
	v, err, _ := c.group.Do(key, func() (interface{}, error) {
		v, err := c.load(ctx, key)
		if err != nil || v == nil {
			return nil, ErrNotFound
		}
		if c.maxAge == 0 {
			c.lru.Add(key, v)
		} else {
			c.lru.Add(key, &entry{
				time: time.Now(),
				value: v,
			})
		}
		return v, nil
	})
	if err != nil {
		return nil, err
	}
	return v, err
}
