package utils

import (
	"strings"

	"golang.org/x/exp/constraints"
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
	// Test cases for base cases
	baseTestCases := map[string]testCaseSortKeys[string, any]{
		"SortKeys(...): nil map": {
			inputMap:    nil,
			expectError: true,
		},
		"SortKeys(...): empty map": {
			inputMap: map[string]any{},
			expected: []string{},
		},
	}

	// Test cases for string keys
	stringTestCases := map[string]testCaseSortKeys[string, any]{
		"SortKeys(...): string": {
			inputMap: map[string]any{
				"b": "",
				"c": "",
				"a": "",
			},
			expected: []string{"a", "b", "c"},
		},
	}

	// Test cases for uint64 keys
	uint64TestCases := map[string]testCaseSortKeys[uint64, any]{
		"SortKeys(...): uint64": {
			inputMap: map[uint64]any{
				2: "",
				1: "",
				3: "",
			},
			expected: []uint64{1, 2, 3},
		},
	}

	runTestSortKeys(c, baseTestCases)
	runTestSortKeys(c, stringTestCases)
	runTestSortKeys(c, uint64TestCases)
}

type testCaseSortKeys[K constraints.Ordered, V any] struct {
	inputMap    map[K]V
	expected    []K
	expectError bool
}

func runTestSortKeys[K constraints.Ordered, V any](c *C, testCases map[string]testCaseSortKeys[K, V]) {
	for testName, tc := range testCases {
		c.Logf("Testing utils.%v", testName)
		result, err := SortKeys(tc.inputMap)

		if tc.expectError {
			c.Assert(err, NotNil, Commentf("Expected error in %v", testName))
			continue
		}
		c.Assert(err, IsNil, Commentf("Unexpected error in %v", testName))
		c.Assert(result, DeepEquals, tc.expected, Commentf("Unexpected result in %v", testName))
	}
}
