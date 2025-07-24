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

func TestCreateClusterRole(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRole     *rbacv1.ClusterRole
		IsAlreadyExists bool
	}
	testCases := map[string]testCase{
		"Not existing": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"Already exists": {
			clusterRole: &rbacv1.ClusterRole{
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
				_, err := kubeClient.RbacV1().ClusterRoles().Create(ctx, testCase.clusterRole, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			clusterRole, err := CreateClusterRole(kubeClient, testCase.clusterRole)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.NotNil(t, clusterRole, Commentf(test.ErrResultFmt, testName))

			clusterRole, err = kubeClient.RbacV1().ClusterRoles().Get(ctx, testCase.clusterRole.Name, metav1.GetOptions{})
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, clusterRole.Name, testCase.clusterRole.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestDeleteClusterRole(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRole    *rbacv1.ClusterRole
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"Not found": {
			clusterRole: &rbacv1.ClusterRole{
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
				_, err := kubeClient.RbacV1().ClusterRoles().Create(ctx, testCase.clusterRole, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

				clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(ctx, testCase.clusterRole.Name, metav1.GetOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
				assert.Equal(t, clusterRole.Name, testCase.clusterRole.Name, Commentf(test.ErrResultFmt, testName))
			}

			err := DeleteClusterRole(kubeClient, testCase.clusterRole.Name)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

			_, err = kubeClient.RbacV1().ClusterRoles().Get(ctx, testCase.clusterRole.Name, metav1.GetOptions{})
			assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetClusterRole(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRole    *rbacv1.ClusterRole
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"Not found": {
			clusterRole: &rbacv1.ClusterRole{
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
				_, err := kubeClient.RbacV1().ClusterRoles().Create(ctx, testCase.clusterRole, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			clusterRole, err := GetClusterRole(kubeClient, testCase.clusterRole.Name)
			if testCase.expectNotFound {
				assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, clusterRole.Name, testCase.clusterRole.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}
