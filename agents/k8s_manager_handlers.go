package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

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

// detectClusterChanges compares the stored cluster_name with the current cluster
// name from the cascade strategy. If different, it triggers handlerRegister to
// collect fresh metadata and send it to the backend.
func (k *K8sManager) detectClusterChanges() error {
	configMapConfiguration, err := k.getConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return fmt.Errorf("detect cluster changes: could not get config map: %w", err)
	}
	storedClusterName := configMapConfiguration.Data["cluster_name"]

	currentClusterName, err := k.operator.GetClusterName()
	if err != nil {
		return fmt.Errorf("detect cluster changes: could not get current cluster name: %w", err)
	}
	if len(currentClusterName) == 0 {
		return nil
	}

	if storedClusterName != currentClusterName {
		if err := k.handlerRegister(); err != nil {
			return fmt.Errorf("detect cluster changes: re-registration failed: %w", err)
		}
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
				if len(metadata.ClusterName) > 0 {
					configMapConfiguration.Data["cluster_name"] = metadata.ClusterName
				}
				if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
					return err
				}
			case 204:
				// compute já existe na base — marca como registered e salva
				// cluster_name sem tentar decodificar JWT (204 não tem body)
				configMapConfiguration.Data["registered"] = "true"
				if len(metadata.ClusterName) > 0 {
					configMapConfiguration.Data["cluster_name"] = metadata.ClusterName
				}
				if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
					return err
				}
			case 400:
				isRateLimit, err := k.operator.ValidateRateLimitInstallAgentError(resp)
				if err != nil {
					return err
				}
				if isRateLimit {
					k.logger.Debug("response from register create", "statusCode", statusCode, "resp", string(resp))
					if err := k.operator.AutoUninstall(k.namespace); err != nil {
						return err
					}
					if err := k.operator.RemoveOryaNamespace(k.namespace); err != nil {
						return err
					}
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
					if len(metadata.ClusterName) > 0 {
						configMapConfiguration.Data["cluster_name"] = metadata.ClusterName
					}
					if err := k.updateConfigMap(k.configMapConfigurationsName, k.namespace, configMapConfiguration.Data); err != nil {
						return err
					}
				case 204:
					// update sem mudanças — salva cluster_name localmente
					if len(metadata.ClusterName) > 0 {
						configMapConfiguration.Data["cluster_name"] = metadata.ClusterName
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
					go k.operator.NotifyStatus("update_signal_received", pkg.TransactionEventOpen, "update signal received", ctxTransaction, &factorySignal)
					time.Sleep(k.delay)
					go k.operator.NotifyStatus("update_signal_completed", pkg.TransactionEventClose, "update signal already exists", ctxTransaction, &factorySignal)

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
