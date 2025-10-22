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
	json                        *pkg.JsonClient
	client                      *http.Client
}

// NewAuthService return instance the auth service
func NewAuthService(logger *utils.K8sLogger) *AuthService {
	return &AuthService{
		logger: logger,
		json:   pkg.NewJsonClient(),
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

// AuthCall execute call for auth service
func (as *AuthService) AuthCall(path string, payload dto.K8sAuthPayload) ([]byte, int, error) {
	var resp dto.K8sAuthResponse
	var statusCode int
	var err error
	choiceIndex := rand.Intn(len(AuthMocks))
	choice := AuthMocks[choiceIndex]
	resp.AccessToken = choice.AccessToken
	statusCode = choice.StatusCode
	respBytes, err := as.json.Marshall(&resp)
	if err != nil {
		return nil, 0, err
	}
	err = choice.ErrorMock
	return respBytes, statusCode, err
}
