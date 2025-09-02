package interfaces

import "github.com/DelfiaProducts/docp-agent-k8s/dto"

// IK8sInterface is interface for agents
type IK8sInterface interface {
	Start() error
}

// IAuthService is interface the auth service
type IAuthService interface {
	Setup() error
	AuthCall(path string, payload dto.K8sAuthPayload) ([]byte, int, error)
}

// IRegisterService is interface the register service
type IRegisterService interface {
	Setup() error
	RegisterCall(path string, registerDto dto.K8sRegister, apiKey string, token string, isCreate bool) ([]byte, int, error)
}

// IStateCheck is interface the state check service
type IStateCheckService interface {
	Setup() error
	GetState(pathUrl string, stateCheckPayload dto.K8sStateCheckPayload, accessToken string) ([]byte, int, error)
	SendStatus(pathUrl string, transactionStatus dto.TransactionStatus, accessToken string) ([]byte, int, error)
}
