package operators

import (
	"context"
	"strings"

	"github.com/OryaHub/agent-k8s/templates"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbcav1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// CreateConfigMap execute creation the config map
func (m *ManagerOperator) CreateConfigMap(configMapName string, namespace string, data map[string]string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			newConfiMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
				},
				Data: data,
			}
			_, errCreate := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &newConfiMap, metav1.CreateOptions{})
			if errCreate != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateConfigMap execute update the config map
func (m *ManagerOperator) UpdateConfigMap(configMapName string, namespace string, data map[string]string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			newConfiMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
				},
				Data: data,
			}
			_, errCreate := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &newConfiMap, metav1.CreateOptions{})
			if errCreate != nil {
				return err
			}
		}
	}
	configMap.Data = data
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteConfigMap execute remove the config map
func (m *ManagerOperator) DeleteConfigMap(configMapName string, namespace string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	if err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

// ListConfigMaps execute get the all config maps in a namespace
func (m *ManagerOperator) ListConfigMaps(namespace string) (*corev1.ConfigMapList, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return configMaps, nil
}

// ListClusterRole execute get the all cluster role
func (m *ManagerOperator) ListClusterRole() (*rbcav1.ClusterRoleList, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return clusterRoles, nil
}

// DeleteClusterRole execute remove the cluster role
func (m *ManagerOperator) DeleteClusterRole(clusterRoleName string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	if err := clientset.RbacV1().ClusterRoles().Delete(ctx, clusterRoleName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// ListClusterRoleBinding execute get the all cluster role bindings
func (m *ManagerOperator) ListClusterRoleBindig() (*rbcav1.ClusterRoleBindingList, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return clusterRoleBindings, nil
}

// DeleteClusterRoleBinding execute remove the cluster role binding
func (m *ManagerOperator) DeleteClusterRoleBinding(clusterRoleBindingName string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	if err := clientset.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBindingName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// ListCustomResourceDefinitions execute get the all custom resource definitions
func (m *ManagerOperator) ListCustomResourceDefinitions() (*unstructured.UnstructuredList, error) {
	ctx := context.Background()
	dynamicClient, err := dynamic.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	crds, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return crds, nil
}

// DeleteCustomResourceDefinition execute remove the custom resource definition
func (m *ManagerOperator) DeleteCustomResourceDefinition(name string) error {
	ctx := context.Background()
	dynamicClient, err := dynamic.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	if err := dynamicClient.Resource(gvr).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// DeleteNamespace execute remove the namespace
func (m *ManagerOperator) DeleteNamespace(namespace string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}

	if err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// DeleteMutatingWebhook execute remove the mutating webhook
func (m *ManagerOperator) DeleteMutatingWebhook(mutatingName string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	if err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, mutatingName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// CreateJob execute creation the job
func (m *ManagerOperator) CreateJob(namespace string, job *batchv1.Job) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	_, err = clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// RemoveConfigMaps execute remove config maps the orya agent
func (m *ManagerOperator) RemoveConfigMaps(namespace string) error {
	job := templates.TemplateJobRemoveConfiMaps(namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveOryaNamespace execute remove orya namespace
func (m *ManagerOperator) RemoveOryaNamespace(namespace string) error {
	job := templates.TemplateJobRemoveOryaNamespace(namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveOryaHelmRelease execute remove orya helm release
func (m *ManagerOperator) RemoveOryaHelmRelease(releaseName, namespace string) error {
	job := templates.TemplateJobRemoveOryaHelmRelease(releaseName, namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveOryaMutatingAgent execute remove orya mutating agent
func (m *ManagerOperator) RemoveOryaMutatingAgent(mutateName, namespace string) error {
	job := templates.TemplateJobRemoveMutatingWebhook(mutateName, namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveOryaClusterRole execute remove orya cluster role
func (m *ManagerOperator) RemoveOryaClusterRole(clusterRoleName, namespace string) error {
	job := templates.TemplateJobRemoveClusterRole(clusterRoleName, namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveOryaClusterRoleBinding execute remove orya cluster role binding
func (m *ManagerOperator) RemoveOryaClusterRoleBinding(clusterRoleBindingName, namespace string) error {
	job := templates.TemplateJobRemoveClusterRoleBinding(clusterRoleBindingName, namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// UpdateDeploymentImage execute update the deployment image
func (m *ManagerOperator) UpdateDeploymentImage(namespace, deploymentName, containerName, imageName string) error {
	m.logger.Debug("update deployment image", "namespace", namespace, "deploymentName", deploymentName, "containerName", containerName, "imageName", imageName)
	job := templates.TemplateJobUpdateDeploymentImage(namespace, deploymentName, containerName, imageName)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// CleanningDatadogLastInstalation execute cleanning the last instalation the datadog
// Removes leftover resources from previous Datadog installations including:
// - ClusterRoles, ClusterRoleBindings, ConfigMaps, and CRDs (datadoghq.com).
func (m *ManagerOperator) CleanningDatadogLastInstalation() error {
	prefix := "datadog"

	m.logger.Info("cleaning up previous Datadog installation")

	// 1. Clean up ClusterRoles and ClusterRoleBindings
	if err := m.cleanupDatadogClusterRoles(prefix); err != nil {
		m.logger.Error("cleanup datadog cluster roles", "error", err.Error())
	}

	if err := m.cleanupDatadogClusterRoleBindings(prefix); err != nil {
		m.logger.Error("cleanup datadog cluster role bindings", "error", err.Error())
	}

	// 2. Clean up ConfigMaps with "datadog" prefix in all namespaces
	if err := m.cleanupDatadogConfigMaps(prefix); err != nil {
		m.logger.Error("cleanup datadog configmaps", "error", err.Error())
	}

	// 3. Clean up datadoghq.com CRDs
	if err := m.cleanupDatadogCRDs(); err != nil {
		m.logger.Error("cleanup datadog crds", "error", err.Error())
	}

	return nil
}

// cleanupDatadogClusterRoles removes all ClusterRoles with the given prefix
func (m *ManagerOperator) cleanupDatadogClusterRoles(prefix string) error {
	clusterRoles, err := m.ListClusterRole()
	if err != nil {
		return err
	}
	for _, cr := range clusterRoles.Items {
		if strings.HasPrefix(cr.Name, prefix) {
			m.logger.Debug("removing datadog cluster role", "name", cr.Name)
			if err := m.DeleteClusterRole(cr.Name); err != nil {
				m.logger.Error("failed to delete cluster role", "name", cr.Name, "error", err.Error())
			}
		}
	}
	return nil
}

// cleanupDatadogClusterRoleBindings removes all ClusterRoleBindings with the given prefix
func (m *ManagerOperator) cleanupDatadogClusterRoleBindings(prefix string) error {
	clusterRoleBindings, err := m.ListClusterRoleBindig()
	if err != nil {
		return err
	}
	for _, crb := range clusterRoleBindings.Items {
		if strings.HasPrefix(crb.Name, prefix) {
			m.logger.Debug("removing datadog cluster role binding", "name", crb.Name)
			if err := m.DeleteClusterRoleBinding(crb.Name); err != nil {
				m.logger.Error("failed to delete cluster role binding", "name", crb.Name, "error", err.Error())
			}
		}
	}
	return nil
}

// cleanupDatadogConfigMaps removes all ConfigMaps with the given prefix across all namespaces
func (m *ManagerOperator) cleanupDatadogConfigMaps(prefix string) error {
	namespaces, err := m.GetNamespaces()
	if err != nil {
		return err
	}
	for _, ns := range namespaces {
		configMaps, err := m.ListConfigMaps(ns.Name)
		if err != nil {
			m.logger.Debug("skip listing configmaps for namespace", "namespace", ns.Name, "error", err.Error())
			continue
		}
		for _, cm := range configMaps.Items {
			if strings.HasPrefix(cm.Name, prefix) {
				m.logger.Debug("removing datadog configmap", "name", cm.Name, "namespace", ns.Name)
				if err := m.DeleteConfigMap(cm.Name, ns.Name); err != nil {
					m.logger.Error("failed to delete configmap", "name", cm.Name, "namespace", ns.Name, "error", err.Error())
				}
			}
		}
	}
	return nil
}

// cleanupDatadogCRDs removes all CRDs from the datadoghq.com API group
func (m *ManagerOperator) cleanupDatadogCRDs() error {
	crds, err := m.ListCustomResourceDefinitions()
	if err != nil {
		return err
	}
	for _, crd := range crds.Items {
		spec, ok := crd.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		group, ok := spec["group"].(string)
		if !ok {
			continue
		}
		if strings.HasSuffix(group, "datadoghq.com") {
			m.logger.Debug("removing datadog CRD", "name", crd.GetName(), "group", group)
			if err := m.DeleteCustomResourceDefinition(crd.GetName()); err != nil {
				m.logger.Error("failed to delete CRD", "name", crd.GetName(), "group", group, "error", err.Error())
			}
		}
	}
	return nil
}

// populateFactorySignals populate map the signals
func (m *ManagerOperator) populateFactorySignals() error {
	m.mapFactorySignal["update_agent"] = m.SignalUpdateAgent
	m.mapFactorySignal["update_vendor"] = m.SignalUpdateVendor
	m.mapFactorySignal["uninstall"] = m.SignalUninstall
	m.mapFactorySignal["debug-session"] = m.SignalDebugSession
	return nil
}

// populateFactoryVendorsSignalsUninstall populate map the vendors signals uninstall
func (m *ManagerOperator) populateFactoryVendorsSignalsUninstall() error {
	m.mapFactoryVendorsSignalUninstall["datadog"] = m.DatadogUninstall
	return nil
}

// populateFactoryVendorsSignalsUpdate populate map the vendors signals update
func (m *ManagerOperator) populateFactoryVendorsSignalsUpdate() error {
	m.mapFactoryVendorsSignalUpdate["datadog"] = m.DatadogUpdate
	return nil
}

// removeAllVendors verify if all remove vendors
func (m *ManagerOperator) removeAllVendors(vendors []string) bool {
	for _, vendor := range vendors {
		if strings.Contains(vendor, "all") {
			return true
		}
	}
	return false
}
