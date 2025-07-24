package kubernetes

import (
	"context"
	"testing"

	"github.com/longhorn/go-common-libs/test"
	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateClusterRoleBinding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRoleBinding *rbacv1.ClusterRoleBinding
		IsAlreadyExists    bool
	}
	testCases := map[string]testCase{
		"Existing": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"Already exists": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			IsAlreadyExists: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if testCase.IsAlreadyExists {
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, testCase.clusterRoleBinding, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			clusterRoleBinding, err := CreateClusterRoleBinding(kubeClient, testCase.clusterRoleBinding)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.NotNil(t, clusterRoleBinding, Commentf(test.ErrResultFmt, testName))

			clusterRoleBinding, err = kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, testCase.clusterRoleBinding.Name, metav1.GetOptions{})
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, clusterRoleBinding.Name, testCase.clusterRoleBinding.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestDeleteClusterRoleBinding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRoleBinding *rbacv1.ClusterRoleBinding
		expectNotFound     bool
	}
	testCases := map[string]testCase{
		"Existing": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"Not found": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expectNotFound: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if !testCase.expectNotFound {
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, testCase.clusterRoleBinding, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

				clusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, testCase.clusterRoleBinding.Name, metav1.GetOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
				assert.Equal(t, clusterRoleBinding.Name, testCase.clusterRoleBinding.Name, Commentf(test.ErrResultFmt, testName))
			}

			err := DeleteClusterRoleBinding(kubeClient, testCase.clusterRoleBinding.Name)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

			_, err = kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, testCase.clusterRoleBinding.Name, metav1.GetOptions{})
			assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetClusterRoleBinding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRoleBinding *rbacv1.ClusterRoleBinding
		expectNotFound     bool
	}
	testCases := map[string]testCase{
		"Existing": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"Not found": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expectNotFound: true,
		},
	}
	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if !testCase.expectNotFound {
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, testCase.clusterRoleBinding, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			clusterRoleBinding, err := GetClusterRoleBinding(kubeClient, testCase.clusterRoleBinding.Name)
			if testCase.expectNotFound {
				assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, clusterRoleBinding.Name, testCase.clusterRoleBinding.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}
