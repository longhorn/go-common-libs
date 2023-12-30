package utils

import (
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func (s *TestSuite) TestContains(c *C) {
	type testCase struct {
		inputSlice []interface{}
		inputValue interface{}

		expected bool
	}
	testCases := map[string]testCase{
		"Contains(...)": {
			inputSlice: []interface{}{"a", "b", "c"},
			inputValue: "b",
			expected:   true,
		},
		"Contains(...): not in slice": {
			inputSlice: []interface{}{"a", "b", "c"},
			inputValue: "d",
			expected:   false,
		},
		"Contains(...): empty slice": {
			inputSlice: []interface{}{},
			inputValue: "a",
			expected:   false,
		},
		"Contains(...): nil slice": {
			inputSlice: nil,
			inputValue: "a",
			expected:   false,
		},
		"Contains(...): nil value": {
			inputSlice: []interface{}{"a", "b", "c"},
			inputValue: nil,
			expected:   false,
		},
		"Contains(...): integer slice": {
			inputSlice: []interface{}{1, 2, 3},
			inputValue: 2,
			expected:   true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := Contains(testCase.inputSlice, testCase.inputValue)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetFunctionName(c *C) {
	type testCase struct {
		inputFunction interface{}

		expected string
	}
	testCases := map[string]testCase{
		"GetFunctionName(...)": {
			inputFunction: GetFunctionName,
			expected:      "utils.GetFunctionName",
		},
		"GetFunctionName(...): not a function": {
			inputFunction: "not a function",
			expected:      "",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := GetFunctionName(testCase.inputFunction)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetFunctionPath(c *C) {
	type testCase struct {
		inputFunction interface{}

		expected string
	}
	testCases := map[string]testCase{
		"GetFunctionPath(...)": {
			inputFunction: GetFunctionName,
			expected:      "github.com/longhorn/go-common-libs/utils.GetFunctionName",
		},
		"GetFunctionPath(...): not a function": {
			inputFunction: "not a function",
			expected:      "",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := GetFunctionPath(testCase.inputFunction)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestIsStringInSlice(c *C) {
	type testCase struct {
		inputList []string
		inputItem string

		expected bool
	}
	testCases := map[string]testCase{
		"IsStringInSlice(...)": {
			inputList: []string{"a", "b", "c"},
			inputItem: "b",
			expected:  true,
		},
		"IsStringInSlice(...): not in slice": {
			inputList: []string{"a", "b", "c"},
			inputItem: "d",
			expected:  false,
		},
		"IsStringInSlice(...): empty slice": {
			inputList: []string{},
			inputItem: "a",
			expected:  false,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := IsStringInSlice(testCase.inputList, testCase.inputItem)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestRandomID(c *C) {
	type testCase struct {
		idLength int

		expectedLength int
	}
	testCases := map[string]testCase{
		"RandomID(...)": {
			idLength:       10,
			expectedLength: 10,
		},
		"RandomID(...): default length": {
			idLength:       0,
			expectedLength: 8,
		},
		"RandomID(...): negative length": {
			idLength:       -1,
			expectedLength: 8,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := RandomID(testCase.idLength)
		c.Assert(len(result), Equals, testCase.expectedLength, Commentf(test.ErrResultFmt, testName))
	}
}

func (s TestSuite) TestGenerateRandomNumber(c *C) {
	type testCase struct {
		inputLower int64
		inputUpper int64

		expectSuccess bool
		retRange      []int64
	}
	testCases := map[string]testCase{
		"GenerateRandomNumber(...)": {
			inputLower:    0,
			inputUpper:    10,
			expectSuccess: true,
			retRange:      []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		"GenerateRandomNumber(...): min > max": {
			inputLower:    10,
			inputUpper:    0,
			expectSuccess: false,
		},
		"GenerateRandomNumber(...): min == max": {
			inputLower:    10,
			inputUpper:    10,
			expectSuccess: true,
			retRange:      []int64{10},
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result, err := GenerateRandomNumber(testCase.inputLower, testCase.inputUpper)
		if testCase.expectSuccess {
			c.Assert(err, IsNil, Commentf(test.ErrResultFmt, testName))
			c.Assert(result, test.IsInList, testCase.retRange, Commentf(test.ErrResultFmt, testName))
		} else {
			c.Assert(err, NotNil, Commentf(test.ErrResultFmt, testName))
		}
	}
}
