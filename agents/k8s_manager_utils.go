package agents

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/utils"
	corev1 "k8s.io/api/core/v1"
)

// populateVersion populate version the manager
func (k *K8sManager) populateVersion() error {
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return err
	}
	k.version = configMapConfiguration.Data["version"]
	return nil
}

// initializeConfigMaps execute creation the config maps initials
func (k *K8sManager) initializeConfigMaps() error {
	dataState := map[string]string{
		"received": "",
		"current":  "",
	}
	if err := k.createConfigMap(k.configMapStateName, k.namespace, dataState); err != nil {
		return err
	}
	dataConfigurations := map[string]string{
		"datadog_installed":   "false",
		"datadog_mode":        "",
		"datadog_version":     "",
		"datadog_namespace":   "",
		"registered":          "false",
		"auto_update_running": "false",
		"signal_hash":         "",
		"version":             os.Getenv("VERSION"),
		"api_key":             os.Getenv("ORYA_API_KEY"),
		"tags":                os.Getenv("TAGS"),
		"cluster_name":        "",
	}
	if err := k.createConfigMap(k.configMapConfigurationsName, k.namespace, dataConfigurations); err != nil {
		return err
	}
	return nil
}

// prepareTags execute prepare tags for send register
func (k *K8sManager) prepareTags(tagsEnv string) []string {
	var arr []string
	if len(tagsEnv) > 0 {
		arr = strings.Split(tagsEnv, ",")
	}
	return arr
}

// getSignalHash returns the hash of the received signal
func (k *K8sManager) getSignalHash() (string, error) {
	configMapState, err := k.getConfigMap(k.configMapStateName, k.namespace)
	if err != nil {
		return "", err
	}
	received := configMapState.Data["received"]
	return utils.GenerateHashMd5([]byte(received)), nil
}

// getSignal return signal for apply
func (k *K8sManager) getSignal(data []byte) ([]dto.K8sSignal, error) {
	var k8sConfig dto.K8sConfig
	var signals []dto.K8sSignal
	var action dto.K8sAction
	if err := json.Unmarshal(data, &k8sConfig); err != nil {
		return nil, err
	}
	k8sSignal := k8sConfig.Signal
	signalType := k8sSignal.TypeSignal
	k.logger.Debug("get signal", "signal", k8sSignal)
	if signalType == "update" {
		if k8sSignal.Agents.DatadogAgent.Enabled {
			if len(k8sSignal.Agents.DatadogAgent.Version) > 0 {
				signal := dto.K8sSignal{}
				signal.TypeSignal = "update_vendor"
				signal.Vendor.Name = "datadog"
				signal.Vendor.Mode = k8sSignal.Agents.DatadogAgent.Mode
				signal.Vendor.Content = k8sSignal.Agents.DatadogAgent.DeployYml
				signal.Vendor.Version = k8sSignal.Agents.DatadogAgent.Version
				signal.Vendor.HostTags = k8sSignal.HostTags
				datadogAgentMode := k8sSignal.Agents.DatadogAgent.Mode
				if datadogAgentMode == "helm" {
					action = dto.K8sAction{
						Action:   "install",
						Provider: "datadog",
						Content:  k8sSignal.Agents.DatadogAgent.DeployYml,
						Version:  k8sSignal.Agents.DatadogAgent.Version,
						HostTags: k8sSignal.HostTags,
						Envs: map[string]string{
							"apiKey":  k8sSignal.Agents.DatadogAgent.ApiKey,
							"mode":    "helm",
							"version": k8sSignal.Agents.DatadogAgent.Version,
						},
					}
					signal.Action = action
				} else if datadogAgentMode == "operator" {
					action = dto.K8sAction{
						Action:   "install",
						Provider: "datadog",
						Content:  k8sSignal.Agents.DatadogAgent.DeployYml,
						Version:  k8sSignal.Agents.DatadogAgent.Version,
						HostTags: k8sSignal.HostTags,
						Envs: map[string]string{
							"apiKey":  k8sSignal.Agents.DatadogAgent.ApiKey,
							"mode":    "operator",
							"version": k8sSignal.Agents.DatadogAgent.Version,
						},
					}
					signal.Action = action

				}
				signals = append(signals, signal)

			}
		}

		if len(k8sSignal.Agents.OryaAgent.Version) > 0 {
			signal := dto.K8sSignal{}
			signal.TypeSignal = "update_agent"
			signal.Version = k8sSignal.Agents.OryaAgent.Version
			signals = append(signals, signal)
		}
	} else if signalType == "debug-session" {
		signal := dto.K8sSignal{}
		signal.TypeSignal = signalType
		signal.Duration = k8sConfig.Signal.Duration
		signals = append(signals, signal)
	} else if signalType == "uninstall" {
		signal := dto.K8sSignal{}
		signal.TypeSignal = signalType
		signal.RemoveOtherVendors = k8sConfig.Signal.RemoveOtherVendors
		signals = append(signals, signal)
	} else if signalType == "standby" {
		signal := dto.K8sSignal{}
		signal.TypeSignal = signalType
		signal.Mode = k8sConfig.Signal.Mode
		if signal.Mode == "stop" {
			signal.Sleep = 1
		}
		signal.Sleep = k8sConfig.Signal.Sleep
		signals = append(signals, signal)
	}
	return signals, nil
}

