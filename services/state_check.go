package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

// StateCheckService is struct for state check service
type StateCheckService struct {
	logger        *utils.K8sLogger
	json          *pkg.JsonClient
	stateCheckUrl string
	client        *http.Client
}

// NewStateCheckService return instance of state check service
func NewStateCheckService(logger *utils.K8sLogger) *StateCheckService {
	return &StateCheckService{
		logger: logger,
		json:   pkg.NewJsonClient(),
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

// GetState return state from state check api
func (sc *StateCheckService) GetState(pathUrl string, stateCheckPayload dto.K8sStateCheckPayload, accessToken string) ([]byte, int, error) {
	urlStateCheck := fmt.Sprintf("%s/%s", sc.stateCheckUrl, pathUrl)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStateCheck, nil)
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

// SendStatus execute send status for state check api
func (sc *StateCheckService) SendStatus(pathUrl string, transactionStatus dto.TransactionStatus, accessToken string) ([]byte, int, error) {
	payloadBytes, err := sc.json.Marshall(transactionStatus)
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
