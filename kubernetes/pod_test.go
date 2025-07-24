package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestIsPodContainerInState(t *testing.T) {
	containerName := "test"
	type testCase struct {
		pod           *corev1.Pod
		conditionFunc func(*corev1.ContainerStatus) bool
		expectedState bool
	}
	testCases := map[string]testCase{
		"Container is completed": {
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: containerName,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
			conditionFunc: IsContainerCompleted,
			expectedState: true,
		},
		"Container is not completed": {
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: containerName,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 1,
								},
							},
						},
					},
				},
			},
			conditionFunc: IsContainerCompleted,
			expectedState: false,
		},
		"Init container is completed": {
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: containerName,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
			conditionFunc: IsContainerCompleted,
			expectedState: true,
		},
		"Init container is not completed": {
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: containerName,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 1,
								},
							},
						},
					},
				},
			},
			conditionFunc: IsContainerCompleted,
			expectedState: false,
		},
		"Container is initializing": {
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: containerName,
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "PodInitializing",
								},
							},
						},
					},
				},
			},
			conditionFunc: IsContainerInitializing,
			expectedState: true,
		},
		"Container is ready": {
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:  containerName,
							Ready: true,
						},
					},
				},
			},
			conditionFunc: IsContainerReady,
			expectedState: true,
		},

		"Container is restarted": {
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: containerName,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
							RestartCount: 1,
						},
					},
				},
			},
			conditionFunc: IsContainerRestarted,
			expectedState: true,
		},

		"Container is waiting crash loop back off": {
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: containerName,
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
			},
			conditionFunc: IsContainerWaitingCrashLoopBackOff,
			expectedState: true,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			isInState := IsPodContainerInState(testCase.pod, containerName, testCase.conditionFunc)
			assert.Equal(t, testCase.expectedState, isInState)
		})
	}
}
