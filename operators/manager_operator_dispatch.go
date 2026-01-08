package operators

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	pkg "github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

// NotifyStatus execute notify the status to state check
func (m *ManagerOperator) NotifyStatus(status string, typeEvent string, message string, ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	configMapConfiguration, err := m.GetConfigMap(factorySignal.ConfigMapConfigurationName, factorySignal.Namespace)
	if err != nil {
		m.logger.Error("notify status get config map", "error", err.Error())
		return err
	}
	//get tracer id from config map state
	configMapState, err := m.GetConfigMap(utils.GetOryaConfiMapStateName(), factorySignal.Namespace)
	if err != nil {
		m.logger.Error("notify status get config map state", "error", err.Error())
		return err
	}

	if accessToken, ok := configMapConfiguration.Data["access_token"]; ok {
		transactionStatus := utils.GetTransactionFromContext(ctx)
		if len(transactionStatus.ID) > 0 {
			transactionStatus.Status = status
			transactionStatus.Message = message
			transactionStatus.TypeEvent = typeEvent
			transactionStatus.UlidEvent = utils.GetUlid()
			//inject tracer id if exists
			if traceID, ok := configMapState.Data["trace_id"]; ok && traceID != "" {
				transactionStatus.TraceID = traceID
			}
			if m.LockedEvents {
				m.pendingTransactionEvents = append(m.pendingTransactionEvents, transactionStatus)
			} else {
				m.logger.Debug("notify status send status", "transactionStatus", transactionStatus)
				res, statusCode, err := m.StateCheckSendStatus(transactionStatus, accessToken)
				if err != nil {
					m.logger.Error("notify status send status", "error", err.Error())
					return err
				}
				if statusCode == 500 {
					m.LockedEvents = true
					m.pendingTransactionEvents = append(m.pendingTransactionEvents, transactionStatus)
				}
				m.logger.Debug("notify status", "timestamp", time.Now(), "response", string(res), "statusCode", statusCode)
			}
		}
	}

	return nil
}

// ExecuteSignal execute functions the signals
func (m *ManagerOperator) ExecuteSignal(factorySignal *dto.FactorySignalDTO) error {
	transaction := utils.NewTransactionStatus()
	ctx := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)
	fn, ok := m.mapFactorySignal[factorySignal.Signal.TypeSignal]
	if ok {
		return fn(ctx, factorySignal)
	} else {
		m.logger.Debug("signal not implemented")
		return nil
	}
}

// ExecuteRegisterCall execute call for service register
func (m *ManagerOperator) ExecuteRegisterCall(mode string, metadata dto.K8sRegister, apiKey string, token string) ([]byte, int, error) {
	transaction := utils.NewTransactionStatus()
	ctx := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)
	factoryDto := dto.FactorySignalDTO{
		Namespace:                  utils.GetOryaNamespace(),
		ConfigMapConfigurationName: utils.GetOryaConfigMapConfigurationsName(),
	}
	switch mode {
	case "create":
		resp, statusCode, err := m.registerService.RegisterCall("compute/v1/docp", metadata, apiKey, token, true)
		if err != nil {
			return nil, 0, err
		}
		return resp, statusCode, err
	case "update":
		go m.NotifyStatus("update_metadata", pkg.TransactionEventOpen, "update metadata", ctx, &factoryDto)
		resp, statusCode, err := m.registerService.RegisterCall("compute/v1/docp", metadata, apiKey, token, false)
		if err != nil {
			go m.NotifyStatus("update_metadata_error", pkg.TransactionEventClose, "error on update metadata", ctx, &factoryDto)
		}
		go m.NotifyStatus("update_metadata_completed", pkg.TransactionEventClose, "update metadata completed", ctx, &factoryDto)
		return resp, statusCode, err

	}
	return nil, 0, nil
}

// StateCheckCall execute call to state check service
func (m *ManagerOperator) StateCheckCall(payload dto.K8sStateCheckPayload, accessToken string) ([]byte, int, error) {
	return m.stateCheckService.GetState("compute/v1/status/info", payload, accessToken)
}

// StateCheckSendStatus execute call to state check and
// send status
func (m *ManagerOperator) StateCheckSendStatus(transactionStatus dto.TransactionStatus, accessToken string) ([]byte, int, error) {
	return m.stateCheckService.SendStatus("compute/transaction", transactionStatus, accessToken)
}

// ExecuteAuthCall execute call to auth
func (m *ManagerOperator) ExecuteAuthCall(payload dto.K8sAuthPayload) ([]byte, int, error) {
	return m.authService.AuthCall("agents/auth/api_key/token", payload)
}

// executeRequestForAgent execute request for api service
func (m *ManagerOperator) executeRequestForAgent(urlService string, path string, data []byte) ([]byte, error) {
	client := http.Client{}
	urlRequest := fmt.Sprintf("%s%s", urlService, path)
	req, err := http.NewRequest(http.MethodPost, urlRequest, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return respBytes, nil
}

// datadogCall execute call for api agent service
func (m *ManagerOperator) datadogCall(pathDatadog string, datadogDto dto.DatadogDTO) error {
	datadogDtoBytes, err := m.json.Marshall(datadogDto)
	if err != nil {
		return err
	}
	urlService := fmt.Sprintf("http://%s:%s", m.getServiceName(), m.getAgentPort())
	resp, err := m.executeRequestForAgent(urlService, pathDatadog, datadogDtoBytes)
	if err != nil {
		return err
	}
	m.logger.Debug("datadogCall", "resp", string(resp))
	return nil
}

// executeSendLockedEvents execute send locked events
func (m *ManagerOperator) executeSendLockedEvents() error {
	m.logger.Debug("execute send locked events", "timestamp", time.Now())
	if len(m.pendingTransactionEvents) > 0 {
		m.logger.Debug("execute send locked events", "timestamp", time.Now(), "pendingTransactionEvents", m.pendingTransactionEvents)
		newPendingTransactionsEvents := []dto.TransactionStatus{}
		configMapConfiguration, err := m.GetConfigMap(m.configMapConfigurationsName, m.namespace)
		if err != nil {
			m.logger.Error("execute send locked events get config map", "error", err.Error())
			return err
		}
		if accessToken, ok := configMapConfiguration.Data["access_token"]; ok {
			for _, trs := range m.pendingTransactionEvents {
				res, statusCode, err := m.StateCheckSendStatus(trs, accessToken)
				if err != nil {
					m.logger.Error("execute send locked events notify status send status", "error", err.Error())
					return err
				}
				if statusCode == 500 {
					m.LockedEvents = true
					newPendingTransactionsEvents = append(newPendingTransactionsEvents, trs)
				}
				m.logger.Debug("notify status", "timestamp", time.Now(), "response", string(res), "statusCode", statusCode)
			}
			m.pendingTransactionEvents = newPendingTransactionsEvents
		}
	}
	return nil
}
