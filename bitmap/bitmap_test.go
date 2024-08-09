package bitmap

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestBitmap(c *C) {
	bm, err := NewBitmap(200, 100)
	c.Assert(bm, IsNil)
	c.Assert(err, NotNil)

	bm, err = NewBitmap(100, 200)
	c.Assert(err, IsNil)

	_, _, err = bm.AllocateRange(0)
	c.Assert(err, NotNil)

	start, end, err := bm.AllocateRange(100)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(100))
	c.Assert(end, Equals, int32(199))

	start, end, err = bm.AllocateRange(1)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(200))
	c.Assert(end, Equals, int32(200))

	_, _, err = bm.AllocateRange(1)
	c.Assert(err, NotNil)

	err = bm.ReleaseRange(100, 100)
	c.Assert(err, IsNil)

	start, end, err = bm.AllocateRange(1)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(100))
	c.Assert(end, Equals, int32(100))

	err = bm.ReleaseRange(105, 120)
	c.Assert(err, IsNil)

	start, end, err = bm.AllocateRange(15)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(105))
	c.Assert(end, Equals, int32(119))

	_, _, err = bm.AllocateRange(2)
	c.Assert(err, NotNil)

	start, end, err = bm.AllocateRange(1)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(120))
	c.Assert(end, Equals, int32(120))

	// No-op
	err = bm.ReleaseRange(0, 0)
	c.Assert(err, IsNil)

	err = bm.ReleaseRange(0, 200)
	c.Assert(err, NotNil)
}

func (s *TestSuite) TestBitmap2(c *C) {
	bm, _ := NewBitmap(100, 200)

	start, end, err := bm.AllocateRange(50)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(100))
	c.Assert(end, Equals, int32(149))

	err = bm.ReleaseRange(149, 100)
	c.Assert(err, NotNil)

	err = bm.ReleaseRange(100, 149)
	c.Assert(err, IsNil)

	// Even though freed, it will start looking after last successful.
	start, end, err = bm.AllocateRange(1)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(150))
	c.Assert(end, Equals, int32(150))

	// There are 100 free, but not contiguous
	_, _, err = bm.AllocateRange(100)
	c.Assert(err, NotNil)

	start, end, err = bm.AllocateRange(40)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(151))
	c.Assert(end, Equals, int32(190))

	// This will work, but only on the fallback search.
	start, end, err = bm.AllocateRange(50)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(100))
	c.Assert(end, Equals, int32(149))

	_, _, err = bm.AllocateRange(20)
	c.Assert(err, NotNil)

	start, end, err = bm.AllocateRange(10)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(191))
	c.Assert(end, Equals, int32(200))

	_, _, err = bm.AllocateRange(1)
	c.Assert(err, NotNil)

	err = bm.ReleaseRange(120, 149)
	c.Assert(err, IsNil)

	start, end, err = bm.AllocateRange(10)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(120))
	c.Assert(end, Equals, int32(129))

	start, end, err = bm.AllocateRange(1)
	c.Assert(err, IsNil)
	c.Assert(start, Equals, int32(130))
	c.Assert(end, Equals, int32(130))
}
