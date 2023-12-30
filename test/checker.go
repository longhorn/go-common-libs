package test

import (
	"fmt"
	"reflect"

	"gopkg.in/check.v1"
)

type isInListChecker struct {
	*check.CheckerInfo
}

// isInListChecker verifies if an element is present in the provided list.
// The list must be a slice or array, and the element must be the same type as the list.
//
// For example:
//
//     c.Assert("b", IsInList, []string{"a", "b", "c"})
//     c.Assert(2, IsInList, []int{1, 2, 3})
//

var IsInList check.Checker = &isInListChecker{
	&check.CheckerInfo{
		Name:   "IsInList",
		Params: []string{"element", "list"},
	},
}

func (checker *isInListChecker) Check(params []interface{}, names []string) (result bool, error string) {
	defer func() {
		if r := recover(); r != nil {
			result = false
			error = fmt.Sprint(r)
		}
	}()

	element := params[0]
	list := params[1]

	valueOfList := reflect.ValueOf(list)
	if valueOfList.Kind() != reflect.Slice && valueOfList.Kind() != reflect.Array {
		return false, fmt.Sprintf("list must be a slice or array, got %T", valueOfList.Type())
	}

	if valueOfList.Len() == 0 {
		return false, "list should not be empty"
	}

	for i := 0; i < valueOfList.Len(); i++ {
		if valueOfList.Index(i).Interface() == element {
			return true, ""
		}
	}
	return false, ""
}
