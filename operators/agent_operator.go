package operators

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DelfiaProducts/docp-agent-k8s/dto"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// AgentOperator is struct for agent operator
type AgentOperator struct {
	logger                      *utils.K8sLogger
	kubeClient                  *utils.KubeClient
	helmClient                  *utils.HelmClient
	namespace                   string
	configMapConfigurationsName string
}

// NewAgentOperator return instance of agent operator
func NewAgentOperator(logger *utils.K8sLogger) *AgentOperator {
	return &AgentOperator{
		logger:                      logger,
		namespace:                   utils.GetDocpNamespace(),
		kubeClient:                  utils.NewKubeClient(),
		helmClient:                  utils.NewHelmClient(logger),
		configMapConfigurationsName: utils.GetDocpConfigMapConfigurationsName(),
	}
}

// Setup configure operator
func (a *AgentOperator) Setup() error {
	if err := a.kubeClient.LoadConfigKube(); err != nil {
		return err
	}

	if err := a.helmClient.Setup(); err != nil {
		return err
	}

	return nil
}

// GetConfigMapConfiguration execute get the config map
func (a *AgentOperator) GetConfigMapConfiguration() (*corev1.ConfigMap, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(a.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(a.namespace).Get(ctx, a.configMapConfigurationsName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return configMap, nil
}

func (a *AgentOperator) applyDatadogOperatorYml(datadogDto dto.DatadogDTO) error {
	clientset, err := dynamic.NewForConfig(a.kubeClient.Config)
	if err != nil {
		return err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(
		strings.NewReader(datadogDto.Content), 4096)

	var datadogAgent unstructured.Unstructured
	if err := decoder.Decode(&datadogAgent); err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "datadoghq.com",
		Version:  "v2alpha1",
		Resource: "datadogagents", // Plural do `kind` em lowercase
	}
	_, err = clientset.Resource(gvr).Namespace(datadogDto.DatadogNamespace).Get(context.Background(), datadogAgent.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return err
		}
	}
	_, err = clientset.Resource(gvr).Namespace(datadogDto.DatadogNamespace).Apply(context.Background(), datadogAgent.GetName(), &datadogAgent, metav1.ApplyOptions{
		FieldManager: "docp-operator-datadog-install",
		Force:        true,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// updateHelmChartOperator execute update chart datadog with operator mode
func (a *AgentOperator) updateHelmChartOperator(datadogDto dto.DatadogDTO) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), datadogDto.DatadogNamespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}
	var version string
	if datadogDto.Version == "latest" {
		versionChart, err := a.helmClient.GetLatestHelmChartVersion("datadog", "datadog-operator")
		if err != nil {
			return err
		}
		version = versionChart
	} else {
		version = datadogDto.Version
	}
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = datadogDto.DatadogNamespace
	upgrade.Version = version
	upgrade.Timeout = 5 * time.Minute
	upgrade.ResetValues = false
	upgrade.ReuseValues = true
	upgrade.DisableHooks = true

	chartPath, err := upgrade.ChartPathOptions.LocateChart("datadog/datadog-operator", a.helmClient.Settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(datadogDto.Content), &values)
	if err != nil {
		return err
	}

	_, err = upgrade.Run("datadog-operator", chart, values)
	if err != nil {
		return err
	}
	return nil
}

// applyDatadogUpdateConfigOperator execute update config datadog with operator mode
func (a *AgentOperator) applyDatadogUpdateConfigOperator(datadogDto dto.DatadogDTO) error {
	clientset, err := dynamic.NewForConfig(a.kubeClient.Config)
	if err != nil {
		return err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(
		strings.NewReader(datadogDto.Content), 4096)

	var datadogAgent unstructured.Unstructured
	if err := decoder.Decode(&datadogAgent); err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "datadoghq.com",
		Version:  "v2alpha1",
		Resource: "datadogagents", // Plural do `kind` em lowercase
	}

	resource, err := clientset.Resource(gvr).Namespace(datadogDto.DatadogNamespace).Get(context.Background(), datadogAgent.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("not found\n")
			return nil
		}
	}
	datadogAgent.SetResourceVersion(resource.GetResourceVersion())
	_, errUpdate := clientset.Resource(gvr).Namespace(datadogDto.DatadogNamespace).Update(context.Background(), &datadogAgent, metav1.UpdateOptions{})
	if errUpdate != nil {
		return err
	}
	return nil
}

