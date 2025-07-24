package kubernetes

import (
	"context"
	"testing"

	"github.com/longhorn/go-common-libs/test"
	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetDeployment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		deployment     *appsv1.Deployment
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Not found": {
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
		t.Run(testName, func(t *testing.T) {

			kubeClient := fake.NewSimpleClientset()
			if !testCase.expectNotFound {
				_, err := kubeClient.AppsV1().Deployments(testCase.deployment.Namespace).Create(ctx, testCase.deployment, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			deployment, err := GetDeployment(kubeClient, testCase.deployment.Namespace, testCase.deployment.Name)
			if testCase.expectNotFound {
				assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, deployment.Name, testCase.deployment.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestListDeployments(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		deployments   []*appsv1.Deployment
		labelSelector map[string]string
		skipCreate    bool
		expectError   bool
	}
	testCases := map[string]testCase{
		"Existing": {
			deployments: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
		},
		"Not found": {
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
		"With single label": {
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
		"With multiple labels": {
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
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			for _, deployment := range testCase.deployments {
				if testCase.skipCreate {
					continue
				}

				_, err := kubeClient.AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			deployments, err := ListDeployments(kubeClient, testCase.deployments[0].Namespace, testCase.labelSelector)
			if testCase.expectError {
				assert.Error(t, err, Commentf(test.ErrErrorFmt, testName))
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

			if testCase.skipCreate {
				assert.Equal(t, 0, len(deployments.Items), Commentf(test.ErrResultFmt, testName))
				return
			}

			assert.Equal(t, len(deployments.Items), len(testCase.deployments), Commentf(test.ErrResultFmt, testName))
		})
	}
}