// getConfigMap execute get the config map
func (k *K8sManager) getConfigMap(name string, namespace string) (*corev1.ConfigMap, error) {
	configMap, err := k.operator.GetConfigMap(name, namespace)
	if err != nil {
		return nil, err
	}
	return configMap, nil
}

// createConfigMap execute creation the config map
func (k *K8sManager) createConfigMap(name string, namespace string, data map[string]string) error {
	if err := k.operator.CreateConfigMap(name, namespace, data); err != nil {
		return err
	}
	return nil
}

// updateConfigMap execute update the config map
func (k *K8sManager) updateConfigMap(name string, namespace string, data map[string]string) error {
	if err := k.operator.UpdateConfigMap(name, namespace, data); err != nil {
		return err
	}
	return nil
}

// cleaningDatadogLastInstalation execute cleanning the last instalation datadog
func (k *K8sManager) cleaningDatadogLastInstalation() error {
	if err := k.operator.CleanningDatadogLastInstalation(); err != nil {
		return err
	}
	return nil
}

// installDatadogWithHelm execute install datadog with helm
func (k *K8sManager) installDatadogWithHelm(datadogDto dto.DatadogDTO) error {
	exists, err := k.operator.VerifyDatadogResourceExists("datadog", datadogDto.Namespace)
	if err != nil {
		k.logger.Error("install datadog with helm", "error", err.Error())
		return err
	}
	k.logger.Debug("install datadog with helm", "exists", exists)
	if !exists {
		k.logger.Info("execute install datadog with helm", "timestamp", time.Now())
		if err := k.operator.InstallDatadogCall("helm", datadogDto); err != nil {
			return err
		}
	}
	return nil
}

// installDatadogWithOperator execute install datadog with operator
func (k *K8sManager) installDatadogWithOperator(datadogDto dto.DatadogDTO) error {
	exists, err := k.operator.VerifyDatadogResourceExists("datadog", datadogDto.Namespace)
	if err != nil {
		k.logger.Error("install datadog with operator", "error", err.Error())
		return err
	}
	k.logger.Debug("install datadog with operator", "exists", exists)
	if !exists {
		k.logger.Info("execute install datadog with operator", "timestamp", time.Now())
		if err := k.operator.InstallDatadogCall("operator", datadogDto); err != nil {
			return err
		}

	}
	return nil
}

// uninstallDatadogWithHelm execute uninstall datadog with helm
func (k *K8sManager) uninstallDatadogWithHelm(datadogDto dto.DatadogDTO) error {
	exists, err := k.operator.VerifyDatadogResourceExists("datadog", datadogDto.Namespace)
	if err != nil {
		k.logger.Error("uninstall datadog with helm", "error", err.Error())
		return err
	}
	k.logger.Debug("uninstall datadog with helm", "exists", exists)
	if exists {
		k.logger.Info("execute uninstall datadog with helm", "timestamp", time.Now())
		if err := k.operator.UninstallDatadogCall("helm", datadogDto); err != nil {
			return err
		}
	}
	return nil
}

// uninstallDatadogWithOperator execute uninstall datadog with operator
func (k *K8sManager) uninstallDatadogWithOperator(datadogDto dto.DatadogDTO) error {
	exists, err := k.operator.VerifyDatadogResourceExists("datadog", datadogDto.Namespace)
	if err != nil {
		k.logger.Error("uninstall datadog with operator", "error", err.Error())
		return err
	}
	k.logger.Debug("uninstall datadog with operator", "exists", exists)
	if exists {
		k.logger.Info("execute uninstall datadog with operator", "timestamp", time.Now())
		if err := k.operator.UninstallDatadogCall("operator", datadogDto); err != nil {
			return err
		}
	}
	return nil
}

// setAutoUpdateRunning configura o auto update como em execução
func (k *K8sManager) setAutoUpdateRunning() error {
	configuration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		k.logger.Error("auto update orya", "error", err.Error())
		return err
	}
	configuration.Data["auto_update_running"] = "true"
	if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configuration.Data); err != nil {
		k.logger.Error("auto update orya", "error", err.Error())
		return err
	}
	return nil
}

// handlerStandbyOryaAgent execute handler standby orya agent
func (k *K8sManager) handlerStandbyOryaAgent(sleep time.Duration) error {
	//protect for invalid value sleep
	if sleep <= time.Duration(0) {
		sleep = k.intervalGetSignal
	}
	k.tickerSignal.Reset(sleep)
	return nil
}
