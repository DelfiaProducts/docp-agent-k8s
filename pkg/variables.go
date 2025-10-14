package pkg

import "errors"

var (
	// errors
	K8sOperatorNotImplemented = errors.New("k8s operator not implemented")
	ConfigMapKeyNotFound      = errors.New("config map key not found")
	OryaApiKeyNotFound        = errors.New("orya api key not found")
	ErrNotAuthorized          = errors.New("not authorized")
	ErrAuthTokenClaimsInvalid = errors.New("invalid token claims")
	ErrNotFound               = errors.New("not found")
	ErrNotFoundChart          = errors.New("not found chart")
	ErrNotFoundChartVersion   = errors.New("not found chart version")
	ErrNotFoundReleaseName    = errors.New("not found release name")
	ErrInvalidDatadogMode     = errors.New("invalid datadog mode")

	// transactions events
	TransactionEventOpen   = "open"
	TransactionEventUpdate = "update"
	TransactionEventClose  = "close"
)