// applyDatadogUpdateConfigHelm execute update config datadog with helm mode
func (a *AgentOperator) applyDatadogUpdateConfigHelm(datadogDto dto.DatadogDTO) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), datadogDto.DatadogNamespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}
	var version string
	if datadogDto.Version == "latest" {
		versionChart, err := a.helmClient.GetLatestHelmChartVersion("datadog", "datadog")
		if err != nil {
			return err
		}
		version = versionChart
	} else {
		version = datadogDto.Version
	}
	namespace := ""
	if len(datadogDto.DatadogNamespace) > 0 {
		namespace = datadogDto.DatadogNamespace
	} else {
		namespace = datadogDto.Namespace
	}
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = namespace
	upgrade.Install = true
	upgrade.Atomic = true
	upgrade.Version = version
	upgrade.Timeout = 5 * time.Minute
	upgrade.ResetValues = false
	upgrade.ReuseValues = true
	upgrade.DisableHooks = true

	chartPath, err := upgrade.ChartPathOptions.LocateChart("datadog/datadog", a.helmClient.Settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(datadogDto.Content), &values)
	if err != nil {
		return err
	}

	_, err = upgrade.Run("datadog-agent", chart, values)
	if err != nil {
		return err
	}
	return nil
}

// createOrUpdateSecret execute creation or update the secret
func (a *AgentOperator) createOrUpdateSecret(datadogDto dto.DatadogDTO, secretName string) error {
	clientset, err := kubernetes.NewForConfig(a.kubeClient.Config)
	if err != nil {
		a.logger.Error("erro ao criar clientset", "error", err.Error())
		return err
	}
	apiKey := datadogDto.ApiKey
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: datadogDto.DatadogNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"api-key": apiKey,
		},
	}

	_, err = clientset.CoreV1().Secrets(datadogDto.DatadogNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			_, err = clientset.CoreV1().Secrets(datadogDto.DatadogNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update existing secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return err
}

func (a *AgentOperator) installHelmChart(releaseName string, datadogDto dto.DatadogDTO) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), datadogDto.DatadogNamespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}
	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = datadogDto.DatadogNamespace
	install.CreateNamespace = false
	install.Timeout = 1 * time.Minute

	var installVersion string
	if len(datadogDto.Version) > 0 && strings.Contains(datadogDto.Version, "latest") {
		latestVersion, err := a.helmClient.GetLatestHelmChartVersion("datadog", "datadog")
		if err != nil {
			return err
		}
		installVersion = latestVersion
	} else {
		installVersion = datadogDto.Version
	}
	a.logger.Debug("install version", "installVersion", installVersion)

	install.Version = installVersion

	chartPath, err := install.ChartPathOptions.LocateChart("datadog/datadog", a.helmClient.Settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(datadogDto.Content), &values)
	if err != nil {
		return err
	}

	_, err = install.Run(chart, values)
	return err
}

func (a *AgentOperator) installHelmChartOperatorDatadog(releaseName string, datadogDto dto.DatadogDTO) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), datadogDto.DatadogNamespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}

	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = datadogDto.DatadogNamespace
	install.CreateNamespace = false
	install.Atomic = true
	install.Timeout = 2 * time.Minute

	if len(datadogDto.Version) > 0 && strings.Contains(datadogDto.Version, "latest") {
		versionChart, err := a.helmClient.GetLatestHelmChartVersion("datadog", "datadog-operator")
		if err != nil {
			return err
		}
		install.Version = versionChart
	} else {
		install.Version = datadogDto.Version
	}
	chartPath, err := install.ChartPathOptions.LocateChart("datadog/datadog-operator", a.helmClient.Settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(datadogDto.Content), &values)
	if err != nil {
		return err
	}

	_, err = install.Run(chart, values)
	return err
}

