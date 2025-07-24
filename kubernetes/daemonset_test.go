package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/longhorn/go-common-libs/test"
)

func TestCreateDaemonSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		daemonSet       *appsv1.DaemonSet
		IsAlreadyExists bool
	}
	testCases := map[string]testCase{
		"Existing": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Already exists": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
			IsAlreadyExists: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if testCase.IsAlreadyExists {
				_, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Create(ctx, testCase.daemonSet, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			daemonSet, err := CreateDaemonSet(kubeClient, testCase.daemonSet)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.NotNil(t, daemonSet, Commentf(test.ErrResultFmt, testName))

			daemonSet, err = kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Get(ctx, testCase.daemonSet.Name, metav1.GetOptions{})
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, daemonSet.Name, testCase.daemonSet.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestDeleteDaemonSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		daemonSet      *appsv1.DaemonSet
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Not found": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
			expectNotFound: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if !testCase.expectNotFound {
				_, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Create(ctx, testCase.daemonSet, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

				daemonSet, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Get(ctx, testCase.daemonSet.Name, metav1.GetOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
				assert.Equal(t, daemonSet.Name, testCase.daemonSet.Name, Commentf(test.ErrResultFmt, testName))
			}

			err := DeleteDaemonSet(kubeClient, testCase.daemonSet.Namespace, testCase.daemonSet.Name)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

			_, err = kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Get(ctx, testCase.daemonSet.Name, metav1.GetOptions{})
			assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetDaemonSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		daemonSet      *appsv1.DaemonSet
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Not found": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
			expectNotFound: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if !testCase.expectNotFound {
				_, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Create(ctx, testCase.daemonSet, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			daemonSet, err := GetDaemonSet(kubeClient, testCase.daemonSet.Namespace, testCase.daemonSet.Name)
			if testCase.expectNotFound {
				assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, daemonSet.Name, testCase.daemonSet.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestIsDaemonSetReady(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		daemonSet     *appsv1.DaemonSet
		expectedReady bool
	}
	testCases := map[string]testCase{
		"Existing": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Status: appsv1.DaemonSetStatus{
					NumberReady:            1,
					DesiredNumberScheduled: 1,
				},
			},
			expectedReady: true,
		},
		"Not ready": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Status: appsv1.DaemonSetStatus{
					NumberReady:            0,
					DesiredNumberScheduled: 1,
				},
			},
			expectedReady: false,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()
			_, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Create(ctx, testCase.daemonSet, metav1.CreateOptions{})
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

			ready := IsDaemonSetReady(testCase.daemonSet)
			assert.Equal(t, ready, testCase.expectedReady, Commentf(test.ErrResultFmt, testName))
		})
	}
}
