package kubernetes

import (
	"context"

	"github.com/longhorn/go-common-libs/test"
	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *TestSuite) TestCreateServiceAccount(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		serviceAccount  *corev1.ServiceAccount
		IsAlreadyExists bool
	}
	testCases := map[string]testCase{
		"CreateServiceAccount(...):": {
			serviceAccount: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"CreateServiceAccount(...): already exists": {
			serviceAccount: &corev1.ServiceAccount{
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
			_, err := kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Create(ctx, testCase.serviceAccount, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		serviceAccount, err := CreateServiceAccount(kubeClient, testCase.serviceAccount)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(serviceAccount, NotNil, Commentf(test.ErrResultFmt, testName))

		serviceAccount, err = kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Get(ctx, testCase.serviceAccount.Name, metav1.GetOptions{})
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(serviceAccount.Name, Equals, testCase.serviceAccount.Name, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestDeleteServiceAccount(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		serviceAccount *corev1.ServiceAccount
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"DeleteServiceAccount(...):": {
			serviceAccount: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"DeleteServiceAccount(...): not found": {
			serviceAccount: &corev1.ServiceAccount{
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
			_, err := kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Create(ctx, testCase.serviceAccount, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

			serviceAccount, err := kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Get(ctx, testCase.serviceAccount.Name, metav1.GetOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
			c.Assert(serviceAccount.Name, Equals, testCase.serviceAccount.Name, Commentf(test.ErrResultFmt, testName))
		}

		err := DeleteServiceAccount(kubeClient, testCase.serviceAccount.Namespace, testCase.serviceAccount.Name)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

		_, err = kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Get(ctx, testCase.serviceAccount.Name, metav1.GetOptions{})
		c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetServiceAccount(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		serviceAccount *corev1.ServiceAccount
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"GetServiceAccount(...):": {
			serviceAccount: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"GetServiceAccount(...): not found": {
			serviceAccount: &corev1.ServiceAccount{
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
			_, err := kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Create(ctx, testCase.serviceAccount, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		serviceAccount, err := GetServiceAccount(kubeClient, testCase.serviceAccount.Namespace, testCase.serviceAccount.Name)
		if testCase.expectNotFound {
			c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
			return
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(serviceAccount.Name, Equals, testCase.serviceAccount.Name, Commentf(test.ErrResultFmt, testName))
	}
}
