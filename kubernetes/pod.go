package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
)

func IsContainerCompleted(status *corev1.ContainerStatus) bool {
	return status.State.Terminated != nil && status.State.Terminated.ExitCode == 0
}

func IsContainerInitializing(status *corev1.ContainerStatus) bool {
	return status.State.Waiting != nil && status.State.Waiting.Reason == "PodInitializing"
}

func IsContainerReady(status *corev1.ContainerStatus) bool {
	return status.Ready
}

func IsContainerRestarted(status *corev1.ContainerStatus) bool {
	return status.State.Terminated != nil && status.RestartCount > 0
}

func IsContainerWaitingCrashLoopBackOff(status *corev1.ContainerStatus) bool {
	return status.State.Waiting != nil && status.State.Waiting.Reason == "CrashLoopBackOff"
}

func IsPodContainerInState(pod *corev1.Pod, containerName string, conditionFunc func(*corev1.ContainerStatus) bool) bool {
	containerStatuses := pod.DeepCopy().Status.InitContainerStatuses
	containerStatuses = append(containerStatuses, pod.DeepCopy().Status.ContainerStatuses...)

	// During the initialization phase of a pod, the statuses of init containers
	// may not be populated immediately. If this function is called during this
	// phase and the target container is an init container, return false to indicate
	// that the container is not yet the desired state.
	isChecked := false

	for _, status := range containerStatuses {
		if status.Name != containerName {
			continue
		}

		if !conditionFunc(&status) {
			return false
		}

		isChecked = true
	}
	return isChecked
}
