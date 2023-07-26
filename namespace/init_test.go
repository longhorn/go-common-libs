package namespace

import (
	"testing"

	"github.com/sirupsen/logrus"

	. "gopkg.in/check.v1"
)

const (
	TestErrErrorFmt  = "Unexpected error for test case: %s: %v"
	TestErrResultFmt = "Unexpected result for test case: %s"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct {
}

var _ = Suite(&TestSuite{})

func (s *TestSuite) SetUpTest(c *C) {
	logrus.SetLevel(logrus.TraceLevel)
}
