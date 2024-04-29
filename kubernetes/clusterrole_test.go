package kubernetes

import (
	"context"

	"github.com/longhorn/go-common-libs/test"
	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *TestSuite) TestCreateClusterRole(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRole     *rbacv1.ClusterRole
		IsAlreadyExists bool
	}
	testCases := map[string]testCase{
		"CreateClusterRole(...):": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"CreateClusterRole(...): already exists": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			IsAlreadyExists: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing kubernetes.%v", testName)

		kubeClient := fake.NewSimpleClientset()

		if testCase.IsAlreadyExists {
			_, err := kubeClient.RbacV1().ClusterRoles().Create(ctx, testCase.clusterRole, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		clusterRole, err := CreateClusterRole(kubeClient, testCase.clusterRole)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(clusterRole, NotNil, Commentf(test.ErrResultFmt, testName))

		clusterRole, err = kubeClient.RbacV1().ClusterRoles().Get(ctx, testCase.clusterRole.Name, metav1.GetOptions{})
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(clusterRole.Name, Equals, testCase.clusterRole.Name, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestDeleteClusterRole(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRole    *rbacv1.ClusterRole
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"DeleteClusterRole(...):": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"DeleteClusterRole(...): not found": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expectNotFound: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing kubernetes.%v", testName)

		kubeClient := fake.NewSimpleClientset()

		if !testCase.expectNotFound {
			_, err := kubeClient.RbacV1().ClusterRoles().Create(ctx, testCase.clusterRole, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

			clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(ctx, testCase.clusterRole.Name, metav1.GetOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
			c.Assert(clusterRole.Name, Equals, testCase.clusterRole.Name, Commentf(test.ErrResultFmt, testName))
		}

		err := DeleteClusterRole(kubeClient, testCase.clusterRole.Name)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

		_, err = kubeClient.RbacV1().ClusterRoles().Get(ctx, testCase.clusterRole.Name, metav1.GetOptions{})
		c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetClusterRole(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRole    *rbacv1.ClusterRole
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"GetClusterRole(...):": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"GetClusterRole(...): not found": {
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expectNotFound: true,
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing kubernetes.%v", testName)

		kubeClient := fake.NewSimpleClientset()

		if !testCase.expectNotFound {
			_, err := kubeClient.RbacV1().ClusterRoles().Create(ctx, testCase.clusterRole, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		clusterRole, err := GetClusterRole(kubeClient, testCase.clusterRole.Name)
		if testCase.expectNotFound {
			c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
			return
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(clusterRole.Name, Equals, testCase.clusterRole.Name, Commentf(test.ErrResultFmt, testName))
	}
}
