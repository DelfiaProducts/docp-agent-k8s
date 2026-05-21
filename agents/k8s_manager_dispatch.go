package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

// executeSignal execute signal
func (k *K8sManager) executeSignal(signal dto.K8sSignal, datadogNamespace string) error {
	if err := k.operator.ExecuteSignal(&dto.FactorySignalDTO{
		Signal:                     &signal,
		Namespace:                  k.namespace,
		ConfigMapConfigurationName: k.configMapConfigurationsName,
		DatadogNamespace:           datadogNamespace,
	}); err != nil {
		return err
	}
	return nil
}

func (k *K8sManager) executeAction(action dto.K8sAction) error {
	transaction := utils.NewTransactionStatus()
	ctx := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)

	factory := dto.FactorySignalDTO{ConfigMapConfigurationName: k.configMapConfigurationsName, Namespace: k.namespace}
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		k.logger.Error("execute action get config map", "error", err.Error())
		return err
	}
	datadogNamespace := configMapConfiguration.Data["datadog_namespace"]
	newMode := action.Envs["mode"]
	switch action.Action {
	case "install":
		if newMode == "helm" {
			datadogDto := dto.DatadogDTO{
				Content:          action.Content,
				DatadogNamespace: datadogNamespace,
				Version:          action.Version,
				ApiKey:           action.Envs["apiKey"],
				HostTags:         action.HostTags,
			}
			go k.operator.NotifyStatus("install_datadog_received", pkg.TransactionEventOpen, "install datadog received", ctx, &factory)
			// execute cleaning last instalation datadog
			if err := k.cleaningDatadogLastInstalation(); err != nil {
				go k.operator.NotifyStatus("install_datadog_error", pkg.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
			// execute install datadog with helm mode
			if err := k.installDatadogWithHelm(datadogDto); err != nil {
				go k.operator.NotifyStatus("install_datadog_error", pkg.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
			go k.operator.NotifyStatus("install_datadog_update", pkg.TransactionEventUpdate, "install datadog update", ctx, &factory)
			configMapConfiguration.Data["datadog_mode"] = "helm"

		} else if newMode == "operator" {
			datadogDto := dto.DatadogDTO{
				Content:          action.Content,
				DatadogNamespace: datadogNamespace,
				Version:          action.Version,
				ApiKey:           action.Envs["apiKey"],
				HostTags:         action.HostTags,
			}
			go k.operator.NotifyStatus("install_datadog_received", pkg.TransactionEventOpen, "install datadog received", ctx, &factory)
			// execute cleaning last instalation datadog
			if err := k.cleaningDatadogLastInstalation(); err != nil {
				go k.operator.NotifyStatus("install_datadog_error", pkg.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
			// execute install datadog with operator mode
			if err := k.installDatadogWithOperator(datadogDto); err != nil {
				go k.operator.NotifyStatus("install_datadog_error", pkg.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
				return err
			}

			go k.operator.NotifyStatus("install_datadog_update", pkg.TransactionEventUpdate, "install datadog update", ctx, &factory)
			configMapConfiguration.Data["datadog_mode"] = "operator"
		}
		if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
			go k.operator.NotifyStatus("install_datadog_error", pkg.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
			return err
		}
		go k.operator.NotifyStatus("install_datadog_completed", pkg.TransactionEventClose, "install datadog update", ctx, &factory)
	case "uninstall":
		if action.Envs["mode"] == "helm" {
			datadogDto := dto.DatadogDTO{
				DatadogNamespace: datadogNamespace,
			}
			go k.operator.NotifyStatus("uninstall_datadog_received", pkg.TransactionEventOpen, "uninstall datadog received", ctx, &factory)
			if err := k.uninstallDatadogWithHelm(datadogDto); err != nil {
				go k.operator.NotifyStatus("uninstall_datadog_error", pkg.TransactionEventClose, fmt.Sprintf("uninstall datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
		} else if action.Envs["mode"] == "operator" {
			datadogDto := dto.DatadogDTO{
				DatadogNamespace: datadogNamespace,
			}
			go k.operator.NotifyStatus("uninstall_datadog_received", pkg.TransactionEventOpen, "uninstall datadog received", ctx, &factory)
			if err := k.uninstallDatadogWithOperator(datadogDto); err != nil {
				go k.operator.NotifyStatus("uninstall_datadog_error", pkg.TransactionEventClose, fmt.Sprintf("uninstall datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
		}
		configMapConfiguration.Data["datadog_mode"] = ""
		configMapConfiguration.Data["datadog_hash"] = ""
		configMapConfiguration.Data["datadog_installed"] = "false"
		if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
			return err
		}
		go k.operator.NotifyStatus("uninstall_datadog_completed", pkg.TransactionEventClose, "uninstall datadog update", ctx, &factory)
	}
	return nil
}

// executeAuthCall execute call for auth service
func (k *K8sManager) executeAuthCall() error {
	k.logger.Debug("execute auth call", "timestamp", time.Now())
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return err
	}
	payload := dto.K8sAuthPayload{}
	if apiKey, ok := configMapConfiguration.Data["api_key"]; ok {
		payload.ApiKey = apiKey
	}
	if computeId, ok := configMapConfiguration.Data["compute_id"]; ok {
		payload.ComputeId = computeId
	}
	resp, statusCode, err := k.operator.ExecuteAuthCall(payload)
	if err != nil {
		return err
	}
	var authResponse dto.K8sAuthResponse
	switch statusCode {
	case 200:
		if err := json.Unmarshal(resp, &authResponse); err != nil {
			return err
		}
		configMapConfiguration.Data["access_token"] = authResponse.AccessToken
		if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
			return err
		}
	}
	return nil
}

func (k *K8sManager) applyState() error {
	//get state config map
	configMapState, err := k.getConfigMap(k.configMapStateName, k.namespace)
	if err != nil {
		return err
	}
	dataStr := configMapState.Data["received"]

	//get configurations config map
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return err
	}

	if len(dataStr) > 0 {
		transaction := utils.NewTransactionStatus()
		ctx := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)

		factory := dto.FactorySignalDTO{ConfigMapConfigurationName: k.configMapConfigurationsName, Namespace: k.namespace}
		signals, err := k.getSignal([]byte(dataStr))
		if err != nil {
			return err
		}
		for _, signalState := range signals {
			k.logger.Debug("apply state", "signalState", signalState)
			if err := k.validateState(signalState); err != nil {
				return err
			}
			if len(signalState.TypeSignal) > 0 {

				validate, ok := configMapConfiguration.Data["validate"]
				if ok {
					k.logger.Debug("apply state validate", "validate", validate)
					if validate == "true" {
						configMapState.Data["current"] = dataStr
					}

					if err := k.updateConfigMap(k.configMapStateName, k.namespace, configMapState.Data); err != nil {
						return err
					}
				}
				datadogHash := configMapConfiguration.Data["datadog_hash"]
				datadogNamespace := configMapConfiguration.Data["datadog_namespace"]

				datadogInstalled, err := k.operator.VerifyDatadogResourceExists("datadog", datadogNamespace)
				if err != nil {
					return err
				}
				//validate if mode datadog is being altered
				if signalState.TypeSignal == "update_vendor" {
					configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
					if err != nil {
						k.logger.Error("execute apply state get config map", "error", err.Error())
						return err
					}
					actualMode := configMapConfiguration.Data["datadog_mode"]
					newMode := signalState.Vendor.Mode
					validDatadogMode := k.validateDatadogMode(newMode)
					if !validDatadogMode {
						k.logger.Error("execute apply state invalid datadog mode", "mode", newMode)
						return utils.ErrInvalidDatadogMode()
					}
					if datadogInstalled && len(actualMode) > 0 && newMode != actualMode {
						k.logger.Info("execute apply state mode change", "actualMode", actualMode, "newMode", newMode)
						if err := k.operator.UninstallDatadogCall(actualMode, dto.DatadogDTO{DatadogNamespace: datadogNamespace}); err != nil {
							k.logger.Error("execute apply state uninstall datadog", "error", err.Error())
							return err
						}
						configMapConfiguration.Data["datadog_hash"] = ""
						if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
							return err
						}
						for {
							time.Sleep(1 * time.Second)
							namespaces, err := k.operator.GetNamespaces()
							if err != nil {
								return err
							}
							vendor, err := k.operator.DatadogAlreadyInstalled("datadog", namespaces)
							if err != nil {
								return err
							}
							if !vendor.Installed {
								break
							}
							k.logger.Debug("waiting for datadog uninstall", "namespaces", namespaces)
						}
					}
				}

				//update version orya agent
				if signalState.TypeSignal == "update_agent" {
					k.logger.Debug("save new version for agent on config map configuration", "version", signalState.Version)
					configMapConfiguration.Data["version"] = signalState.Version
					if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
						return err
					}
				}
				datadogInstalled, err = k.operator.VerifyDatadogResourceExists("datadog", datadogNamespace)
				if err != nil {
					return err
				}
				k.logger.Debug("execute apply state datadog installed", "installed", datadogInstalled, "datadogHash", datadogHash, "action", signalState.Action.Action)
				if !datadogInstalled && signalState.Action.Action == "install" || datadogInstalled && signalState.Action.Action == "uninstall" {
					if err := k.executeAction(signalState.Action); err != nil {
						return err
					}
					newDatadogHash := utils.GenerateDatadogHash(signalState.Action.Content, signalState.Action.HostTags)
					configMapConfiguration.Data["datadog_hash"] = newDatadogHash
					if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
						return err
					}
				}
				//signal standby
				if signalState.TypeSignal == "standby" {
					k.logger.Debug("signal standby", "signal", signalState)
					go k.operator.NotifyStatus("orya_standby_received", pkg.TransactionEventOpen, "orya standby received", ctx, &factory)
					if err := k.handlerStandbyOryaAgent(time.Duration(signalState.Sleep) * time.Minute); err != nil {
						go k.operator.NotifyStatus("orya_standby_error", pkg.TransactionEventClose, "orya standby error", ctx, &factory)
						return err
					}
					time.Sleep(k.delay)
					go k.operator.NotifyStatus("orya_standby_complete", pkg.TransactionEventClose, "orya standby complete", ctx, &factory)
					continue
				}

				if err := k.executeSignal(signalState, datadogNamespace); err != nil {
					return err
				}
			}
		}

	}
	return nil
}

// periodicAutoUpdate periodic execute auto update
func (k *K8sManager) periodicAutoUdpate() error {
	defer k.wg.Done()

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			k.logger.Debug("periodic execute auto update", "timestamp", time.Now())
			if err := k.setAutoUpdateRunning(); err != nil {
				k.logger.Error("failed to set auto update running", "error", err.Error())
			}

			if err := k.AutoUpdateOrya(); err != nil {
				k.logger.Error("failed to execute auto update", "error", err.Error())
			}
			if err := k.AutoUpdateDatadog(); err != nil {
				k.logger.Error("failed to execute auto update datadog", "error", err.Error())
			}

		case <-k.done:
			close(k.done)
			return nil
		}
	}
}

// periodicSendMetadata execute periodic send metadata
func (k *K8sManager) periodicSendMetadata() error {
	defer k.wg.Done()

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			k.logger.Debug("periodic send metadata", "timestamp", time.Now())
			if err := k.handlerRegister(); err != nil {
				k.logger.Error("send metadata", "error", err.Error())
			}
		case <-k.done:
			close(k.done)
			return nil
		}
	}
}

