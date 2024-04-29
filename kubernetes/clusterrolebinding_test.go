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

func (s *TestSuite) TestCreateClusterRoleBinding(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRoleBinding *rbacv1.ClusterRoleBinding
		IsAlreadyExists    bool
	}
	testCases := map[string]testCase{
		"CreateClusterRoleBinding(...):": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"CreateClusterRoleBinding(...): already exists": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
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
			_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, testCase.clusterRoleBinding, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		clusterRoleBinding, err := CreateClusterRoleBinding(kubeClient, testCase.clusterRoleBinding)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(clusterRoleBinding, NotNil, Commentf(test.ErrResultFmt, testName))

		clusterRoleBinding, err = kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, testCase.clusterRoleBinding.Name, metav1.GetOptions{})
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(clusterRoleBinding.Name, Equals, testCase.clusterRoleBinding.Name, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestDeleteClusterRoleBinding(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRoleBinding *rbacv1.ClusterRoleBinding
		expectNotFound     bool
	}
	testCases := map[string]testCase{
		"DeleteClusterRoleBinding(...):": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"DeleteClusterRoleBinding(...): not found": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
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
			_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, testCase.clusterRoleBinding, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

			clusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, testCase.clusterRoleBinding.Name, metav1.GetOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
			c.Assert(clusterRoleBinding.Name, Equals, testCase.clusterRoleBinding.Name, Commentf(test.ErrResultFmt, testName))
		}

		err := DeleteClusterRoleBinding(kubeClient, testCase.clusterRoleBinding.Name)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

		_, err = kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, testCase.clusterRoleBinding.Name, metav1.GetOptions{})
		c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetClusterRoleBinding(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		clusterRoleBinding *rbacv1.ClusterRoleBinding
		expectNotFound     bool
	}
	testCases := map[string]testCase{
		"GetClusterRoleBinding(...):": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
		"GetClusterRoleBinding(...): not found": {
			clusterRoleBinding: &rbacv1.ClusterRoleBinding{
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
			_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, testCase.clusterRoleBinding, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		clusterRoleBinding, err := GetClusterRoleBinding(kubeClient, testCase.clusterRoleBinding.Name)
		if testCase.expectNotFound {
			c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
			return
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(clusterRoleBinding.Name, Equals, testCase.clusterRoleBinding.Name, Commentf(test.ErrResultFmt, testName))
	}
}
