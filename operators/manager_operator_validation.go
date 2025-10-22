package operators

import (
	"context"
	"strings"

	"github.com/OryaHub/agent-k8s/dto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ValidateVersionHelmAlreadyUpdated return if helm chart version is already updated
func (m *ManagerOperator) ValidateVersionHelmAlreadyUpdated(currentVersion, targetVersion string) bool {
	return targetVersion == currentVersion
}

// DatadogAlreadyInstalled execute validation if datadog exists and return data from instalation
func (m *ManagerOperator) DatadogAlreadyInstalled(resourceName string, namespaces []corev1.Namespace) (dto.VendorInstalled, error) {
	vendor := dto.VendorInstalled{
		Installed: false,
	}
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return vendor, err
	}

	for _, namespace := range namespaces {
		pods, err := clientset.CoreV1().Pods(namespace.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			continue
		}
		prefix := "datadog"
		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, prefix) {
				vendor.Installed = true
				vendor.Namespace = namespace.Name

				labels := pod.GetLabels()

				mode := "unknown"
				if managedBy, ok := labels["app.kubernetes.io/managed-by"]; ok && managedBy == "datadog-operator" {
					mode = "operator"
				} else if managedBy, ok := labels["app.kubernetes.io/managed-by"]; ok && managedBy == "Helm" {
					mode = "helm"
				}
				vendor.Mode = mode
				return vendor, nil
			}
		}

	}

	return vendor, nil
}

// VerifyDatadogResourceExists identify if exist resource datadog
func (m *ManagerOperator) VerifyDatadogResourceExists(resourceName, namespace string) (bool, error) {
	clientsetDynamic, err := dynamic.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return false, err
	}
	gvr := schema.GroupVersionResource{
		Group:    "datadoghq.com",
		Version:  "v2alpha1",
		Resource: "datadogagents",
	}

	objDatadog, err := clientsetDynamic.Resource(gvr).Namespace(namespace).Get(context.Background(), resourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return true, nil
		}
	}

	m.logger.Debug("verify datadog resource exists", "objDatadog", objDatadog)
	if objDatadog != nil {
		return true, nil
	}
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return false, err
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	prefix := "datadog"
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, prefix) {
			m.logger.Debug("verify datadog resource exists", "pod name datadog", pod.Name)
			return true, nil
		}
	}
	return false, nil
}

// IsLocked return if operator locked for send transactions events
func (m *ManagerOperator) IsLockedEvents() bool {
	return m.LockedEvents
}

// verifyAndUpdateLockedEvents verify and update locked events variable
func (m *ManagerOperator) verifyAndUpdateLockedEvents() error {
	m.logger.Debug("verify and update locked events", "pendingTransactionEvents", m.pendingTransactionEvents)
	if len(m.pendingTransactionEvents) == 0 {
		m.LockedEvents = false
	}
	return nil
}
