package kubernetes

import (
	. "gopkg.in/check.v1"

	corev1 "k8s.io/api/core/v1"
)

func (s *TestSuite) TestIsPodContainerInState(c *C) {
	containerName := "test"
	type testCase struct {
		pod           *corev1.Pod
		conditionFunc func(*corev1.ContainerStatus) bool
		expectedState bool
	}
	testCases := map[string]testCase{
		"IsPodContainerInState(...): container is completed": {
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
		"IsPodContainerInState(...): container is not completed": {
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
		"IsPodContainerInState(...): init container is completed": {
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
		"IsPodContainerInState(...): init container is not completed": {
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
		"IsPodContainerInState(...): container is initializing": {
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
		"IsPodContainerInState(...): container is ready": {
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

		"IsPodContainerInState(...): container is restarted": {
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

		"IsPodContainerInState(...): container is waiting crash loop back off": {
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
		c.Logf("testing kubernetes.%v", testName)

		isInState := IsPodContainerInState(testCase.pod, containerName, testCase.conditionFunc)
		c.Assert(isInState, Equals, testCase.expectedState)
	}

}
