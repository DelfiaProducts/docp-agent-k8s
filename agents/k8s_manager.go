package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/templates"
	"github.com/OryaHub/agent-k8s/utils"
)

// K8sManager is struct for manager the k8s
type K8sManager struct {
	logger                      *utils.K8sLogger
	wg                          *sync.WaitGroup
	done                        chan struct{}
	operator                    *operators.ManagerOperator
	configMapStateName          string
	configMapConfigurationsName string
	namespace                   string
	retryRegister               int
	maxRetry                    int
	version                     string
	delay                       time.Duration
}

// NewK8sManager return instance of k8s manager
func NewK8sManager(logger *utils.K8sLogger) *K8sManager {
	return &K8sManager{
		logger:                      logger,
		wg:                          &sync.WaitGroup{},
		done:                        make(chan struct{}),
		namespace:                   utils.GetOryaNamespace(),
		configMapStateName:          utils.GetOryaConfiMapStateName(),
		configMapConfigurationsName: utils.GetOryaConfigMapConfigurationsName(),
		retryRegister:               0,
		maxRetry:                    10,
		delay:                       time.Second * 1,
	}
}

// Initialize execute initialization the manager
func (k *K8sManager) Initialize() error {
	operator := operators.NewManagerOperator(k.logger)
	if err := operator.Setup(); err != nil {
		k.logger.Error("initialize new manager operator", "error", err.Error())
		return err
	}
	k.operator = operator
	if err := k.initializeConfigMaps(); err != nil {
		k.logger.Error("initialize config maps", "error", err.Error())
		return err
	}
	if err := k.handlerRegister(); err != nil {
		k.logger.Error("initialize handler register", "error", err.Error())
	}
	if err := k.operator.Start(); err != nil {
		k.logger.Error("initialize operator start", "error", err.Error())
		return err
	}
	if err := k.validateDatadogInstalled(); err != nil {
		k.logger.Error("initialize validate datadog", "error", err.Error())
		return err
	}
	if err := k.populateVersion(); err != nil {
		k.logger.Error("initialize populate version", "error", err.Error())
		return err
	}
	return nil
}

// AutoUpdateDatadog execute auto update the datadog
func (k *K8sManager) AutoUpdateDatadog() error {
	//get config state
	configState, err := k.getConfigMap(k.configMapStateName, k.namespace)
	if err != nil {
		k.logger.Error("auto update orya", "error", err.Error())
		return err
	}
	configurations, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		k.logger.Error("auto update orya", "error", err.Error())
		return err
	}
	//get received
	received, ok := configState.Data["received"]
	if ok {
		var k8sConfig dto.K8sConfig
		if err := json.Unmarshal([]byte(received), &k8sConfig); err != nil {
			return err
		}
		if k8sConfig.Signal.TypeSignal == "update" {
			datadog := k8sConfig.Signal.Agents.DatadogAgent
			version := datadog.Version
			if version == "latest" {
				//update agent
				k.logger.Debug("execute auto update version datadog", "timestamp", time.Now())
				//update repository
				if err := k.operator.UpdateChartRepository(); err != nil {
					k.logger.Error("failed to update helm repository", "error", err.Error())
					return err
				}
				//get release mode datadog
				releaseMode, err := k.operator.GetReleaseModeDatadog(k.namespace)
				if err != nil {
					k.logger.Error("failed to get release mode datadog", "error", err.Error())
					return err
				}
				k.logger.Debug("release mode datadog", "releaseMode", releaseMode)
				var chartName string
				switch releaseMode {
				case "helm":
					chartName = "datadog"
				case "operator":
					chartName = "datadog-operator"
				}
				//get release name the datadog
				releaseName, err := k.operator.GetReleaseName(k.namespace, chartName)
				if err != nil {
					k.logger.Error("failed to get release name datadog", "error", err.Error())
					return err
				}
				k.logger.Debug("release name datadog", "releaseName", releaseName)
				//validate if already updated
				currentChartVersion, err := k.operator.GetCurrentHelmChartVersion(k.namespace, releaseName)
				if err != nil {
					k.logger.Error("failed to get current helm chart version", "error", err.Error())
					return err
				}

				latestChartVersion, err := k.operator.GetLatestHelmChartVersion(k.namespace, releaseName)
				if err != nil {
					k.logger.Error("failed to get latest helm chart version", "error", err.Error())
					return err
				}

				k.logger.Debug("current helm chart version", "version", currentChartVersion)
				k.logger.Debug("latest helm chart version", "version", latestChartVersion)

				updated := k.operator.ValidateVersionHelmAlreadyUpdated(currentChartVersion, latestChartVersion)
				k.logger.Debug("validate if helm chart version is already updated", "updated", updated)
				if updated {
					k.logger.Debug("version is already updated", "version", version)
					return nil
				}
				content := datadog.DeployYml
				apiKey := datadog.ApiKey
				appKey := datadog.AppKey
				datadogNamespace := configurations.Data["datadog_namespace"]

				datadogDto := dto.DatadogDTO{
					Content:          content,
					DatadogNamespace: datadogNamespace,
					Namespace:        k.namespace,
					ApiKey:           apiKey,
					AppKey:           appKey,
					Version:          version,
				}

				//apply update the datadog
				if err := k.operator.UpdateDatadogConfigurations(releaseMode, datadogDto); err != nil {
					k.logger.Error("failed to update datadog configurations", "error", err.Error())
					return err
				}

				//create transaction
				transaction := utils.NewTransactionStatus()
				factorySignal := &dto.FactorySignalDTO{
					Namespace:                  k.namespace,
					ConfigMapConfigurationName: k.configMapConfigurationsName,
				}
				ctx := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)
				go k.operator.NotifyStatus("auto_update_orya_received", pkg.TransactionEventUpdate, "update orya received", ctx, factorySignal)

				//update repository

				go k.operator.NotifyStatus("auto_update_orya_completed", pkg.TransactionEventClose, "update orya completed", ctx, factorySignal)
			}
		}
	}
	k.logger.Debug("auto update datadog", "configState", configState)

	return nil
}

