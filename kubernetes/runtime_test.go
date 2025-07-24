package kubernetes

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetObjMetaAccesser(t *testing.T) {
	type testCase struct {
		obj         interface{}
		expectError bool
	}
	testCases := map[string]testCase{
		"Obj is corev1.Pod": {
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
			},
		},
		"Obj is *appv1.Deployment": {
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
			},
		},
		"Invalid obj": {
			obj:         1,
			expectError: true,
		},
		"Obj is nil": {
			obj:         nil,
			expectError: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {

			meta, err := GetObjMetaAccesser(testCase.obj)

			if testCase.expectError {
				assert.NotNil(t, err, Commentf(test.ErrErrorFmt, testName, err))
				return
			}
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName, err))

			assert.NotNil(t, meta, Commentf(test.ErrResultFmt, testName, "meta is nil"))
			assert.True(t, reflect.DeepEqual(meta, testCase.obj), Commentf(test.ErrResultFmt, testName, "meta is not equal to obj"))
		})
	}
}
