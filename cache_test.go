package sfcache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew_Config(t *testing.T) {
	check := func(c *Config) error {
		_, err := New(c)
		return err
	}
	load := func(ctx context.Context, key interface{}) (interface{}, error) {
		return nil, nil
	}

	require.EqualError(t, check(nil), "config required")
	require.EqualError(t, check(&Config{}), "config.Load required")
	require.NoError(t, check(&Config{Load: load}))
	require.EqualError(t, check(&Config{
		Load: load,
		Capacity: -1,
	}), "config.Capacity must be positive")
	require.EqualError(t, check(&Config{
		Load: load,
		MaxAge: -1,
	}), "config.MaxAge must be positive")
}

func TestNew_NoMaxAge(t *testing.T) {
	var seq int
	c, err := New(&Config{
		Load: func(ctx context.Context, key interface{}) (interface{}, error) {
			seq++
			return fmt.Sprintf("%s-%d", key, seq), nil
		},
	})
	require.NoError(t, err)

	v, found := c.Peek("test")
	require.False(t, found)
	require.Nil(t, v)

	v, err = c.Get(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-1", v)

	v, err = c.Get(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-1", v)

	v, found = c.Peek("foo")
	require.True(t, found)
	require.Equal(t, "foo-1", v)

	v, err = c.Get(context.Background(), "bar")
	require.NoError(t, err)
	require.Equal(t, "bar-2", v)

	v, err = c.Get(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-1", v)

	c.Remove("foo")

	v, err = c.Get(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-3", v)
}

func TestNew_MaxAge(t *testing.T) {
	var seq int
	c, err := New(&Config{
		Load: func(ctx context.Context, key interface{}) (interface{}, error) {
			seq++
			return fmt.Sprintf("%s-%d", key, seq), nil
		},
		MaxAge: 50 * time.Millisecond,
	})
	require.NoError(t, err)

	v, err := c.Get(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-1", v)

	v, err = c.Get(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-1", v)

	time.Sleep(100 * time.Millisecond)

	v, err = c.Get(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-2", v)
}