package onepassword

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func IsDaemonSetUsingSecrets(daemonSet *appsv1.DaemonSet, secrets map[string]*corev1.Secret) bool {
	volumes := daemonSet.Spec.Template.Spec.Volumes
	containers := daemonSet.Spec.Template.Spec.Containers
	containers = append(containers, daemonSet.Spec.Template.Spec.InitContainers...)
	return AreAnnotationsUsingSecrets(daemonSet.Annotations, secrets) || AreContainersUsingSecrets(containers, secrets) || AreVolumesUsingSecrets(volumes, secrets)
}