// AutoUpdateOrya execute auto update the orya
func (k *K8sManager) AutoUpdateOrya() error {
	//execute auto update
	configState, err := k.getConfigMap(k.configMapStateName, k.namespace)
	if err != nil {
		k.logger.Error("auto update orya", "error", err.Error())
		return err
	}
	//get received
	received, ok := configState.Data["received"]
	if ok {
		var k8sConfig dto.K8sConfig
		if err := json.Unmarshal([]byte(received), &k8sConfig); err != nil {
			return err
		}
		if k8sConfig.Signal.TypeSignal == "update" {
			agent := k8sConfig.Signal.Agents.OryaAgent
			version := agent.Version
			if version == "latest" {
				//update agent
				k.logger.Debug("execute auto update version agent", "timestamp", time.Now())
				//update repository
				if err := k.operator.UpdateChartRepository(); err != nil {
					k.logger.Error("failed to update helm repository", "error", err.Error())
					return err
				}
				//validate if already updated
				currentChartVersion, err := k.operator.GetCurrentHelmChartVersion(k.namespace, utils.GetOryaReleaseName())
				if err != nil {
					k.logger.Error("failed to get current helm chart version", "error", err.Error())
					return err
				}

				latestChartVersion, err := k.operator.GetLatestHelmChartVersion(k.namespace, utils.GetOryaReleaseName())
				if err != nil {
					k.logger.Error("failed to get latest helm chart version", "error", err.Error())
					return err
				}

				k.logger.Debug("current helm chart version", "version", currentChartVersion)
				k.logger.Debug("latest helm chart version", "version", latestChartVersion)

				updated := k.operator.ValidateVersionHelmAlreadyUpdated(currentChartVersion, latestChartVersion)
				k.logger.Debug("validate if helm chart version is already updated", "updated", updated)
				if updated {
					k.logger.Debug("version is already updated", "version", version)
					return nil
				}

				//create transaction
				transaction := utils.NewTransactionStatus()
				factorySignal := &dto.FactorySignalDTO{
					Namespace:                  k.namespace,
					ConfigMapConfigurationName: k.configMapConfigurationsName,
				}
				ctx := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)
				go k.operator.NotifyStatus("auto_update_orya_received", pkg.TransactionEventUpdate, "update orya received", ctx, factorySignal)

				//update repository
				job := templates.TemplateJobAutoUpdateHelmRelease(k.namespace, utils.GetOryaReleaseName(), utils.GetHelmRepository(), latestChartVersion, utils.GetOryaUpdaterRepositoryName(latestChartVersion))
				if err := k.operator.CreateJob(k.namespace, &job); err != nil {
					k.logger.Error("failed to create job", "error", err.Error())
					go k.operator.NotifyStatus("auto_update_orya_error", pkg.TransactionEventClose, "failed update orya", ctx, factorySignal)
					return err
				}
				go k.operator.NotifyStatus("auto_update_orya_completed", pkg.TransactionEventClose, "update orya completed", ctx, factorySignal)
			}
		}
	}

	k.logger.Debug("auto update orya", "configState", configState)
	return nil
}

// Listen execute lintening the api
func (k *K8sManager) Listen() error {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", k.health)
	mux.HandleFunc("/state", k.state)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

// Start execute running the manager
func (k *K8sManager) Start() error {
	k.logger.Info("Orya Manager Kubernetes Running")
	if err := k.Initialize(); err != nil {
		return err
	}
	k.wg.Add(5)
	go k.periodicSendMetadata()
	go k.periodicValidateVendor()
	go k.periodicCollect()
	go k.periodicExecute()
	go k.periodicAutoUdpate()
	if err := k.Listen(); err != nil {
		return err
	}
	return nil
}