// ValidateAllDatadogDeploymentSuccess verifica se a implantação foi bem-sucedida
func (a *AgentOperator) ValidateAllDatadogDeploymentSuccess(mode string, datadogDto dto.DatadogDTO) (bool, error) {
	var namespace string
	if len(datadogDto.DatadogNamespace) > 0 {
		namespace = datadogDto.DatadogNamespace
	} else {
		namespace = datadogDto.Namespace
	}

	return a.helmClient.ValidateAllDatadogDeploymentsSuccess(mode, namespace)
}

// InstallDatadogHelm exeucte install datadog with helm chart
func (a *AgentOperator) InstallDatadogHelm(datadogDto dto.DatadogDTO) error {
	if err := a.UpdateChartRepository("datadog", utils.GetDatadogHelmRepository()); err != nil {
		a.logger.Error("error add helm repo", "error", err.Error())
		return err
	}
	if err := a.createOrUpdateSecret(datadogDto, "datadog-secret"); err != nil {
		a.logger.Error("error create secret", "error", err.Error())
		return err
	}
	if err := a.installHelmChart("datadog-agent", datadogDto); err != nil {
		a.logger.Error("error install helm chart", "error", err.Error())
		return err
	}
	return nil
}

// UninstallDatadogHelm execute uninstall datadog with helm chart
func (a *AgentOperator) UninstallDatadogHelm(releaseName, namespace string) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}

	uninstall := action.NewUninstall(actionConfig)
	uninstall.IgnoreNotFound = true
	_, err := uninstall.Run(releaseName)
	if err != nil {
		return err
	}
	return nil
}

// InstallDatadogOperator execute install datadog with operator
func (a *AgentOperator) InstallDatadogOperator(datadogDto dto.DatadogDTO) error {
	if err := a.UpdateChartRepository("datadog", utils.GetDatadogHelmRepository()); err != nil {
		a.logger.Error("error add helm repo", "error", err.Error())
		return err
	}
	if err := a.createOrUpdateSecret(datadogDto, "datadog-secret"); err != nil {
		a.logger.Error("error create secret", "error", err.Error())
		return err
	}
	if err := a.installHelmChartOperatorDatadog("datadog-operator", datadogDto); err != nil {
		a.logger.Error("error install helm chart operator", "error", err.Error())
		return err
	}
	time.Sleep(time.Second * 10)
	if err := a.applyDatadogOperatorYml(datadogDto); err != nil {
		a.logger.Error("error apply datadog operator yml", "error", err.Error())
		return err
	}
	return nil
}

// UninstallDatadogOperator execute uninstall datadog with operator
func (a *AgentOperator) UninstallDatadogOperator(resourceName, releaseName, namespace string) error {
	dynamClientset, err := dynamic.NewForConfig(a.kubeClient.Config)
	if err != nil {
		return err
	}
	ctx := context.Background()

	// Defina o GroupVersionResource (GVR) para o DatadogAgent CRD.
	gvr := schema.GroupVersionResource{
		Group:    "datadoghq.com", // O grupo do CRD
		Version:  "v2alpha1",      // A versão do CRD
		Resource: "datadogagents", // O plural do `kind` (DatadogAgent -> datadogagents)
	}

	resource, err := dynamClientset.Resource(gvr).Namespace(namespace).Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
	}

	finalizers := resource.GetFinalizers()
	newFinalizers := []string{}
	for _, f := range finalizers {
		if f != "finalizer.agent.datadoghq.com" {
			newFinalizers = append(newFinalizers, f)
		}
	}
	resource.SetFinalizers(newFinalizers)

	_, err = dynamClientset.Resource(gvr).Namespace(namespace).Update(ctx, resource, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: new(int64), // Set to 0
		PropagationPolicy: func() *metav1.DeletionPropagation {
			policy := metav1.DeletePropagationForeground
			return &policy
		}(),
	}
	errDel := dynamClientset.Resource(gvr).Namespace(resource.GetNamespace()).Delete(ctx, resource.GetName(), deleteOptions)
	if errDel != nil {
		return err
	}
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}

	uninstall := action.NewUninstall(actionConfig)
	uninstall.Timeout = 5 * time.Minute
	uninstall.Wait = true
	uninstall.IgnoreNotFound = true
	_, err = uninstall.Run(releaseName)
	if err != nil {
		return err
	}

	return nil
}

