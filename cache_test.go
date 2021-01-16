package sfcache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	_, err := New(0, nil)
	require.EqualError(t, err, "size must be 1 or greater")

	_, err = New(10, nil)
	require.EqualError(t, err, "loader is required")

	seq := 0
	ttl := func() time.Time { return time.Now().Add(50 * time.Millisecond) }

	c, err := New(1000, func(ctx context.Context, key interface{}) (interface{}, time.Time, error) {
		seq++
		return fmt.Sprintf("%s-%d", key, seq), ttl(), nil
	})
	require.NoError(t, err)

	v, found := c.Peek("test")
	require.False(t, found)
	require.Nil(t, v)

	v, err = c.Load(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-1", v)

	v, err = c.Load(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-1", v)

	v, found = c.Peek("foo")
	require.True(t, found)
	require.Equal(t, "foo-1", v)

	v, err = c.Load(context.Background(), "bar")
	require.NoError(t, err)
	require.Equal(t, "bar-2", v)

	v, err = c.Load(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-1", v)

	c.Delete("foo")

	v, found = c.Get("foo")
	require.Nil(t, v)
	require.False(t, found)

	v, err = c.Load(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-3", v)

	time.Sleep(100 * time.Millisecond)

	v, err = c.Load(context.Background(), "foo")
	require.NoError(t, err)
	require.Equal(t, "foo-4", v)

	ttl = func() time.Time { return NoExpireTime }

	v, err = c.Load(context.Background(), "happy")
	require.NoError(t, err)
	require.Equal(t, "happy-5", v)

	time.Sleep(100 * time.Millisecond)

	v, err = c.Load(context.Background(), "happy")
	require.NoError(t, err)
	require.Equal(t, "happy-5", v)
}