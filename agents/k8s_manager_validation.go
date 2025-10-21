package agents

import (
	"encoding/json"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/utils"
)

// validateDatadogInstalled execute validate if datadog
// is installed
func (k *K8sManager) validateDatadogInstalled() error {
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return err
	}
	namespaces, err := k.operator.GetNamespaces()
	if err != nil {
		return err
	}
	vendor, err := k.operator.DatadogAlreadyInstalled("datadog", namespaces)
	if err != nil {
		return err
	}
	if vendor.Installed {
		configMapConfiguration.Data["datadog_installed"] = "true"
		configMapConfiguration.Data["datadog_namespace"] = vendor.Namespace
		configMapConfiguration.Data["datadog_mode"] = vendor.Mode
	} else {
		configMapConfiguration.Data["datadog_namespace"] = k.namespace
		configMapConfiguration.Data["datadog_mode"] = ""
	}
	if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
		return err
	}
	return nil
}

// isSignalNotAlreadyApplied checks if a signal is new or duplicated.
func (k *K8sManager) isSignalNotAlreadyApplied() (bool, error) {
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return false, err
	}

	newSignalHash, err := k.getSignalHash()
	if err != nil {
		return false, err
	}

	lastSignalHash := configMapConfiguration.Data["signal_hash"]
	k.logger.Debug("check if signal is not already applied", "lastSignalHash", lastSignalHash, "newSignalHash", newSignalHash)
	// If the signal is new, allow notification and update hash
	if lastSignalHash != newSignalHash {
		configMapConfiguration.Data["signal_hash"] = newSignalHash
		if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// validateDatadogMode validate datadog mode
func (k *K8sManager) validateDatadogMode(mode string) bool {
	switch mode {
	case "helm", "operator":
		return true
	default:
		return false
	}
}

// validateState execute validation the state
func (k *K8sManager) validateState(signal dto.K8sSignal) error {
	var k8sConfig dto.K8sConfig
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return err
	}
	datadogNamespace := configMapConfiguration.Data["datadog_namespace"]
	datadogExists, err := k.operator.VerifyDatadogResourceExists("datadog", datadogNamespace)
	if err != nil {
		return err
	}
	configMapState, err := k.getConfigMap(k.configMapStateName, k.namespace)
	if err != nil {
		return err
	}
	received, ok := configMapState.Data["received"]
	if ok {
		if err := json.Unmarshal([]byte(received), &k8sConfig); err != nil {
			return err
		}
		if datadogExists {
			// datadog already installed
			if configMapConfiguration.Data["datadog_mode"] == k8sConfig.Signal.Agents.DatadogAgent.Mode {
				// mode ok
				contentSignal := signal.Vendor.Content
				contentReceived := k8sConfig.Signal.Agents.DatadogAgent.DeployYml
				equalsContent := utils.CompareValuesWithMd5Hash([]byte(contentSignal), []byte(contentReceived))
				if equalsContent {
					// equals contents
					configMapConfiguration.Data["validate"] = "true"
				} else {
					configMapConfiguration.Data["validate"] = "false"
				}
			} else {
				configMapConfiguration.Data["validate"] = "false"
			}
		}
		if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
			return err
		}
	}

	return nil
}
