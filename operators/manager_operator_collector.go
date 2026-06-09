package operators

import (
	"bytes"
	"context"
	defaultErrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	pkg "github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
	corev1 "k8s.io/api/core/v1"
	rbcav1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
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

	uniqID, err := m.GetOrCreateClusterID()
	if err != nil {
		m.logger.Warn("collect metadata: could not get/create cluster orya_id", "error", err.Error())
	} else {
		registerData.Metadata.ComputeInfo.OryaId = uniqID
	}

	clusterName, err := m.GetClusterName()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.ClusterName = clusterName
	registerData.Metadata.ComputeInfo.ComputeName = clusterName
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
	ns, err := m.GetNamespaces()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	vendor, err := m.DatadogAlreadyInstalled("datadog", ns)
	if err != nil {
		return dto.K8sRegister{}, err
	}

	if vendor.Installed {
		//collect vendor infos
		vendorInfos, err := m.GetVendorInfos("datadog", vendor.Namespace)
		if err != nil {
			return dto.K8sRegister{}, err
		}
		registerData.Metadata.VendorsInfo = vendorInfos
	}
	return registerData, nil
}

// GetVendorInfos return vendor infos
func (m *ManagerOperator) GetVendorInfos(name, namespace string) (dto.VendorInfo, error) {
	switch name {
	case "datadog":
		return m.geDatadogInfos(namespace)
	default:
		return dto.VendorInfo{}, fmt.Errorf("vendor %s not supported", name)
	}
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

// extractClusterNameFromYamlContent parses a Datadog deploy-yml and returns
// clusterName if explicitly set. Tries both helm values format and operator
// CRD format, returning the first match found.
func extractClusterNameFromYamlContent(content string) string {
	if len(content) == 0 {
		return ""
	}
	// Try helm format: datadog.clusterName
	var values map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &values); err == nil {
		if datadogRaw, ok := values["datadog"]; ok {
			if datadogMap, ok := datadogRaw.(map[string]interface{}); ok {
				if name, ok := datadogMap["clusterName"]; ok {
					if s, ok := name.(string); ok && len(s) > 0 {
						return s
					}
				}
			}
		}
	}
	// Try operator CRD format: spec.global.clusterName
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(content), 4096)
	var obj unstructured.Unstructured
	if err := decoder.Decode(&obj); err == nil {
		if name, _, _ := unstructured.NestedString(obj.Object, "spec", "global", "clusterName"); len(name) > 0 {
			return name
		}
	}
	return ""
}

// GetEffectiveClusterName returns the cluster name to use for Datadog config.
// If the deploy-yml already has a clusterName set, it returns that value
// (the deploy-yml is authoritative). Otherwise, it falls back to the cascade
// detection strategy via GetClusterName().
func (m *ManagerOperator) GetEffectiveClusterName(content string) (string, error) {
	if name := extractClusterNameFromYamlContent(content); len(name) > 0 {
		return name, nil
	}
	return m.GetClusterName()
}

// getDatadogClusterName attempts to read the cluster name from an installed
// Datadog agent's "agent status" output. Returns empty string if Datadog is
// not installed or the name cannot be determined.
func (m *ManagerOperator) getDatadogClusterName() (string, error) {
	nsList, err := m.GetNamespaces()
	if err != nil {
		return "", err
	}
	vendor, err := m.DatadogAlreadyInstalled("datadog", nsList)
	if err != nil {
		return "", err
	}
	if !vendor.Installed {
		return "", nil
	}
	info, err := m.geDatadogInfos(vendor.Namespace)
	if err != nil {
		return "", err
	}
	return info.Datadog.ClusterName, nil
}

