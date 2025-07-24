package kubernetes

import (
	"context"
	"testing"

	"github.com/longhorn/go-common-libs/test"
	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateServiceAccount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		serviceAccount  *corev1.ServiceAccount
		IsAlreadyExists bool
	}
	testCases := map[string]testCase{
		"Existing": {
			serviceAccount: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Already exists": {
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
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if testCase.IsAlreadyExists {
				_, err := kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Create(ctx, testCase.serviceAccount, metav1.CreateOptions{})
				assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			serviceAccount, err := CreateServiceAccount(kubeClient, testCase.serviceAccount)
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.NotNil(t, serviceAccount, Commentf(test.ErrResultFmt, testName))

			serviceAccount, err = kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Get(ctx, testCase.serviceAccount.Name, metav1.GetOptions{})
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, serviceAccount.Name, testCase.serviceAccount.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestDeleteServiceAccount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		serviceAccount *corev1.ServiceAccount
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			serviceAccount: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Not found": {
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
		t.Run(testName, func(t *testing.T) {

			kubeClient := fake.NewSimpleClientset()
			if !testCase.expectNotFound {
				_, err := kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Create(ctx, testCase.serviceAccount, metav1.CreateOptions{})
				assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))

				serviceAccount, err := kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Get(ctx, testCase.serviceAccount.Name, metav1.GetOptions{})
				assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))
				assert.Equal(t, serviceAccount.Name, testCase.serviceAccount.Name, Commentf(test.ErrResultFmt, testName))
			}

			err := DeleteServiceAccount(kubeClient, testCase.serviceAccount.Namespace, testCase.serviceAccount.Name)
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))

			_, err = kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Get(ctx, testCase.serviceAccount.Name, metav1.GetOptions{})
			assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetServiceAccount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		serviceAccount *corev1.ServiceAccount
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			serviceAccount: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Not found": {
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
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if !testCase.expectNotFound {
				_, err := kubeClient.CoreV1().ServiceAccounts(testCase.serviceAccount.Namespace).Create(ctx, testCase.serviceAccount, metav1.CreateOptions{})
				assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			serviceAccount, err := GetServiceAccount(kubeClient, testCase.serviceAccount.Namespace, testCase.serviceAccount.Name)
			if testCase.expectNotFound {
				assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
				return
			}
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, serviceAccount.Name, testCase.serviceAccount.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}