// ValidateAndRollbackDatadogIfNeeded valida o deployment datadog e executa rollback se necessário
func (a *AgentOperator) ValidateAndRollbackDatadogIfNeeded(mode, namespace, releaseName string, maxRetries int) error {
	return a.helmClient.ValidateAndRollbackDatadogIfNeeded(mode, namespace, releaseName, maxRetries)
}

// UpdateChartRepository updates or add a Helm chart repository
func (a *AgentOperator) UpdateChartRepository(repositoryName, repositoryURL string) error {
	return a.helmClient.UpdateChartRepository(repositoryName, repositoryURL)
}

// UpdateDatadogConfigOperator execute update datadog configurations with operator mode
func (a *AgentOperator) UpdateDatadogConfigOperator(datadogDto dto.DatadogDTO) error {
	if err := a.UpdateChartRepository("datadog", utils.GetDatadogHelmRepository()); err != nil {
		a.logger.Error("error apply datadog update config add helm repo", "error", err.Error())
		return err
	}
	if err := a.updateHelmChartOperator(datadogDto); err != nil {
		a.logger.Error("error update helm chart operator", "error", err.Error())
		return err
	}

	if err := a.applyDatadogUpdateConfigOperator(datadogDto); err != nil {
		a.logger.Error("error apply datadog update config", "error", err.Error())
		return err
	}

	var namespace string
	if len(datadogDto.DatadogNamespace) > 0 {
		namespace = datadogDto.DatadogNamespace
	} else {
		namespace = datadogDto.Namespace
	}

	releaseName, err := a.helmClient.GetReleaseName(namespace, "datadog-operator")
	if err != nil {
		a.logger.Error("error getting release name", "error", err.Error())
		return err
	}

	if err := a.ValidateAndRollbackDatadogIfNeeded("operator", namespace, releaseName, 3); err != nil {
		a.logger.Error("error validate and rollback datadog if needed", "error", err.Error())
		return err
	}
	return nil
}

// UpdateDatadogConfigHelm execute update datadog configurations with helm mode
func (a *AgentOperator) UpdateDatadogConfigHelm(datadogDto dto.DatadogDTO) error {
	if err := a.UpdateChartRepository("datadog", utils.GetDatadogHelmRepository()); err != nil {
		a.logger.Error("error apply datadog update config add helm repo", "error", err.Error())
		return err
	}
	if err := a.applyDatadogUpdateConfigHelm(datadogDto); err != nil {
		a.logger.Error("error apply datadog update config", "error", err.Error())
		return err
	}
	var namespace string
	if len(datadogDto.DatadogNamespace) > 0 {
		namespace = datadogDto.DatadogNamespace
	} else {
		namespace = datadogDto.Namespace
	}
	releaseName, err := a.helmClient.GetReleaseName(namespace, "datadog")
	if err != nil {
		a.logger.Error("error getting release name", "error", err.Error())
		return err
	}

	if err := a.ValidateAndRollbackDatadogIfNeeded("helm", namespace, releaseName, 3); err != nil {
		a.logger.Error("error validate and rollback datadog if needed", "error", err.Error())
		return err
	}
	return nil
}
