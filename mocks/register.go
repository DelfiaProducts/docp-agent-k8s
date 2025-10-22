package mocks

import (
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

type RegisterMockResponse struct {
	StatusCode  int
	AccessToken string
	ErrorMock   error
}

var RegisterMocks = []RegisterMockResponse{
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
	var resp dto.K8sRegisterResponse
	var statusCode int
	var err error
	choiceIndex := rand.Intn(len(RegisterMocks))
	choice := RegisterMocks[choiceIndex]
	resp.AccessToken = choice.AccessToken
	statusCode = choice.StatusCode
	respBytes, err := rs.json.Marshall(&resp)
	if err != nil {
		return nil, 0, err
	}
	err = choice.ErrorMock
	return respBytes, statusCode, err
}
