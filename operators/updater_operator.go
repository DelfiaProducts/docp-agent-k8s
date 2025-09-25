package operators

import (
	"sync"

	"github.com/OryaHub/agent-k8s/utils"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
)

type UpdaterOperator struct {
	wg                          *sync.WaitGroup
	kubeClient                  *utils.KubeClient
	helmClient                  *utils.HelmClient
	configMapStateName          string
	configMapConfigurationsName string
	namespace                   string
	logger                      *utils.K8sLogger
}

func NewUpdaterOperator(logger *utils.K8sLogger) *UpdaterOperator {
	return &UpdaterOperator{
		namespace:                   utils.GetDocpNamespace(),
		configMapStateName:          utils.GetDocpConfiMapStateName(),
		configMapConfigurationsName: utils.GetDocpConfigMapConfigurationsName(),
		logger:                      logger,
		helmClient:                  utils.NewHelmClient(logger),
		kubeClient:                  utils.NewKubeClient(),
	}
}

// Setup configure operator
func (uo *UpdaterOperator) Setup() error {
	//configure helm client
	if err := uo.helmClient.Setup(); err != nil {
		return err
	}
	//configure kube client
	if err := uo.kubeClient.LoadConfigKube(); err != nil {
		return err
	}

	return nil
}

// ValidateDeploymentSuccess valida se o último deployment teve sucesso
func (uo *UpdaterOperator) ValidateDeploymentSuccess(namespace, deploymentName string) (bool, error) {
	return uo.helmClient.ValidateDeploymentSuccess(namespace, deploymentName)
}

// ValidateAllDocpDeploymentsSuccess valida se todos os deployments do DOCP tiveram sucesso
func (uo *UpdaterOperator) ValidateAllDocpDeploymentsSuccess(namespace string) (bool, error) {
	return uo.helmClient.ValidateAllDocpDeploymentsSuccess(namespace)
}

// GetLatestHelmVersion busca a versão mais atual do chart Helm do DOCP
func (uo *UpdaterOperator) GetLatestHelmVersion(namespace, releaseName string) (string, error) {
	latestVersion, err := uo.helmClient.GetLatestHelmVersion(namespace, releaseName)
	if err != nil {
		uo.logger.Error("failed to get latest helm version", "error", err.Error())
		return "", err
	}

	return latestVersion, nil
}

// GetReleaseByVersion busca a versão do release do DOCP
func (uo *UpdaterOperator) GetReleaseByVersion(namespace, releaseName, version string) (*release.Release, error) {
	latestRelease, err := uo.helmClient.GetReleaseByVersion(namespace, releaseName, version)
	if err != nil {
		uo.logger.Error("failed to get latest release", "error", err.Error())
		return nil, err
	}

	return latestRelease, nil
}

// UpgradeDocpHelmChart executa o upgrade do chart Helm do DOCP
func (uo *UpdaterOperator) UpgradeDocpHelmChart(namespace, releaseName, repositoryURL, targetVersion string, values map[string]interface{}) error {
	if err := uo.helmClient.UpgradeDocpHelmChart(namespace, releaseName, repositoryURL, targetVersion, values); err != nil {
		uo.logger.Error("failed to upgrade helm chart", "error", err.Error())
		return err
	}

	return nil
}

// UpgradeDocpToLatestVersion busca a versão mais atual e executa o upgrade do DOCP
func (m *UpdaterOperator) UpgradeDocpToLatestVersion(namespace, releaseName, repositoryURL string, values map[string]interface{}) error {
	m.logger.Debug("upgrading DOCP to latest version", "namespace", namespace, "releaseName", releaseName)

	if err := m.helmClient.UpgradeDocpToLatestVersion(namespace, releaseName, repositoryURL, values); err != nil {
		m.logger.Error("failed to upgrade DOCP to latest version", "error", err.Error())
		return err
	}

	return nil
}

// RollbackDocpHelmChart executa o rollback do chart Helm do DOCP para a versão anterior
func (m *UpdaterOperator) RollbackDocpHelmChart(namespace, releaseName string) error {
	if err := m.helmClient.RollbackDocpHelmChart(namespace, releaseName); err != nil {
		m.logger.Error("failed to rollback DOCP helm chart", "error", err.Error())
		return err
	}

	return nil
}

// ValidateAndRollbackIfNeeded valida o deployment e executa rollback se necessário
func (m *UpdaterOperator) ValidateAndRollbackIfNeeded(namespace, releaseName string, maxRetries int) error {
	return m.helmClient.ValidateAndRollbackIfNeeded(namespace, releaseName, maxRetries)
}

// UpgradeDocpWithRollbackProtection executa upgrade com proteção de rollback automático
func (m *UpdaterOperator) UpgradeDocpWithRollbackProtection(namespace, releaseName, repositoryURL, targetVersion string) error {
	m.logger.Debug("upgrading DOCP with rollback protection",
		"namespace", namespace,
		"releaseName", releaseName)
	upgradeVersion := targetVersion

	if err := m.helmClient.UpdateChartRepository(utils.GetHelmRepositoryName(), repositoryURL); err != nil {
		m.logger.Error("failed to update chart repository", "error", err.Error())
		return err
	}

	newValues, err := m.helmClient.GetChartValuesByVersion(upgradeVersion)
	if err != nil {
		m.logger.Error("failed to get latest helm chart values", "error", err.Error())
		return err
	}
	m.logger.Debug("latest release values retrieved successfully", "newValues", newValues)

	// Executar o upgrade
	err = m.UpgradeDocpHelmChart(namespace, releaseName, repositoryURL, upgradeVersion, newValues)
	if err != nil {
		m.logger.Error("upgrade failed", "error", err.Error())
		return err
	}

	// Validar e fazer rollback se necessário (máximo 3 tentativas)
	err = m.ValidateAndRollbackIfNeeded(namespace, releaseName, 3)
	if err != nil {
		m.logger.Error("validation and rollback process failed", "error", err.Error())
		return err
	}

	m.logger.Debug("upgrade with rollback protection completed successfully",
		"finalVersion", upgradeVersion)
	return nil
}

// GetConfigMap retrieves a config map by name and namespace
func (uo *UpdaterOperator) GetConfigMap(configMapName string, namespace string) (*corev1.ConfigMap, error) {
	return uo.kubeClient.GetConfigMap(configMapName, namespace)
}

// UpdateConfigMap updates a config map by name and namespace
func (uo *UpdaterOperator) UpdateConfigMap(configMapName string, namespace string, data map[string]string) error {
	return uo.kubeClient.UpdateConfigMap(configMapName, namespace, data)
}

// Start execute running the goroutines operator
func (uo *UpdaterOperator) Start() error {
	uo.logger.Debug("start updater operator")
	return nil
}
