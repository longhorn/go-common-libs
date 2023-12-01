package io

import (
	"testing"

	. "gopkg.in/check.v1"

	_ "github.com/longhorn/go-common-libs/test"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})