// GetClusterName return cluster name using a cascade strategy:
//  1. Datadog agent cluster name (if installed — authoritative, pois é o
//     cluster name que está efetivamente rodando no Datadog)
//  2. CLUSTER_NAME env var (admin hint, usado quando Datadog não está rodando)
//  3. Provider-specific detection (node labels, API hostname, cloud metadata)
//  4. KUBERNETES_SERVICE_HOST (fallback)
func (m *ManagerOperator) GetClusterName() (string, error) {
	mode := os.Getenv("MODE")
	if mode == "local" {
		return m.getClusterNameFromKubeconfig()
	}

	// Datadog já instalado tem prioridade: o cluster name que está rodando
	// no Datadog é a fonte da verdade, independente do valor em values.yaml.
	name, err := m.getDatadogClusterName()
	if err == nil && len(name) > 0 {
		return name, nil
	}

	// CLUSTER_NAME env var serve como fallback para quando Datadog ainda
	// não está instalado ou não foi possível determinar o nome.
	if name := os.Getenv("CLUSTER_NAME"); len(name) > 0 {
		return name, nil
	}

	name, err = m.detectClusterName()
	if err == nil && len(name) > 0 {
		return name, nil
	}

	if host := os.Getenv("KUBERNETES_SERVICE_HOST"); len(host) > 0 {
		return host, nil
	}

	return "", defaultErrors.New("could not determine cluster name")
}

// getClusterNameFromKubeconfig reads cluster name from the kubeconfig context.
func (m *ManagerOperator) getClusterNameFromKubeconfig() (string, error) {
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
}

// detectClusterName attempts to determine the cluster name from provider-specific
// node labels, API server hostname patterns, or cloud metadata endpoints.
func (m *ManagerOperator) detectClusterName() (string, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return "", err
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return "", err
	}
	if len(nodes.Items) == 0 {
		return "", defaultErrors.New("no nodes found")
	}
	node := nodes.Items[0]

	for _, key := range []string{
		"alpha.eksctl.io/cluster-name",
		"kubernetes.azure.com/cluster",
		"kind.x-k8s.io/cluster",
	} {
		if val, ok := node.Labels[key]; ok && len(val) > 0 {
			return val, nil
		}
	}

	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	if strings.Contains(host, "eks.amazonaws.com") {
		parts := strings.SplitN(host, ".", 2)
		if len(parts[0]) > 0 {
			return parts[0], nil
		}
	}

	if strings.HasPrefix(node.Spec.ProviderID, "gce://") {
		return m.getClusterNameFromGCE()
	}

	// kubeadm: read from ConfigMap in kube-system
	if name, err := m.getClusterNameFromKubeadm(ctx, clientset); err == nil && len(name) > 0 {
		return name, nil
	}

	return "", defaultErrors.New("no provider-specific cluster name found")
}

