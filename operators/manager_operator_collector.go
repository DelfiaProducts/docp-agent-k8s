package operators

import (
	"context"
	defaultErrors "errors"
	"os"
	"path/filepath"

	"github.com/OryaHub/agent-k8s/dto"
	pkg "github.com/OryaHub/agent-k8s/pkg"
	corev1 "k8s.io/api/core/v1"
	rbcav1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetCurrentHelmChartVersion return the current helm chart version
func (m *ManagerOperator) GetCurrentHelmChartVersion(namespace, releaseName string) (string, error) {
	currentRelease, err := m.helmClient.GetCurrentRelease(namespace, releaseName)
	if err != nil {
		return "", err
	}
	version := currentRelease.Chart.Metadata.Version
	return version, nil
}

// GetLatestHelmChartVersion return the latest helm chart version
func (m *ManagerOperator) GetLatestHelmChartVersion(namespace, releaseName string) (string, error) {
	latestVersion, err := m.helmClient.GetLatestHelmVersion(namespace, releaseName)
	if err != nil {
		return "", err
	}
	return latestVersion, nil
}

// GetReleaseName return release name
func (m *ManagerOperator) GetReleaseName(namespace, chartName string) (string, error) {
	return m.helmClient.GetReleaseName(namespace, chartName)
}

// GetReleaseModeDatadog return release mode from datadog
func (m *ManagerOperator) GetReleaseModeDatadog(namespace string) (string, error) {
	return m.helmClient.GetReleaseModeDatadog(namespace)
}

// GetConfigMap execute get the config map
func (m *ManagerOperator) GetConfigMap(configMapName, namespace string) (*corev1.ConfigMap, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return configMap, nil
}

// GetNamespaces execute get the namespaces
func (m *ManagerOperator) GetNamespaces() ([]corev1.Namespace, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	if len(namespaces.Items) == 0 {
		return nil, pkg.ErrNotFound
	}
	return namespaces.Items, nil
}

// GetKeyFromConfigMap execute get key from the config map
func (m *ManagerOperator) GetKeyFromConfigMap(key, configMapName, namespace string) (string, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return "", err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", err
		}
	}
	data := configMap.Data
	if val, ok := data[key]; ok {
		return val, nil
	}

	return "", pkg.ConfigMapKeyNotFound
}

// GetClusterRole execute get the cluster role
func (m *ManagerOperator) GetClusterRole(clusterRoleName string) (*rbcav1.ClusterRole, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	clusterRole, err := clientset.RbacV1().ClusterRoles().Get(ctx, clusterRoleName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return clusterRole, nil
}

// GetClusterRoleBinding execute get the cluster role binding
func (m *ManagerOperator) GetClusterRoleBinding(clusterRoleBindingName string) (*rbcav1.ClusterRoleBinding, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBindingName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return clusterRoleBinding, nil
}

// CollectMetadataK8s return metadata from k8s
func (m *ManagerOperator) CollectMetadataK8s() (dto.K8sRegister, error) {
	registerData := dto.K8sRegister{}
	clusterName, err := m.getClusterName()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.ClusterName = clusterName
	nodeNames, err := m.getNodeNames()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.NodeNames = nodeNames
	namespaces, err := m.getNamespaces()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.Namespaces = namespaces
	arch, err := m.getClusterArch()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.Metadata.ComputeInfo.PlatformArch = arch
	return registerData, nil
}

// getServiceName return service name from envs
func (m *ManagerOperator) getServiceName() string {
	return os.Getenv("SERVICE_AGENT_NAME")
}

// getAgentPort return agent port from envs
func (m *ManagerOperator) getAgentPort() string {
	return os.Getenv("AGENT_PORT")
}

func (m *ManagerOperator) getNamespaces() ([]string, error) {
	var slcNamespaces []string
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, namespace := range namespaces.Items {
		slcNamespaces = append(slcNamespaces, namespace.Name)
	}
	return slcNamespaces, nil
}

func (m *ManagerOperator) getClusterArch() (string, error) {
	var arch string
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return "", err
	}
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, node := range nodes.Items {
		arch = node.Status.NodeInfo.Architecture
		if len(arch) > 0 {
			return arch, nil
		}
	}
	return arch, nil
}

// getClusterName return cluster name
func (m *ManagerOperator) getClusterName() (string, error) {
	mode := os.Getenv("MODE")
	if mode == "local" {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			kubeconfig = filepath.Join(homeDir, ".kube", "config")
		}

		config, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			return "", err
		}
		currentContextName := config.CurrentContext

		if currentContextName == "" {
			return "", defaultErrors.New("no current context found in kubeconfig")
		}

		currentContext, ok := config.Contexts[currentContextName]
		if !ok {
			return "", defaultErrors.New("current context not found in kubeconfig")
		}
		clusterName := currentContext.Cluster
		if clusterName == "" {
			return "", defaultErrors.New("no cluster name associated with current context")
		}
		return clusterName, nil
	} else {
		var kubeClusterName string
		clusterName := os.Getenv("CLUSTER_NAME")
		kubernetesHost := os.Getenv("KUBERNETES_SERVICE_HOST")
		if len(clusterName) > 0 {
			kubeClusterName = clusterName
		} else if len(kubernetesHost) > 0 {
			kubeClusterName = kubernetesHost
		}
		return kubeClusterName, nil
	}
}

func (m *ManagerOperator) getNodeNames() ([]string, error) {
	var nodesNames []string
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		nodesNames = append(nodesNames, node.Name)
	}
	return nodesNames, nil
}
