package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/longhorn/go-common-libs/test"
)

func TestCreateConfigMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		configMap       *corev1.ConfigMap
		IsAlreadyExists bool
	}
	testCases := map[string]testCase{
		"Existing": {
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Already exists": {
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
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if testCase.IsAlreadyExists {
				_, err := kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Create(ctx, testCase.configMap, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			configMap, err := CreateConfigMap(kubeClient, testCase.configMap)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.NotNil(t, configMap, Commentf(test.ErrResultFmt, testName))

			configMap, err = kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Get(ctx, testCase.configMap.Name, metav1.GetOptions{})
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, configMap.Name, testCase.configMap.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestDeleteConfigMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		configMap      *corev1.ConfigMap
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Not found": {
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
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if !testCase.expectNotFound {
				_, err := kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Create(ctx, testCase.configMap, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

				configMap, err := kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Get(ctx, testCase.configMap.Name, metav1.GetOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
				assert.Equal(t, configMap.Name, testCase.configMap.Name, Commentf(test.ErrResultFmt, testName))
			}

			err := DeleteConfigMap(kubeClient, testCase.configMap.Namespace, testCase.configMap.Name)
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))

			_, err = kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Get(ctx, testCase.configMap.Name, metav1.GetOptions{})
			assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetConfigMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testCase struct {
		configMap      *corev1.ConfigMap
		expectNotFound bool
	}
	testCases := map[string]testCase{
		"Existing": {
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
		},
		"Not found": {
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
		t.Run(testName, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()

			if !testCase.expectNotFound {
				_, err := kubeClient.CoreV1().ConfigMaps(testCase.configMap.Namespace).Create(ctx, testCase.configMap, metav1.CreateOptions{})
				assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			}

			configMap, err := GetConfigMap(kubeClient, testCase.configMap.Namespace, testCase.configMap.Name)
			if testCase.expectNotFound {
				assert.True(t, apierrors.IsNotFound(err), Commentf(test.ErrResultFmt, testName))
				return
			}
			assert.NoError(t, err, Commentf(test.ErrErrorFmt, testName))
			assert.Equal(t, configMap.Name, testCase.configMap.Name, Commentf(test.ErrResultFmt, testName))
		})
	}
}
