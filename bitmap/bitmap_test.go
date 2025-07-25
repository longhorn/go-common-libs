package bitmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitmap(t *testing.T) {
	t.Run("NewBitmap with invalid range", func(t *testing.T) {
		bm, err := NewBitmap(200, 100)
		assert.Nil(t, bm)
		assert.NotNil(t, err)
	})

	t.Run("Basic allocation and release", func(t *testing.T) {
		bm, err := NewBitmap(100, 200)
		assert.NoError(t, err)

		_, _, err = bm.AllocateRange(0)
		assert.Error(t, err)

		start, end, err := bm.AllocateRange(100)
		assert.NoError(t, err)
		assert.Equal(t, int32(100), start)
		assert.Equal(t, int32(199), end)

		start, end, err = bm.AllocateRange(1)
		assert.NoError(t, err)
		assert.Equal(t, int32(200), start)
		assert.Equal(t, int32(200), end)

		_, _, err = bm.AllocateRange(1)
		assert.Error(t, err)

		err = bm.ReleaseRange(100, 100)
		assert.NoError(t, err)

		start, end, err = bm.AllocateRange(1)
		assert.NoError(t, err)
		assert.Equal(t, int32(100), start)
		assert.Equal(t, int32(100), end)

		err = bm.ReleaseRange(105, 120)
		assert.NoError(t, err)

		start, end, err = bm.AllocateRange(15)
		assert.NoError(t, err)
		assert.Equal(t, int32(105), start)
		assert.Equal(t, int32(119), end)

		_, _, err = bm.AllocateRange(2)
		assert.Error(t, err)

		start, end, err = bm.AllocateRange(1)
		assert.NoError(t, err)
		assert.Equal(t, int32(120), start)
		assert.Equal(t, int32(120), end)

		// No-op
		err = bm.ReleaseRange(0, 0)
		assert.NoError(t, err)

		err = bm.ReleaseRange(0, 200)
		assert.Error(t, err)
	})
}

func TestBitmap2(t *testing.T) {
	bm, _ := NewBitmap(100, 200)

	t.Run("Initial allocation", func(t *testing.T) {
		start, end, err := bm.AllocateRange(50)
		assert.NoError(t, err)
		assert.Equal(t, int32(100), start)
		assert.Equal(t, int32(149), end)
	})

	t.Run("Invalid release", func(t *testing.T) {
		err := bm.ReleaseRange(149, 100)
		assert.NotNil(t, err)
	})

	t.Run("Valid release and reuse", func(t *testing.T) {
		err := bm.ReleaseRange(100, 149)
		assert.NoError(t, err)

		// Even though freed, it will start looking after last successful.
		start, end, err := bm.AllocateRange(1)
		assert.NoError(t, err)
		assert.Equal(t, int32(150), start)
		assert.Equal(t, int32(150), end)
	})

	t.Run("Allocate fragmented", func(t *testing.T) {
		// There are 100 free, but not contiguous
		_, _, err := bm.AllocateRange(100)
		assert.NotNil(t, err)

		start, end, err := bm.AllocateRange(40)
		assert.NoError(t, err)
		assert.Equal(t, int32(151), start)
		assert.Equal(t, int32(190), end)

		// This will work, but only on the fallback search.
		start, end, err = bm.AllocateRange(50)
		assert.NoError(t, err)
		assert.Equal(t, int32(100), start)
		assert.Equal(t, int32(149), end)

		_, _, err = bm.AllocateRange(20)
		assert.NotNil(t, err)

		start, end, err = bm.AllocateRange(10)
		assert.NoError(t, err)
		assert.Equal(t, int32(191), start)
		assert.Equal(t, int32(200), end)

		_, _, err = bm.AllocateRange(1)
		assert.NotNil(t, err)
	})

	t.Run("Partial release and reuse", func(t *testing.T) {
		err := bm.ReleaseRange(120, 149)
		assert.NoError(t, err)

		start, end, err := bm.AllocateRange(10)
		assert.NoError(t, err)
		assert.Equal(t, int32(120), start)
		assert.Equal(t, int32(129), end)

		start, end, err = bm.AllocateRange(1)
		assert.NoError(t, err)
		assert.Equal(t, int32(130), start)
		assert.Equal(t, int32(130), end)
	})
}
