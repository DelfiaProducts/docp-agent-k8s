package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/internal"
	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/templates"
	"github.com/OryaHub/agent-k8s/utils"

	corev1 "k8s.io/api/core/v1"
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
		"datadog_namespace":   "",
		"registered":          "false",
		"auto_update_running": "false",
		"signal_hash":         "",
		"version":             os.Getenv("VERSION"),
		"api_key":             os.Getenv("ORYA_API_KEY"),
		"tags":                os.Getenv("TAGS"),
	}
	if err := k.createConfigMap(k.configMapConfigurationsName, k.namespace, dataConfigurations); err != nil {
		return err
	}
	return nil
}

// retryHandlerRegister execute retry the create initial data
func (k *K8sManager) retryHandlerRegister() error {
	k.logger.Debug("retry handler register", "timestamp", time.Now())
	k.retryRegister += 1
	time.Sleep(time.Minute * time.Duration(k.retryRegister))
	if err := k.handlerRegister(); err != nil {
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

// handlerRegister execute send metadata for register service
func (k *K8sManager) handlerRegister() error {
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return err
	}
	registered, ok := configMapConfiguration.Data["registered"]
	if !ok {
		configMapConfiguration.Data["registered"] = "false"
		if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
			return err
		}
	}
	metadata, err := k.operator.CollectMetadataK8s()
	if err != nil {
		return err
	}
	var tagsEnv string
	tagsFromConfig, ok := configMapConfiguration.Data["tags"]
	if ok {
		tagsEnv = tagsFromConfig
	} else {
		tagsEnv = os.Getenv("TAGS")
	}
	metadata.Tags = k.prepareTags(tagsEnv)
	apiKey, ok := configMapConfiguration.Data["api_key"]
	if ok && len(apiKey) > 0 {
		var response dto.K8sRegisterResponse
		if registered == "false" {
			resp, statusCode, err := k.operator.ExecuteRegisterCall("create", metadata, apiKey, "")
			if err != nil {
				if k.retryRegister <= k.maxRetry {
					go k.retryHandlerRegister()
				}
				return err
			}
			k.logger.Debug("handler metadata create", "statusCode", statusCode, "resp", string(resp))
			switch statusCode {
			case 202:
				if err := json.Unmarshal(resp, &response); err != nil {
					return err
				}
				// add more data from response the register
				if len(response.AccessToken) > 0 {
					configMapConfiguration.Data["access_token"] = response.AccessToken
					// decode jwt
					claims, err := utils.DecodeJwt(response.AccessToken)
					if err != nil {
						return err
					}
					configMapConfiguration.Data["org_id"] = strconv.Itoa(claims.OryaOrgId)
					configMapConfiguration.Data["compute_id"] = claims.ComputeId
				}
				configMapConfiguration.Data["registered"] = "true"
				if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
					return err
				}
			default:
				k.logger.Debug("response from register create", "statusCode", statusCode, "resp", string(resp))
				if k.retryRegister <= k.maxRetry {
					go k.retryHandlerRegister()
				}
			}
		} else {
			if token, ok := configMapConfiguration.Data["access_token"]; ok {
				resp, statusCode, err := k.operator.ExecuteRegisterCall("update", metadata, apiKey, token)
				// validar status code pra salvar dados no config map de configurations
				if err != nil {
					return err
				}
				if len(response.AccessToken) > 0 {
					configMapConfiguration.Data["access_token"] = response.AccessToken
				}
				if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
					return err
				}
				k.logger.Debug("response from register update", "statusCode", statusCode, "resp", string(resp))
				switch statusCode {
				case 202:
					if err := json.Unmarshal(resp, &response); err != nil {
						return err
					}
					if len(response.AccessToken) > 0 {
						configMapConfiguration.Data["access_token"] = response.AccessToken
					}
					if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
						return err
					}
				case 401:
					if err := k.executeAuthCall(); err != nil {
						return err
					}
					if k.retryRegister <= k.maxRetry {
						go k.retryHandlerRegister()
					}
				}
				k.logger.Debug("handler metadata update", "statusCode", statusCode, "resp", string(resp))
			}
		}
	} else {
		k.logger.Info("handler register", "error", utils.ErrorOryaApiKeyNotFound().Error())
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

// handlerStateCheck execute handler the state check
func (k *K8sManager) handlerStateCheck() error {
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		k.logger.Error("handler state check get config map configuration", "error", err.Error())
		return err
	}
	registered, ok := configMapConfiguration.Data["registered"]
	if ok && registered == "true" {
		var state dto.K8sConfig
		payload := dto.K8sStateCheckPayload{}
		accessToken, ok := configMapConfiguration.Data["access_token"]
		if !ok {
			payloadAuth := dto.K8sAuthPayload{}
			if apiKey, okAuth := configMapConfiguration.Data["api_key"]; okAuth {
				payloadAuth.ApiKey = apiKey
			}
			if computeId, okAuth := configMapConfiguration.Data["compute_id"]; okAuth {
				payloadAuth.ComputeId = computeId
			}

			respAuth, statusCodeAuth, err := k.operator.ExecuteAuthCall(payloadAuth)
			if err != nil {
				return err
			}
			switch statusCodeAuth {
			case 200:
				var authResponse dto.K8sAuthResponse
				if err := json.Unmarshal(respAuth, &authResponse); err != nil {
					return err
				}
				configMapConfiguration.Data["access_token"] = authResponse.AccessToken
				if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
					return err
				}
			}
		}
		isLockedEvents := k.operator.IsLockedEvents()
		k.logger.Debug("handler state check", "isLockedEvents", isLockedEvents)
		autoUpdateRunning := configMapConfiguration.Data["auto_update_running"]
		k.logger.Debug("handler state check", "autoUpdateRunning", autoUpdateRunning)
		if autoUpdateRunning == "true" {
			k.logger.Debug("handler state check", "autoUpdateRunning", "enabled")
			return nil
		}

		if !isLockedEvents {
			resp, statusCode, err := k.operator.StateCheckCall(payload, accessToken)
			if err != nil {
				k.logger.Error("handler state check", "error", err.Error())
				return err
			}
			switch statusCode {
			case 200:
				if err := json.Unmarshal(resp, &state); err != nil {
					k.logger.Error("unmarshal response state check", "error", err.Error())
					return err
				}
				configMapState, err := k.getConfigMap(k.configMapStateName, k.namespace)
				if err != nil {
					k.logger.Error("get config state for handler state check", "error", err.Error())
					return err
				}
				stateBytes, err := json.Marshal(&state)
				if err != nil {
					k.logger.Error("marshall state for save on config map", "error", err.Error())
					return err
				}
				//validate duplicated signal
				newHash := utils.GenerateHashMd5(stateBytes)
				lastSignalHash, err := k.getSignalHash()
				if err != nil {
					k.logger.Error("get last signal hash on handler state check", "error", err.Error())
					return err
				}
				//if new signal hash is equal to last signal hash notify duplicated
				if newHash == lastSignalHash {
					k.logger.Debug("handler state check", "status", "duplicated signal")
					transaction := utils.NewTransactionStatus()
					ctxTransaction := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)
					factorySignal := dto.FactorySignalDTO{
						Namespace:                  k.namespace,
						ConfigMapConfigurationName: k.configMapConfigurationsName,
					}
					//notify status already exists signal
					go k.operator.NotifyStatus("update_signal_received", internal.TransactionEventOpen, "update signal received", ctxTransaction, &factorySignal)
					time.Sleep(k.delay)
					go k.operator.NotifyStatus("update_signal_completed", internal.TransactionEventClose, "update signal already exists", ctxTransaction, &factorySignal)

					return nil
				}
				configMapState.Data["received"] = string(stateBytes)
				if err := k.updateConfigMap(k.configMapStateName, k.namespace, configMapState.Data); err != nil {
					k.logger.Error("update config map on handler state check", "error", err.Error())
					return err
				}
				k.logger.Debug("updated config map state with resp from state check", "timestamp", time.Now())
			case 204:
				k.logger.Debug("handler state check", "status", "not state present")
				return nil
			case 403:
				k.logger.Debug("handler state check", "status", "not authorized")
				if err := k.executeAuthCall(); err != nil {
					return err
				}
			}
		} else {
			k.logger.Debug("handler state check locked", "isLockedEvents", isLockedEvents)
		}

	}

	return nil
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
		signals, err := k.getSignal([]byte(dataStr))
		if err != nil {
			return err
		}
		for _, signalState := range signals {
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
					newDatadogHash := utils.GenerateHashMd5([]byte(signalState.Action.Content))
					configMapConfiguration.Data["datadog_hash"] = newDatadogHash
					if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
						return err
					}
				}

				if err := k.executeSignal(signalState, datadogNamespace); err != nil {
					return err
				}
			}
		}

	}
	return nil
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
		if len(k8sSignal.Agents.DatadogAgent.Version) > 0 {
			if k8sSignal.Agents.DatadogAgent.Enabled {
				signal := dto.K8sSignal{}
				signal.TypeSignal = "update_vendor"
				signal.Vendor.Name = "datadog"
				signal.Vendor.Mode = k8sSignal.Agents.DatadogAgent.Mode
				signal.Vendor.Content = k8sSignal.Agents.DatadogAgent.DeployYml
				signal.Vendor.Version = k8sSignal.Agents.DatadogAgent.Version
				datadogAgentMode := k8sSignal.Agents.DatadogAgent.Mode
				if datadogAgentMode == "helm" {
					action = dto.K8sAction{
						Action:   "install",
						Provider: "datadog",
						Content:  k8sSignal.Agents.DatadogAgent.DeployYml,
						Version:  k8sSignal.Agents.DatadogAgent.Version,
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
	}
	return signals, nil
}

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

// validateDatadogMode validate datadog mode
func (k *K8sManager) validateDatadogMode(mode string) bool {
	switch mode {
	case "helm", "operator":
		return true
	default:
		return false
	}
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
			}
			go k.operator.NotifyStatus("install_datadog_received", internal.TransactionEventOpen, "install datadog received", ctx, &factory)
			// execute cleaning last instalation datadog
			if err := k.cleaningDatadogLastInstalation(); err != nil {
				go k.operator.NotifyStatus("install_datadog_error", internal.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
			// execute install datadog with helm mode
			if err := k.installDatadogWithHelm(datadogDto); err != nil {
				go k.operator.NotifyStatus("install_datadog_error", internal.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
			go k.operator.NotifyStatus("install_datadog_update", internal.TransactionEventUpdate, "install datadog update", ctx, &factory)
			configMapConfiguration.Data["datadog_mode"] = "helm"

		} else if newMode == "operator" {
			datadogDto := dto.DatadogDTO{
				Content:          action.Content,
				DatadogNamespace: datadogNamespace,
				Version:          action.Version,
				ApiKey:           action.Envs["apiKey"],
			}
			go k.operator.NotifyStatus("install_datadog_received", internal.TransactionEventOpen, "install datadog received", ctx, &factory)
			// execute cleaning last instalation datadog
			if err := k.cleaningDatadogLastInstalation(); err != nil {
				go k.operator.NotifyStatus("install_datadog_error", internal.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
			// execute install datadog with operator mode
			if err := k.installDatadogWithOperator(datadogDto); err != nil {
				go k.operator.NotifyStatus("install_datadog_error", internal.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
				return err
			}

			go k.operator.NotifyStatus("install_datadog_update", internal.TransactionEventUpdate, "install datadog update", ctx, &factory)
			configMapConfiguration.Data["datadog_mode"] = "operator"
		}
		if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
			go k.operator.NotifyStatus("install_datadog_error", internal.TransactionEventClose, fmt.Sprintf("install datadog error: %s", err.Error()), ctx, &factory)
			return err
		}
		go k.operator.NotifyStatus("install_datadog_completed", internal.TransactionEventClose, "install datadog update", ctx, &factory)
	case "uninstall":
		if action.Envs["mode"] == "helm" {
			datadogDto := dto.DatadogDTO{
				DatadogNamespace: datadogNamespace,
			}
			go k.operator.NotifyStatus("uninstall_datadog_received", internal.TransactionEventOpen, "uninstall datadog received", ctx, &factory)
			if err := k.uninstallDatadogWithHelm(datadogDto); err != nil {
				go k.operator.NotifyStatus("uninstall_datadog_error", internal.TransactionEventClose, fmt.Sprintf("uninstall datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
		} else if action.Envs["mode"] == "operator" {
			datadogDto := dto.DatadogDTO{
				DatadogNamespace: datadogNamespace,
			}
			go k.operator.NotifyStatus("uninstall_datadog_received", internal.TransactionEventOpen, "uninstall datadog received", ctx, &factory)
			if err := k.uninstallDatadogWithOperator(datadogDto); err != nil {
				go k.operator.NotifyStatus("uninstall_datadog_error", internal.TransactionEventClose, fmt.Sprintf("uninstall datadog error: %s", err.Error()), ctx, &factory)
				return err
			}
		}
		configMapConfiguration.Data["datadog_mode"] = ""
		configMapConfiguration.Data["datadog_hash"] = ""
		configMapConfiguration.Data["datadog_installed"] = "false"
		if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
			return err
		}
		go k.operator.NotifyStatus("uninstall_datadog_completed", internal.TransactionEventClose, "uninstall datadog update", ctx, &factory)
	}
	return nil
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
				go k.operator.NotifyStatus("auto_update_orya_received", internal.TransactionEventUpdate, "update orya received", ctx, factorySignal)

				//update repository

				go k.operator.NotifyStatus("auto_update_orya_completed", internal.TransactionEventClose, "update orya completed", ctx, factorySignal)
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
				go k.operator.NotifyStatus("auto_update_orya_received", internal.TransactionEventUpdate, "update orya received", ctx, factorySignal)

				//update repository
				job := templates.TemplateJobAutoUpdateHelmRelease(k.namespace, utils.GetOryaReleaseName(), utils.GetHelmRepository(), latestChartVersion, utils.GetOryaUpdaterRepositoryName(latestChartVersion))
				if err := k.operator.CreateJob(k.namespace, &job); err != nil {
					k.logger.Error("failed to create job", "error", err.Error())
					go k.operator.NotifyStatus("auto_update_orya_error", internal.TransactionEventClose, "failed update orya", ctx, factorySignal)
					return err
				}
				go k.operator.NotifyStatus("auto_update_orya_completed", internal.TransactionEventClose, "update orya completed", ctx, factorySignal)
			}
		}
	}

	k.logger.Debug("auto update orya", "configState", configState)
	return nil
}

func (k *K8sManager) state(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	go k.handlerStateCheck()
	fmt.Fprintf(w, "OK")
}

func (k *K8sManager) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
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

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
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
