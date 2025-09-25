package templates

import (
	"fmt"
	"strings"

	"github.com/OryaHub/agent-k8s/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TemplateJobAutoUninstall return job the auto uninstall
func TemplateJobAutoUninstall(namespace string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	managerName := utils.GetDocpDeploymentName("manager")
	agentName := utils.GetDocpDeploymentName("agent")
	webhookName := utils.GetDocpDeploymentName("webhook")
	commandManager := fmt.Sprintf("kubectl delete deployment %s -n %s", managerName, namespace)
	commandAgent := fmt.Sprintf("kubectl delete deployment %s -n %s", agentName, namespace)
	commandWebhook := fmt.Sprintf("kubectl delete deployment %s -n %s", webhookName, namespace)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docp-auto-uninstall",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "docp-job-auto-uninstall",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "docp-pod-auto-uninstall",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  "job-remove-manager",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								commandManager,
								"echo 'deployment deletion command executed'",
							},
						},
						{
							Name:  "job-remove-agent",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								commandAgent,
								"echo 'deployment deletion command executed'",
							},
						},
						{
							Name:  "job-remove-webhook",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								commandWebhook,
								"echo 'deployment deletion command executed'",
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// TemplateJobRemoveConfiMaps return job the remove config maps
func TemplateJobRemoveConfiMaps(namespace string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	configMapStateName := utils.GetDocpConfiMapStateName()
	configMapConfigurationName := utils.GetDocpConfigMapConfigurationsName()
	commandConfigMapState := fmt.Sprintf("kubectl delete configmap %s -n %s", configMapStateName, namespace)
	commandConfigMapConfiguration := fmt.Sprintf("kubectl delete configmap %s -n %s", configMapConfigurationName, namespace)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docp-remove-config-maps",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "docp-job-remove-config-maps",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "docp-pod-config-maps",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  "job-remove-config-map-state",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								commandConfigMapState,
							},
						}, {
							Name:  "job-remove-config-map-configuration",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								commandConfigMapConfiguration,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// TemplateJobRemoveDocpNamespace return job the remove docp namespace
func TemplateJobRemoveDocpNamespace(namespace string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	command := fmt.Sprintf("kubectl delete ns %s", namespace)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docp-remove-docp-namespace",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "docp-job-remove-docp-namespace",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "docp-pod-remove-docp-namespace",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  "job-remove-docp-namespace",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								command,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// TemplateJobRemoveMutatingWebhook return job the remove docp mutate webhook
func TemplateJobRemoveMutatingWebhook(mutateName, namespace string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	command := fmt.Sprintf("kubectl delete mutatingwebhookconfiguration %s -n %s", mutateName, namespace)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docp-remove-docp-mutate-agent",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "docp-job-remove-docp-mutate-agent",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "docp-pod-remove-docp-mutate-agent",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  "job-remove-docp-mutate-agent",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								command,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// TemplateJobRemoveClusterRole return job the remove docp cluster role
func TemplateJobRemoveClusterRole(clusterRoleName, namespace string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	command := fmt.Sprintf("kubectl delete clusterrole %s -n %s", clusterRoleName, namespace)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docp-remove-docp-cluster-role",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "docp-job-remove-docp-cluster-role",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "docp-pod-remove-docp-cluster-role",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  "job-remove-docp-cluster-role",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								command,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// TemplateJobRemoveClusterRoleBinding return job the remove docp cluster role binding
func TemplateJobRemoveClusterRoleBinding(clusterRoleBindingName, namespace string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	command := fmt.Sprintf("kubectl delete clusterrolebinding %s -n %s", clusterRoleBindingName, namespace)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docp-remove-docp-cluster-role-binding",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "docp-job-remove-docp-cluster-role-binding",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "docp-pod-remove-docp-cluster-role-binding",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  "job-remove-docp-cluster-role-binding",
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								command,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// TemplateJobRemoveDocpHelmRelease return job the remove docp helm release
func TemplateJobRemoveDocpHelmRelease(releaseName, namespace string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	command := fmt.Sprintf("helm uninstall %s -n %s --wait --timeout=5m", releaseName, namespace)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docp-remove-docp-helm-release",
			Namespace: namespace,
			Labels: map[string]string{
				"app":     "docp-job-remove-docp-helm-release",
				"release": releaseName,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "docp-pod-remove-docp-helm-release",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  "job-remove-docp-helm-release",
							Image: "alpine/helm:3.11.1",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								command,
							},
							Env: []corev1.EnvVar{
								{
									Name:  "HELM_DRIVER",
									Value: "secret",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// TemplateJobUpdateDeploymentImage return job the update deployment image
func TemplateJobUpdateDeploymentImage(namespace, deploymentName, containerName, imageName string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	command := fmt.Sprintf("kubectl set image deployment/%s %s=%s -n %s", deploymentName, containerName, imageName, namespace)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("docp-update-%s", deploymentName),
			Namespace: namespace,
			Labels: map[string]string{
				"app": fmt.Sprintf("docp-job-update-%s", deploymentName),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("docp-pod-update-%s", deploymentName),
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  fmt.Sprintf("job-update-%s", containerName),
							Image: "bitnami/kubectl:1.29.3",
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								command,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// // TemplateJobAutoUpdateHelmRelease returns a job that performs auto update of a Helm release
func TemplateJobAutoUpdateHelmRelease(namespace, releaseName, repositoryURL, targetVersion, repositoryImage string) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	ulidName := fmt.Sprintf("docp-auto-update-helm-release-%s", strings.ToLower(utils.GetUlid()))
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ulidName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     ulidName,
				"release": releaseName,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": ulidName,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  ulidName,
							Image: repositoryImage, // substitua pela sua imagem
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								"./app",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "NAMESPACE",
									Value: namespace,
								},
								{
									Name:  "RELEASE_NAME",
									Value: releaseName,
								},
								{
									Name:  "REPOSITORY_URL",
									Value: repositoryURL,
								},
								{
									Name:  "TARGET_VERSION",
									Value: targetVersion,
								},
								{
									Name:  "LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "MANAGER_DEPLOYMENT_NAME",
									Value: utils.GetDocpDeploymentName("manager"),
								},
								{
									Name:  "AGENT_DEPLOYMENT_NAME",
									Value: utils.GetDocpDeploymentName("agent"),
								},
								{
									Name:  "WEBHOOK_DEPLOYMENT_NAME",
									Value: utils.GetDocpDeploymentName("webhook"),
								},
								{
									Name:  "HELM_DRIVER",
									Value: "secret",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

// TemplateJobRunUpdateAgent retorna um job que executa update do agente docp
func TemplateJobRunUpdateAgent(namespace, jobName, imageName string, command []string, args []string, envVars []corev1.EnvVar) batchv1.Job {
	backOffLimit := int32(5)
	ttlPod := int32(300)
	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": fmt.Sprintf("docp-job-%s", jobName),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlPod,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("docp-pod-%s", jobName),
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: utils.GetServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:    jobName,
							Image:   imageName,
							Command: command,
							Args:    args,
							Env:     envVars,
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}