// periodicValidateVendor execute periodic validate vendor
func (k *K8sManager) periodicValidateVendor() error {
	defer k.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			k.logger.Debug("periodic validate vendor", "timestamp", time.Now())
			if err := k.validateDatadogInstalled(); err != nil {
				k.logger.Error("periodic validate vendor", "error", err.Error())
			}
		case <-k.done:
			close(k.done)
			return nil
		}
	}
}

// periodicCollect execute periodic collect
func (k *K8sManager) periodicCollect() error {
	defer k.wg.Done()

	defer k.tickerSignal.Stop()

	for {
		select {
		case <-k.tickerSignal.C:
			if err := k.handlerStateCheck(); err != nil {
				k.logger.Error("handler state check", "error", err.Error())
			}
		case <-k.done:
			close(k.done)
			return nil
		}
	}
}

// periodicExecute periodic execution
func (k *K8sManager) periodicExecute() error {
	defer k.wg.Done()

	ticker := time.NewTicker(40 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			k.logger.Debug("periodic execute", "timestamp", time.Now())
			notAlreadyApplied, err := k.isSignalNotAlreadyApplied()
			if err != nil {
				k.logger.Error("check if signal is not already applied", "error", err.Error())
			}
			if notAlreadyApplied {
				if err := k.applyState(); err != nil {
					k.logger.Error("apply state", "error", err.Error())
				}
			} else {
				k.logger.Debug("signal already exists", "timestamp", time.Now())
			}
		case <-k.done:
			close(k.done)
			return nil
		}
	}
}
