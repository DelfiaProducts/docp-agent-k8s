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

// RegisterService is struct for register service
type RegisterService struct {
	urlRegister                 string
	configMapConfigurationsName string
	logger                      *utils.K8sLogger
	json                        *pkg.JsonClient
	client                      *http.Client
}

// NewRegisterService return instance the register service
func NewRegisterService(logger *utils.K8sLogger) *RegisterService {
	return &RegisterService{
		logger: logger,
		json:   pkg.NewJsonClient(),
	}
}

// Setup execute configurations for register service
func (rs *RegisterService) Setup() error {
	urlDomain, err := utils.GetDomainUrl()
	if err != nil {
		return err
	}
	rs.urlRegister = urlDomain
	rs.configMapConfigurationsName = utils.GetOryaConfigMapConfigurationsName()
	client := &http.Client{
		Timeout: time.Second * 90,
	}
	rs.client = client
	return nil
}

// RegisterCall execute send metadata from cluster to
// register service
func (rs *RegisterService) RegisterCall(path string, registerDto dto.K8sRegister, apiKey string, token string, isCreate bool) ([]byte, int, error) {
	url := fmt.Sprintf("%s/%s", rs.urlRegister, path)
	metaBytes, err := rs.json.Marshall(registerDto)
	if err != nil {
		return nil, 0, err
	}
	var method string
	if isCreate {
		method = http.MethodPost
	} else {
		method = http.MethodPut
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(metaBytes))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if isCreate {
		req.Header.Set("docp-api-key", apiKey)
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	res, err := rs.client.Do(req)
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
