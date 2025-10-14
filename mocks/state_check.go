package mocks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

type StateCheckMockResponse struct {
	StatusCode int
	K8sConfig  dto.K8sConfig
	ErrorMock  error
}

var StateCheckMocks = []StateCheckMockResponse{
	{
		StatusCode: 204,
		K8sConfig:  dto.K8sConfig{},
		ErrorMock:  nil,
	},
	{
		StatusCode: 403,
		K8sConfig:  dto.K8sConfig{},
		ErrorMock:  pkg.ErrNotAuthorized,
	},
	{
		StatusCode: 200,
		K8sConfig: dto.K8sConfig{
			Signal: dto.K8sConfigSignal{
				TypeSignal: "update",
				Agents: dto.K8sConfigSignalAgents{
					OryaAgent: dto.K8sConfigSignalOrya{
						Version: "v1.1",
					},
				},
			},
		},
	},
	{
		StatusCode: 200,
		K8sConfig: dto.K8sConfig{
			Signal: dto.K8sConfigSignal{
				Agents: dto.K8sConfigSignalAgents{
					DatadogAgent: dto.K8sConfigSignalDatadogAgent{
						Version:   "latest",
						Mode:      "operator",
						Enabled:   true,
						DeployYml: MockDatadogOperator,
						ApiKey:    os.Getenv("DATADOG_API_KEY"),
					},
				},
			},
		},
	},
	{
		StatusCode: 200,
		K8sConfig: dto.K8sConfig{
			Signal: dto.K8sConfigSignal{
				Agents: dto.K8sConfigSignalAgents{
					DatadogAgent: dto.K8sConfigSignalDatadogAgent{
						Version:   "latest",
						Mode:      "helm",
						Enabled:   true,
						DeployYml: MockDatadogHelm,
						ApiKey:    os.Getenv("DATADOG_API_KEY"),
					},
				},
				Labels: dto.K8sConfigSignalLabels{
					Namespaces: []dto.K8sConfigSignalLabelsNamespace{
						{
							Name: "teste-mutate-cliente",
							Add: []string{
								"label.orya.com/env=sandbox",
								"tags.datadoghq.com/service=$[labels.app]",
								"tags.datadoghq.com/service2=$[annotations.service2]",
							},
						},
					},
					Deployments: []dto.K8sConfigSignalLabelsDeployment{
						{
							Name: "app1",
							Add: []string{
								"label.orya.com/env=local",
							},
						},
					},
				},
			},
		},
	},
	{
		StatusCode: 200,
		K8sConfig: dto.K8sConfig{
			Signal: dto.K8sConfigSignal{
				Agents: dto.K8sConfigSignalAgents{
					DatadogAgent: dto.K8sConfigSignalDatadogAgent{
						Version:   "latest",
						Mode:      "operator",
						Enabled:   false,
						DeployYml: MockDatadogOperator,
						ApiKey:    os.Getenv("DATADOG_API_KEY"),
					},
				},
			},
		},
	},
	{
		StatusCode: 200,
		K8sConfig: dto.K8sConfig{
			Signal: dto.K8sConfigSignal{
				Agents: dto.K8sConfigSignalAgents{
					DatadogAgent: dto.K8sConfigSignalDatadogAgent{
						Version:   "latest",
						Mode:      "helm",
						Enabled:   false,
						DeployYml: MockDatadogOperator,
						ApiKey:    os.Getenv("DATADOG_API_KEY"),
					},
				},
			},
		},
	},
}

// StateCheckService is struct for state check service
type StateCheckService struct {
	logger        *utils.K8sLogger
	stateCheckUrl string
	client        *http.Client
}

// NewStateCheckService return instance of state check service
func NewStateCheckService(logger *utils.K8sLogger) *StateCheckService {
	return &StateCheckService{
		logger: logger,
	}
}

// Setup configure state check
func (sc *StateCheckService) Setup() error {
	client := &http.Client{
		Timeout: time.Second * 90,
	}
	sc.client = client
	urlDomain, err := utils.GetDomainUrl()
	if err != nil {
		return err
	}
	sc.stateCheckUrl = urlDomain
	return nil
}

// marshaller execute marshal the struct for slice the bytes
func (sc *StateCheckService) marshaller(inner any) ([]byte, error) {
	sc.logger.Debug("execute marshaller", "inner", inner)
	resBytes, err := json.Marshal(inner)
	if err != nil {
		sc.logger.Error("error in execute marshaller", "error", err.Error())
		return nil, err
	}
	return resBytes, nil
}

// GetState return state from state check api
func (sc *StateCheckService) GetState(pathUrl string, stateCheckPayload dto.K8sStateCheckPayload, accessToken string) ([]byte, int, error) {
	var resp dto.K8sConfig
	var statusCode int
	var err error
	choiceIndex := rand.Intn(len(StateCheckMocks))
	choice := StateCheckMocks[choiceIndex]
	resp = choice.K8sConfig
	statusCode = choice.StatusCode
	respBytes, err := json.Marshal(&resp)
	if err != nil {
		return nil, 0, err
	}
	err = choice.ErrorMock
	return respBytes, statusCode, err
}

// SendStatus execute send status for state check api
func (sc *StateCheckService) SendStatus(pathUrl string, transactionStatus dto.TransactionStatus, accessToken string) ([]byte, int, error) {
	payloadBytes, err := sc.marshaller(dto.K8sStateCheckSendStatus{
		Id:        transactionStatus.ID,
		TypeEvent: transactionStatus.TypeEvent,
		Status:    transactionStatus.Status,
		Message:   transactionStatus.Message,
	})
	if err != nil {
		return nil, 0, err
	}
	urlStateCheckStatus := fmt.Sprintf("%s/%s", sc.stateCheckUrl, pathUrl)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStateCheckStatus, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	res, err := sc.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, err
	}
	return respBytes, res.StatusCode, nil
}
