package kubernetes

import (
	"reflect"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *TestSuite) TestGetObjMetaAccesser(c *C) {
	type testCase struct {
		obj         interface{}
		expectError bool
	}
	testCases := map[string]testCase{
		"GetObjMetaAccesser(...): obj is corev1.Pod": {
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
			},
		},
		"GetObjMetaAccesser(...): obj is *appv1.Deployment": {
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
			},
		},
		"GetObjMetaAccesser(...): invalid obj": {
			obj:         1,
			expectError: true,
		},
		"GetObjMetaAccesser(...): obj is nil": {
			obj:         nil,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing utils.%v", testName)

		meta, err := GetObjMetaAccesser(testCase.obj)

		if testCase.expectError {
			c.Assert(err, NotNil, Commentf(test.ErrErrorFmt, testName, err))
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName, err))

		c.Assert(meta, NotNil, Commentf(test.ErrResultFmt, testName, "meta is nil"))
		c.Assert(reflect.DeepEqual(meta, testCase.obj), Equals, true, Commentf(test.ErrResultFmt, testName, "meta is not equal to obj"))
	}
}
