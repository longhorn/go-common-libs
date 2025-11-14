package utils

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/constraints"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/test/fake"
)

func TestContains(t *testing.T) {
	type testCase struct {
		inputSlice []interface{}
		inputValue interface{}

		expected bool
	}
	testCases := map[string]testCase{
		"Valid slice": {
			inputSlice: []interface{}{"a", "b", "c"},
			inputValue: "b",
			expected:   true,
		},
		"Not in slice": {
			inputSlice: []interface{}{"a", "b", "c"},
			inputValue: "d",
			expected:   false,
		},
		"Empty slice": {
			inputSlice: []interface{}{},
			inputValue: "a",
			expected:   false,
		},
		"Nil slice": {
			inputSlice: nil,
			inputValue: "a",
			expected:   false,
		},
		"Nil value": {
			inputSlice: []interface{}{"a", "b", "c"},
			inputValue: nil,
			expected:   false,
		},
		"Integer slice": {
			inputSlice: []interface{}{1, 2, 3},
			inputValue: 2,
			expected:   true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := Contains(testCase.inputSlice, testCase.inputValue)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetFunctionName(t *testing.T) {
	type testCase struct {
		inputFunction interface{}

		expected string
	}
	testCases := map[string]testCase{
		"Valid function": {
			inputFunction: GetFunctionName,
			expected:      "utils.GetFunctionName",
		},
		"Not a function": {
			inputFunction: "not a function",
			expected:      "",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := GetFunctionName(testCase.inputFunction)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetFunctionPath(t *testing.T) {
	type testCase struct {
		inputFunction interface{}

		expected string
	}
	testCases := map[string]testCase{
		"Valid function": {
			inputFunction: GetFunctionName,
			expected:      "github.com/longhorn/go-common-libs/utils.GetFunctionName",
		},
		"Not a function": {
			inputFunction: "not a function",
			expected:      "",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := GetFunctionPath(testCase.inputFunction)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestIsStringInSlice(t *testing.T) {
	type testCase struct {
		inputList []string
		inputItem string

		expected bool
	}
	testCases := map[string]testCase{
		"In slice": {
			inputList: []string{"a", "b", "c"},
			inputItem: "b",
			expected:  true,
		},
		"Not in slice": {
			inputList: []string{"a", "b", "c"},
			inputItem: "d",
			expected:  false,
		},
		"Empty slice": {
			inputList: []string{},
			inputItem: "a",
			expected:  false,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := IsStringInSlice(testCase.inputList, testCase.inputItem)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestRandomID(t *testing.T) {
	type testCase struct {
		idLength int

		expectedLength int
	}
	testCases := map[string]testCase{
		"Positive length": {
			idLength:       10,
			expectedLength: 10,
		},
		"Default length": {
			idLength:       0,
			expectedLength: 8,
		},
		"Negative length": {
			idLength:       -1,
			expectedLength: 8,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := RandomID(testCase.idLength)
			assert.Equal(t, testCase.expectedLength, len(result), Commentf(test.ErrResultFmt, testName))
			assert.False(t, strings.Contains(result, "-"), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGenerateRandomNumber(t *testing.T) {
	type testCase struct {
		inputLower int64
		inputUpper int64

		expectSuccess bool
		retRange      []int64
	}
	testCases := map[string]testCase{
		"Min < max": {
			inputLower:    0,
			inputUpper:    10,
			expectSuccess: true,
			retRange:      []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		"Min > max": {
			inputLower:    10,
			inputUpper:    0,
			expectSuccess: false,
		},
		"Min == max": {
			inputLower:    10,
			inputUpper:    10,
			expectSuccess: true,
			retRange:      []int64{10},
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := GenerateRandomNumber(testCase.inputLower, testCase.inputUpper)
			if testCase.expectSuccess {
				assert.NoError(t, err, Commentf(test.ErrResultFmt, testName))
				assert.Contains(t, testCase.retRange, result, Commentf(test.ErrResultFmt, testName))
			} else {
				assert.Error(t, err, Commentf(test.ErrResultFmt, testName))
			}
		})
	}
}

func TestConvertTypeToString(t *testing.T) {
	type testCase struct {
		inputValue interface{}

		expected string
	}
	testCases := map[string]testCase{
		"String": {
			inputValue: "abc",
			expected:   "abc",
		},
		"Int": {
			inputValue: 123,
			expected:   "123",
		},
		"Int64": {
			inputValue: int64(123),
			expected:   "123",
		},
		"Float": {
			inputValue: 123.456,
			expected:   "123.456",
		},
		"Bool": {
			inputValue: true,
			expected:   "true",
		},
		"Unsupported": {
			inputValue: nil,
			expected:   "Unsupported type: invalid",
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := ConvertTypeToString(testCase.inputValue)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestSortKeys(t *testing.T) {
	// Test cases for base cases
	baseTestCases := map[string]testCaseSortKeys[string, any]{
		"Nil map": {
			inputMap:    nil,
			expectError: true,
		},
		"Empty map": {
			inputMap: map[string]any{},
			expected: []string{},
		},
	}

	// Test cases for string keys
	stringTestCases := map[string]testCaseSortKeys[string, any]{
		"String": {
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
		"Uint64": {
			inputMap: map[uint64]any{
				2: "",
				1: "",
				3: "",
			},
			expected: []uint64{1, 2, 3},
		},
	}

	runTestSortKeys(t, baseTestCases)
	runTestSortKeys(t, stringTestCases)
	runTestSortKeys(t, uint64TestCases)
}

type testCaseSortKeys[K constraints.Ordered, V any] struct {
	inputMap    map[K]V
	expected    []K
	expectError bool
}

func runTestSortKeys[K constraints.Ordered, V any](t *testing.T, testCases map[string]testCaseSortKeys[K, V]) {
	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := SortKeys(tc.inputMap)

			if tc.expectError {
				assert.Error(t, err, Commentf("Expected error in %v", testName))
				return
			}
			assert.NoError(t, err, Commentf("Unexpected error in %v", testName))
			assert.Equal(t, tc.expected, result, Commentf("Unexpected result in %v", testName))
		})
	}
}

func TestGetNumberFromMap(t *testing.T) {
	t.Run("Uint64 type", func(t *testing.T) {
		testCases := map[string]struct {
			inputMap map[string]any
			key      string
			expected uint64
		}{
			"Nil map": {
				inputMap: nil,
				key:      "key",
				expected: 0,
			},
			"Empty map": {
				inputMap: map[string]any{},
				key:      "key",
				expected: 0,
			},
			"Valid uint64 value": {
				inputMap: map[string]any{"count": uint64(100)},
				key:      "count",
				expected: 100,
			},
			"Float64 to uint64 conversion": {
				inputMap: map[string]any{"count": float64(100)},
				key:      "count",
				expected: 100,
			},
			"Float64 with decimal to uint64 (truncates)": {
				inputMap: map[string]any{"count": float64(100.99)},
				key:      "count",
				expected: 100,
			},
			"Key not found": {
				inputMap: map[string]any{"count": uint64(100)},
				key:      "missing",
				expected: 0,
			},
			"Wrong type (string instead of uint64)": {
				inputMap: map[string]any{"count": "100"},
				key:      "count",
				expected: 0,
			},
			"Zero value": {
				inputMap: map[string]any{"count": uint64(0)},
				key:      "count",
				expected: 0,
			},
			"Nil value": {
				inputMap: map[string]any{"count": nil},
				key:      "count",
				expected: 0,
			},
			"Large uint64 value": {
				inputMap: map[string]any{"count": uint64(18446744073709551615)},
				key:      "count",
				expected: 18446744073709551615,
			},
			"Negative float64 value": {
				inputMap: map[string]any{"count": float64(-1)},
				key:      "count",
				expected: 0,
			},
			"Float64 value exceeding max uint64": {
				inputMap: map[string]any{"count": math.MaxFloat64},
				key:      "count",
				expected: 0,
			},
		}

		for testName, tc := range testCases {
			t.Run(testName, func(t *testing.T) {
				result := GetNumberFromMap[uint64](tc.inputMap, tc.key)
				assert.Equal(t, tc.expected, result, Commentf(test.ErrResultFmt, testName))
			})
		}
	})

	t.Run("Uint16 type", func(t *testing.T) {
		testCases := map[string]struct {
			inputMap map[string]any
			key      string
			expected uint16
		}{
			"Valid uint16 value": {
				inputMap: map[string]any{"temp": uint16(25)},
				key:      "temp",
				expected: 25,
			},
			"Float64 to uint16 conversion": {
				inputMap: map[string]any{"temp": float64(25)},
				key:      "temp",
				expected: 25,
			},
			"Float64 with decimal to uint16 (truncates)": {
				inputMap: map[string]any{"temp": float64(10.5)},
				key:      "temp",
				expected: 10,
			},
			"Key not found": {
				inputMap: map[string]any{"temp": uint16(25)},
				key:      "missing",
				expected: 0,
			},
			"Wrong type": {
				inputMap: map[string]any{"temp": "25"},
				key:      "temp",
				expected: 0,
			},
			"Nil value": {
				inputMap: map[string]any{"temp": nil},
				key:      "temp",
				expected: 0,
			},
			"Zero value": {
				inputMap: map[string]any{"temp": uint16(0)},
				key:      "temp",
				expected: 0,
			},
			"Max uint16 value": {
				inputMap: map[string]any{"temp": uint16(65535)},
				key:      "temp",
				expected: 65535,
			},
			"Negative float64 value": {
				inputMap: map[string]any{"temp": float64(-1)},
				key:      "temp",
				expected: 0,
			},
			"Float64 value exceeding max uint16": {
				inputMap: map[string]any{"temp": float64(70000)},
				key:      "temp",
				expected: 0,
			},
		}

		for testName, tc := range testCases {
			t.Run(testName, func(t *testing.T) {
				result := GetNumberFromMap[uint16](tc.inputMap, tc.key)
				assert.Equal(t, tc.expected, result, Commentf(test.ErrResultFmt, testName))
			})
		}
	})

	t.Run("Uint8 type", func(t *testing.T) {
		testCases := map[string]struct {
			inputMap map[string]any
			key      string
			expected uint8
		}{
			"Valid uint8 value": {
				inputMap: map[string]any{"key": uint8(1)},
				key:      "key",
				expected: 1,
			},
			"Float64 to uint8 conversion": {
				inputMap: map[string]any{"key": float64(1)},
				key:      "key",
				expected: 1,
			},
			"Float64 with decimal to uint8 (truncates)": {
				inputMap: map[string]any{"key": float64(5.7)},
				key:      "key",
				expected: 5,
			},
			"Key not found": {
				inputMap: map[string]any{"key": uint8(1)},
				key:      "missing",
				expected: 0,
			},
			"Wrong type": {
				inputMap: map[string]any{"key": "1"},
				key:      "key",
				expected: 0,
			},
			"Zero value": {
				inputMap: map[string]any{"key": uint8(0)},
				key:      "key",
				expected: 0,
			},
			"Nil value": {
				inputMap: map[string]any{"key": nil},
				key:      "key",
				expected: 0,
			},
			"Max uint8 value": {
				inputMap: map[string]any{"key": uint8(255)},
				key:      "key",
				expected: 255,
			},
			"Negative float64 value": {
				inputMap: map[string]any{"key": float64(-1)},
				key:      "key",
				expected: 0,
			},
			"Float64 value exceeding max uint8": {
				inputMap: map[string]any{"key": float64(300)},
				key:      "key",
				expected: 0,
			},
		}

		for testName, tc := range testCases {
			t.Run(testName, func(t *testing.T) {
				result := GetNumberFromMap[uint8](tc.inputMap, tc.key)
				assert.Equal(t, tc.expected, result, Commentf(test.ErrResultFmt, testName))
			})
		}
	})
}

func TestGetStringFromMap(t *testing.T) {
	testCases := map[string]struct {
		inputMap map[string]any
		key      string
		expected string
	}{
		"Valid string value": {
			inputMap: map[string]any{"name": "test"},
			key:      "name",
			expected: "test",
		},
		"Key not found": {
			inputMap: map[string]any{"name": "test"},
			key:      "missing",
			expected: "",
		},
		"Nil map": {
			inputMap: nil,
			key:      "name",
			expected: "",
		},
		"Empty map": {
			inputMap: map[string]any{},
			key:      "name",
			expected: "",
		},
		"Empty string value": {
			inputMap: map[string]any{"name": ""},
			key:      "name",
			expected: "",
		},
		"Nil value": {
			inputMap: map[string]any{"name": nil},
			key:      "name",
			expected: "",
		},
		"Int converted to string": {
			inputMap: map[string]any{"count": 123},
			key:      "count",
			expected: "123",
		},
		"Float64 converted to string": {
			inputMap: map[string]any{"price": 99.99},
			key:      "price",
			expected: "99.99",
		},
		"Bool converted to string": {
			inputMap: map[string]any{"enabled": true},
			key:      "enabled",
			expected: "true",
		},
		"Negative number converted to string": {
			inputMap: map[string]any{"temp": -10},
			key:      "temp",
			expected: "-10",
		},
		"Zero int converted to string": {
			inputMap: map[string]any{"count": 0},
			key:      "count",
			expected: "0",
		},
		"False bool converted to string": {
			inputMap: map[string]any{"enabled": false},
			key:      "enabled",
			expected: "false",
		},
		"String with special characters": {
			inputMap: map[string]any{"path": "/dev/sda1"},
			key:      "path",
			expected: "/dev/sda1",
		},
		"String with spaces": {
			inputMap: map[string]any{"message": "hello world"},
			key:      "message",
			expected: "hello world",
		},
		"fmt.Stringer value": {
			inputMap: map[string]any{
				"obj": fake.Stringer{V: "alpha"},
			},
			key:      "obj",
			expected: "S:alpha",
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := GetStringFromMap(tc.inputMap, tc.key)
			assert.Equal(t, tc.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}
