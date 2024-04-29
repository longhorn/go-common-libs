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

func (s *TestSuite) TestCreateConfigMap(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		configMap       *corev1.ConfigMap
		IsAlreadyExists bool
	}
	testCases := map[string]testCase{
		"CreateConfigMap(...):": {
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"CreateConfigMap(...): already exists": {
			configMap: &corev1.ConfigMap{
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
			_, err := kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Create(ctx, testCase.configMap, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		configMap, err := CreateConfigMap(kubeClient, testCase.configMap)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(configMap, NotNil, Commentf(test.ErrResultFmt, testName))

		configMap, err = kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Get(ctx, testCase.configMap.Name, metav1.GetOptions{})
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(configMap.Name, Equals, testCase.configMap.Name, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestDeleteConfigMap(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		configMap      *corev1.ConfigMap
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"DeleteConfigMap(...):": {
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"DeleteConfigMap(...): not found": {
			configMap: &corev1.ConfigMap{
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
			_, err := kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Create(ctx, testCase.configMap, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

			configMap, err := kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Get(ctx, testCase.configMap.Name, metav1.GetOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
			c.Assert(configMap.Name, Equals, testCase.configMap.Name, Commentf(test.ErrResultFmt, testName))
		}

		err := DeleteConfigMap(kubeClient, testCase.configMap.Namespace, testCase.configMap.Name)
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))

		_, err = kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Get(ctx, testCase.configMap.Name, metav1.GetOptions{})
		c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
	}
}

func (s *TestSuite) TestGetConfigMap(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		configMap      *corev1.ConfigMap
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"GetConfigMap(...):": {
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"GetConfigMap(...): not found": {
			configMap: &corev1.ConfigMap{
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
			_, err := kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Create(ctx, testCase.configMap, metav1.CreateOptions{})
			c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		}

		configMap, err := GetConfigMap(kubeClient, testCase.configMap.Namespace, testCase.configMap.Name)
		if testCase.expectNotFound {
			c.Assert(apierrors.IsNotFound(err), Equals, true, Commentf(test.ErrResultFmt, testName))
			return
		}
		c.Assert(err, IsNil, Commentf(test.ErrErrorFmt, testName))
		c.Assert(configMap.Name, Equals, testCase.configMap.Name, Commentf(test.ErrResultFmt, testName))
	}
}
