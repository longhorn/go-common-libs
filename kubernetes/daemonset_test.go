package kubernetes

import (
	"context"

	"github.com/longhorn/go-common-libs/test"
	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *TestSuite) TestCreateDaemonSet(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		daemonSet       *appsv1.DaemonSet
		IsAlreadyExists bool
	}
	testCases := map[string]testCase{
		"CreateDaemonSet(...):": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"CreateDaemonSet(...): already exists": {
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
		c.Logf("testing kubernetes.%v", testName)

		kubeClient := fake.NewSimpleClientset()

		if testCase.IsAlreadyExists {
			_, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Create(ctx, testCase.daemonSet, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		daemonSet, err := CreateDaemonSet(kubeClient, testCase.daemonSet)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(daemonSet, NotNil, Commentf(test.ErrResultFmt, testName))

		daemonSet, err = kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Get(ctx, testCase.daemonSet.Name, metav1.GetOptions{})
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(daemonSet.Name, Equals, testCase.daemonSet.Name, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestDeleteDaemonSet(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		daemonSet      *appsv1.DaemonSet
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"DeleteDaemonSet(...):": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"DeleteDaemonSet(...): not found": {
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
		c.Logf("testing kubernetes.%v", testName)

		kubeClient := fake.NewSimpleClientset()

		if !testCase.expectNotFound {
			_, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Create(ctx, testCase.daemonSet, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

			daemonSet, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Get(ctx, testCase.daemonSet.Name, metav1.GetOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
			c.Assert(daemonSet.Name, Equals, testCase.daemonSet.Name, Commentf(test.ErrResultFmt, testName))
		}

		err := DeleteDaemonSet(kubeClient, testCase.daemonSet.Namespace, testCase.daemonSet.Name)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

		_, err = kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Get(ctx, testCase.daemonSet.Name, metav1.GetOptions{})
		c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetDaemonSet(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		daemonSet      *appsv1.DaemonSet
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"GetDaemonSet(...):": {
			daemonSet: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"GetDaemonSet(...): not found": {
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
		c.Logf("testing kubernetes.%v", testName)

		kubeClient := fake.NewSimpleClientset()

		if !testCase.expectNotFound {
			_, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Create(ctx, testCase.daemonSet, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		daemonSet, err := GetDaemonSet(kubeClient, testCase.daemonSet.Namespace, testCase.daemonSet.Name)
		if testCase.expectNotFound {
			c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
			return
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(daemonSet.Name, Equals, testCase.daemonSet.Name, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestIsDaemonSetReady(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		daemonSet     *appsv1.DaemonSet
		expectedReady bool
	}
	testCases := map[string]testCase{
		"IsDaemonSetReady(...):": {
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
		"IsDaemonSetReady(...): not ready": {
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
		c.Logf("testing kubernetes.%v", testName)

		kubeClient := fake.NewSimpleClientset()
		_, err := kubeClient.AppsV1().DaemonSets(testCase.daemonSet.Namespace).Create(ctx, testCase.daemonSet, metav1.CreateOptions{})
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

		ready := IsDaemonSetReady(testCase.daemonSet)
		c.Assert(ready, Equals, testCase.expectedReady, Commentf(test.ErrResultFmt, testName))
	}
}
