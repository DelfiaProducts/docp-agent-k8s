package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DelfiaProducts/docp-agent-k8s/dto"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"
)

// RegisterService is struct for register service
type RegisterService struct {
	urlRegister                 string
	configMapConfigurationsName string
	logger                      *utils.K8sLogger
	client                      *http.Client
}

// NewRegisterService return instance the register service
func NewRegisterService(logger *utils.K8sLogger) *RegisterService {
	return &RegisterService{
		logger: logger,
	}
}

// Setup execute configurations for register service
func (rs *RegisterService) Setup() error {
	urlDomain, err := utils.GetDomainUrl()
	if err != nil {
		return err
	}
	rs.urlRegister = urlDomain
	rs.configMapConfigurationsName = utils.GetDocpConfigMapConfigurationsName()
	client := &http.Client{
		Timeout: time.Second * 90,
	}
	rs.client = client
	return nil
}

// marshaller execute marshal the struct for slice the bytes
func (rs *RegisterService) marshaller(inner any) ([]byte, error) {
	rs.logger.Debug("execute marshaller", "inner", inner)
	resBytes, err := json.Marshal(inner)
	if err != nil {
		rs.logger.Error("error in execute marshaller", "error", err.Error())
		return nil, err
	}
	return resBytes, nil
}

// unmarshaller execute unmarshal the content bytes
func (rs *RegisterService) unmarshaller(content []byte, inner any) error {
	rs.logger.Debug("execute unmarshaller", "content", string(content), "inner", inner)
	if err := json.Unmarshal(content, inner); err != nil {
		rs.logger.Error("error in execute unmarshaller", "error", err.Error())
		return err
	}
	return nil
}

// RegisterCall execute send metadata from cluster to
// register service
func (rs *RegisterService) RegisterCall(path string, registerDto dto.K8sRegister, apiKey string, token string, isCreate bool) ([]byte, int, error) {
	url := fmt.Sprintf("%s/%s", rs.urlRegister, path)
	metaBytes, err := rs.marshaller(registerDto)
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
