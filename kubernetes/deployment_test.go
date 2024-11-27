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

func (s *TestSuite) TestGetDeployment(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		deployment     *appsv1.Deployment
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"GetDeployment(...):": {
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"GetDeployment(...): not found": {
			deployment: &appsv1.Deployment{
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
			_, err := kubeClient.AppsV1().Deployments(testCase.deployment.Namespace).Create(ctx, testCase.deployment, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		deployment, err := GetDeployment(kubeClient, testCase.deployment.Namespace, testCase.deployment.Name)
		if testCase.expectNotFound {
			c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(deployment.Name, Equals, testCase.deployment.Name, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestListDeployments(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		deployments   []*appsv1.Deployment
		labelSelector map[string]string
		skipCreate    bool
		expectError   bool
	}
	testCases := map[string]testCase{
		"ListDeployments(...):": {
			deployments: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
		},
		"ListDeployments(...): not found": {
			deployments: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			skipCreate: true,
		},
		"ListDeployments(...): with single label": {
			deployments: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							"foo":  "bar",
							"baz":  "qux",
							"quux": "corge",
						},
					},
				},
			},
			labelSelector: map[string]string{
				"foo": "bar",
			},
		},
		"ListDeployments(...): with multiple labels": {
			deployments: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							"foo":  "bar",
							"baz":  "qux",
							"quux": "corge",
						},
					},
				},
			},
			labelSelector: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
		},
	}
	for testName, testCase := range testCases {
		c.Logf("testing kubernetes.%v", testName)

		kubeClient := fake.NewSimpleClientset()

		for _, deployment := range testCase.deployments {
			if testCase.skipCreate {
				continue
			}

			_, err := kubeClient.AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		deployments, err := ListDeployments(kubeClient, testCase.deployments[0].Namespace, testCase.labelSelector)
		if testCase.expectError {
			c.Assert(err, NotNil, Commentf(test.ErrErrorFmt, testName))
			continue
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

		if testCase.skipCreate {
			c.Assert(len(deployments.Items), Equals, 0, Commentf(test.ErrResultFmt, testName))
			continue
		}

		c.Assert(len(deployments.Items), Equals, len(testCase.deployments), Commentf(test.ErrResultFmt, testName))
	}
}
