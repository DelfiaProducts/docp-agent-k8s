package operators

import (
	"sync"

	"github.com/OryaHub/agent-k8s/utils"
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
		namespace:                   utils.GetOryaNamespace(),
		configMapStateName:          utils.GetOryaConfiMapStateName(),
		configMapConfigurationsName: utils.GetOryaConfigMapConfigurationsName(),
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

// UpgradeOryaHelmChart executa o upgrade do chart Helm do DOCP
func (uo *UpdaterOperator) UpgradeOryaHelmChart(namespace, releaseName, repositoryURL, targetVersion string, values map[string]interface{}) error {
	if err := uo.helmClient.UpgradeOryaHelmChart(namespace, releaseName, repositoryURL, targetVersion, values); err != nil {
		uo.logger.Error("failed to upgrade helm chart", "error", err.Error())
		return err
	}

	return nil
}

// UpgradeOryaToLatestVersion busca a versão mais atual e executa o upgrade do DOCP
func (m *UpdaterOperator) UpgradeOryaToLatestVersion(namespace, releaseName, repositoryURL string, values map[string]interface{}) error {
	m.logger.Debug("upgrading DOCP to latest version", "namespace", namespace, "releaseName", releaseName)

	if err := m.helmClient.UpgradeOryaToLatestVersion(namespace, releaseName, repositoryURL, values); err != nil {
		m.logger.Error("failed to upgrade DOCP to latest version", "error", err.Error())
		return err
	}

	return nil
}

// RollbackOryaHelmChart executa o rollback do chart Helm do DOCP para a versão anterior
func (m *UpdaterOperator) RollbackOryaHelmChart(namespace, releaseName string) error {
	if err := m.helmClient.RollbackOryaHelmChart(namespace, releaseName); err != nil {
		m.logger.Error("failed to rollback DOCP helm chart", "error", err.Error())
		return err
	}

	return nil
}

// UpgradeOryaWithRollbackProtection executa upgrade com proteção de rollback automático
func (m *UpdaterOperator) UpgradeOryaWithRollbackProtection(namespace, releaseName, repositoryURL, targetVersion string) error {
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
	err = m.UpgradeOryaHelmChart(namespace, releaseName, repositoryURL, upgradeVersion, newValues)
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

// UpdateConfigMap updates a config map by name and namespace
func (uo *UpdaterOperator) UpdateConfigMap(configMapName string, namespace string, data map[string]string) error {
	return uo.kubeClient.UpdateConfigMap(configMapName, namespace, data)
}

// Start execute running the goroutines operator
func (uo *UpdaterOperator) Start() error {
	uo.logger.Debug("start updater operator")
	return nil
}
