package pkg

import (
	"os"
	"sync"

	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/utils"
)

// K8sUpdater is struct for updating the k8s
type K8sUpdater struct {
	logger                      *utils.K8sLogger
	wg                          *sync.WaitGroup
	done                        chan struct{}
	operator                    *operators.UpdaterOperator
	configMapStateName          string
	configMapConfigurationsName string
	namespace                   string
	retryRegister               int
	maxRetry                    int
	version                     string
}

// NewK8sUpdater return instance of k8s updater
func NewK8sUpdater(logger *utils.K8sLogger) *K8sUpdater {
	return &K8sUpdater{
		logger:                      logger,
		wg:                          &sync.WaitGroup{},
		done:                        make(chan struct{}),
		namespace:                   utils.GetOryaNamespace(),
		configMapStateName:          utils.GetOryaConfiMapStateName(),
		configMapConfigurationsName: utils.GetOryaConfigMapConfigurationsName(),
		retryRegister:               0,
		maxRetry:                    10,
	}
}

// Initialize execute initialization the updater
func (k *K8sUpdater) Initialize() error {
	operator := operators.NewUpdaterOperator(k.logger)
	if err := operator.Setup(); err != nil {
		k.logger.Error("initialize new updater operator", "error", err.Error())
		return err
	}
	k.operator = operator
	return nil
}

// ExecuteUpdate execute update the k8s
func (k *K8sUpdater) ExecuteUpdate() error {
	namespace := os.Getenv("NAMESPACE")
	releaseName := os.Getenv("RELEASE_NAME")
	repositoryUrl := os.Getenv("REPOSITORY_URL")
	targetVersion := os.Getenv("TARGET_VERSION")
	k.logger.Debug("upgrading orya helm", "namespace", namespace, "releaseName", releaseName, "repositoryUrl", repositoryUrl, "targetVersion", targetVersion)
	if err := k.operator.UpgradeOryaWithRollbackProtection(namespace, releaseName, repositoryUrl, targetVersion); err != nil {
		k.logger.Error("failed to upgrade orya", "error", err.Error())
		return err
	}
	//get config map configurations
	configurations, err := k.operator.GetConfigMap(utils.GetOryaConfigMapConfigurationsName(), namespace)
	if err != nil {
		k.logger.Error("failed to get config map", "error", err.Error())
		return err
	}
	configurations.Data["auto_update_running"] = "false"
	configurations.Data["version"] = targetVersion

	if err := k.operator.UpdateConfigMap(utils.GetOryaConfigMapConfigurationsName(), namespace, configurations.Data); err != nil {
		k.logger.Error("failed to update config map", "error", err.Error())
		return err
	}

	return nil
}

// Start execute running the updater
func (k *K8sUpdater) Start() error {
	k.logger.Info("Orya Updater Kubernetes Running")
	if err := k.Initialize(); err != nil {
		return err
	}

	if err := k.ExecuteUpdate(); err != nil {
		return err
	}

	return nil
}
