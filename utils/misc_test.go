package utils

import (
	"strings"

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
		c.Assert(strings.Contains(result, "-"), Equals, false, Commentf(test.ErrResultFmt, testName))
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

func (s *TestSuite) TestConvertTypeToString(c *C) {
	type testCase struct {
		inputValue interface{}

		expected string
	}
	testCases := map[string]testCase{
		"ConvertTypeToString(...): string": {
			inputValue: "abc",
			expected:   "abc",
		},
		"ConvertTypeToString(...): int": {
			inputValue: 123,
			expected:   "123",
		},
		"ConvertTypeToString(...): int64": {
			inputValue: int64(123),
			expected:   "123",
		},
		"ConvertTypeToString(...): float": {
			inputValue: 123.456,
			expected:   "123.456",
		},
		"ConvertTypeToString(...): bool": {
			inputValue: true,
			expected:   "true",
		},
		"ConvertTypeToString(...): unsupported": {
			inputValue: nil,
			expected:   "Unsupported type: invalid",
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result := ConvertTypeToString(testCase.inputValue)
		c.Assert(result, Equals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestSortKeys(c *C) {
	type testCase struct {
		inputMap map[any]any

		expected    []any
		expectError bool
	}
	testCases := map[string]testCase{
		"SortKeys(...): string": {
			inputMap: map[any]any{
				"b": "",
				"c": "",
				"a": "",
			},
			expected: []any{"a", "b", "c"},
		},
		"SortKeys(...): uint64": {
			inputMap: map[any]any{
				uint64(2): "",
				uint64(1): "",
				uint64(3): "",
			},
			expected: []any{uint64(1), uint64(2), uint64(3)},
		},
		"SortKeys(...): empty map": {
			inputMap: map[any]any{},
			expected: []any{},
		},
		"SortKeys(...): nil map": {
			inputMap:    nil,
			expectError: true,
		},
		"SortKeys(...): unsupported key type": {
			inputMap: map[any]any{
				complex64(0): "",
				complex64(1): "",
				complex64(2): "",
			},
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		result, err := SortKeys(testCase.inputMap)
		if testCase.expectError {
			c.Assert(err, NotNil, Commentf(test.ErrResultFmt, testName))
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrResultFmt, testName))

		c.Assert(result, DeepEquals, testCase.expected, Commentf(test.ErrResultFmt, testName))
	}
}
