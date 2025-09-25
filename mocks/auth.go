package mocks

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/utils"
)

type AuthMockResponse struct {
	StatusCode  int
	AccessToken string
	ErrorMock   error
}

var AuthMocks = []AuthMockResponse{
	{
		StatusCode:  200,
		AccessToken: GetMockJwt("tom@email.com"),
		ErrorMock:   nil,
	},
	{
		StatusCode:  403,
		AccessToken: "",
		ErrorMock:   nil,
	},
	{
		StatusCode:  500,
		AccessToken: "",
		ErrorMock:   errors.New("internal server error"),
	},
}

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
	as.configMapConfigurationsName = utils.GetDocpConfigMapConfigurationsName()
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
	var resp dto.K8sAuthResponse
	var statusCode int
	var err error
	choiceIndex := rand.Intn(len(AuthMocks))
	choice := AuthMocks[choiceIndex]
	resp.AccessToken = choice.AccessToken
	statusCode = choice.StatusCode
	respBytes, err := as.marshaller(&resp)
	if err != nil {
		return nil, 0, err
	}
	err = choice.ErrorMock
	return respBytes, statusCode, err
}
