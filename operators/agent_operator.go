package operators

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/utils"

	"helm.sh/helm/v3/pkg/action"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		namespace:                   utils.GetOryaNamespace(),
		kubeClient:                  utils.NewKubeClient(),
		helmClient:                  utils.NewHelmClient(logger),
		configMapConfigurationsName: utils.GetOryaConfigMapConfigurationsName(),
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