// getClusterNameFromGCE calls the GCE metadata endpoint for the GKE cluster name.
func (m *ManagerOperator) getClusterNameFromGCE() (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/attributes/cluster-name", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GCE metadata returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// getClusterNameFromKubeadm reads the cluster name from the kubeadm-config
// ConfigMap in the kube-system namespace.
func (m *ManagerOperator) getClusterNameFromKubeadm(ctx context.Context, clientset *kubernetes.Clientset) (string, error) {
	cm, err := clientset.CoreV1().ConfigMaps("kube-system").Get(ctx, "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	raw, ok := cm.Data["ClusterConfiguration"]
	if !ok || len(raw) == 0 {
		return "", defaultErrors.New("kubeadm-config has no ClusterConfiguration key")
	}
	// Parse YAML line-by-line to find clusterName: <value>
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "clusterName:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "clusterName:"))
			if len(val) > 0 {
				return val, nil
			}
		}
	}
	return "", defaultErrors.New("clusterName not found in kubeadm ClusterConfiguration")
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

// getAgentContainerName returns the name of the main Datadog agent container
// in the pod, excluding known sidecar containers (trace-agent, process-agent,
// system-probe, security-agent, etc.). Falls back to the first container if
// no clear main agent is identified.
func getAgentContainerName(pod corev1.Pod) string {
	sidecars := map[string]bool{
		"trace-agent":    true,
		"process-agent":  true,
		"system-probe":   true,
		"security-agent": true,
		"secconfig":      true,
		"jmx":            true,
	}
	for _, c := range pod.Spec.Containers {
		if !sidecars[c.Name] {
			return c.Name
		}
	}
	if len(pod.Spec.Containers) > 0 {
		return pod.Spec.Containers[0].Name
	}
	return ""
}

// geDatadogInfos return datadog infos
func (m *ManagerOperator) geDatadogInfos(namespace string) (dto.VendorInfo, error) {
	var vendorInfos dto.VendorInfo
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return dto.VendorInfo{}, err
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return dto.VendorInfo{}, err
	}
	var podFound corev1.Pod
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.ObjectMeta.Name, "datadog-agent") {
			podFound = pod
			break
		}
	}
	// read the Datadog cluster UUID from ConfigMap datadog-cluster-id
	clusterID := ""
	if cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "datadog-cluster-id", metav1.GetOptions{}); err == nil {
		if id, ok := cm.Data["id"]; ok && len(id) > 0 {
			clusterID = id
		}
	}

	if podFound.ObjectMeta.Name != "" {
		containerName := getAgentContainerName(podFound)
		if containerName == "" {
			return dto.VendorInfo{}, defaultErrors.New("no container found in datadog-agent pod")
		}
		out, err := m.execInPod(namespace, podFound.ObjectMeta.Name, containerName, []string{"agent", "status"})
		if err != nil {
			return dto.VendorInfo{}, err
		}
		output := utils.RemoveLinesByPrefix([]string{"ERROR", "Error"}, out)
		datadogInfos := dto.DatadogInfos{
			ClusterID:                     clusterID,
			ClusterName:                   utils.ParseValueByPrefix(output, "cluster-name:"),
			HostId:                        utils.ParseValueByPrefix(output, "hostId:"),
			Hostname:                      utils.ParseValueByPrefix(output, "hostname:"),
			KernelArch:                    utils.ParseValueByPrefix(output, "kernelArch:"),
			KernelVersion:                 utils.ParseValueByPrefix(output, "kernelVersion:"),
			Os:                            utils.ParseValueByPrefix(output, "os:"),
			Platform:                      utils.ParseValueByPrefix(output, "platform:"),
			PlatformFamily:                utils.ParseValueByPrefix(output, "platformFamily:"),
			PlatformVersion:               utils.ParseValueByPrefix(output, "platformVersion:"),
			AgentVersion:                  utils.ParseValueByPrefix(output, "agent_version:"),
			Flavor:                        utils.ParseValueByPrefix(output, "flavor:"),
			InfrastructureMode:            utils.ParseValueByPrefix(output, "infrastructure_mode:"),
			InstallMethodInstallerVersion: utils.ParseValueByPrefix(output, "install_method_installer_version:"),
			InstallMethodTool:             utils.ParseValueByPrefix(output, "install_method_tool:"),
			InstallMethodToolVersion:      utils.ParseValueByPrefix(output, "install_method_tool_version:"),
		}
		vendorInfos.Datadog = datadogInfos
	}

	return vendorInfos, nil
}

// execInPod execute command in pod and return output
func (m *ManagerOperator) execInPod(namespace, pod, container string, command []string) (string, error) {

	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return "", err
	}

	req := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer

	exec, err := remotecommand.NewSPDYExecutor(
		m.kubeClient.Config,
		"POST",
		req.URL(),
	)
	if err != nil {
		return "", err
	}

	err = exec.StreamWithContext(
		context.Background(),
		remotecommand.StreamOptions{
			Stdout: &stdout,
			Stderr: &stderr,
		},
	)

	if err != nil {
		return "", fmt.Errorf(
			"exec error: %w | stderr: %s",
			err, stderr.String(),
		)
	}

	if stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf("%s", stderr.String())
	}

	return stdout.String(), nil
}
