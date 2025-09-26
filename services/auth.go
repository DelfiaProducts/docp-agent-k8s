package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/utils"
)

// AuthService is struct for auth service
type AuthService struct {
	urlAuth                     string
	configMapConfigurationsName string
	logger                      *utils.K8sLogger
	client                      *http.Client
}

// NewAuthService return instance the auth service
func NewAuthService(logger *utils.K8sLogger) *AuthService {
	return &AuthService{
		logger: logger,
	}
}

// Setup execute configurations for auth service
func (as *AuthService) Setup() error {
	urlDomain, err := utils.GetDomainUrl()
	if err != nil {
		return err
	}
	as.urlAuth = urlDomain
	as.configMapConfigurationsName = utils.GetOryaConfigMapConfigurationsName()
	client := &http.Client{
		Timeout: time.Second * 90,
	}
	as.client = client
	return nil
}

// marshaller execute marshal the struct for slice the bytes
func (as *AuthService) marshaller(inner any) ([]byte, error) {
	as.logger.Debug("execute marshaller", "inner", inner)
	resBytes, err := json.Marshal(inner)
	if err != nil {
		as.logger.Error("error in execute marshaller", "error", err.Error())
		return nil, err
	}
	return resBytes, nil
}

// unmarshaller execute unmarshal the content bytes
func (as *AuthService) unmarshaller(content []byte, inner any) error {
	as.logger.Debug("execute unmarshaller", "content", string(content), "inner", inner)
	if err := json.Unmarshal(content, inner); err != nil {
		as.logger.Error("error in execute unmarshaller", "error", err.Error())
		return err
	}
	return nil
}

// AuthCall execute call for auth service
func (as *AuthService) AuthCall(path string, payload dto.K8sAuthPayload) ([]byte, int, error) {
	url := fmt.Sprintf("%s/%s", as.urlAuth, path)
	apiKey := payload.ApiKey
	payloadBytes, err := as.marshaller(&payload)
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("docp-api-key", apiKey)
	res, err := as.client.Do(req)
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
