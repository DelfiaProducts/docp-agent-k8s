package dto

// K8sConfig is struct for state
type K8sConfig struct {
	Signal K8sConfigSignal `json:"signal"`
}

type K8sConfigSignal struct {
	TypeSignal         string                     `json:"type"`
	TraceID            string                     `json:"trace_id"`
	RemoveOtherVendors []string                   `json:"remove_other_vendors"`
	Duration           string                     `json:"duration"`
	Agents             K8sConfigSignalAgents      `json:"agents"`
	Labels             K8sConfigSignalLabels      `json:"labels"`
	Annotations        K8sConfigSignalAnnotations `json:"annotations"`
}

type K8sConfigSignalAgents struct {
	OryaAgent    K8sConfigSignalOrya         `json:"docp-agent"`
	DatadogAgent K8sConfigSignalDatadogAgent `json:"datadog-agent"`
}

type K8sConfigSignalOrya struct {
	Version string `json:"version"`
}

type K8sConfigSignalDatadogAgent struct {
	Version   string `json:"version"`
	Mode      string `json:"mode"`
	Enabled   bool   `json:"enabled"`
	DeployYml string `json:"deploy-yml"`
	ApiKey    string `json:"api-key"`
	AppKey    string `json:"app-key"`
}

// DatadogDTO is struct for dto the operations with datadog
type DatadogDTO struct {
	Content          string `json:"content"`
	Namespace        string `json:"namespace"`
	DatadogNamespace string `json:"datadog_namespace"`
	ApiKey           string `json:"api_key"`
	AppKey           string `json:"app_key"`
	Version          string `json:"version"`
}

// K8sAction is struct for actions
type K8sAction struct {
	Action   string
	Provider string
	Content  string
	Version  string
	Envs     map[string]string
}

// K8sSignal is struct for signal
type K8sSignal struct {
	TypeSignal         string
	Version            string
	Duration           string
	RemoveOtherVendors []string
	Vendor             K8sSignalVendor
	Action             K8sAction
}

// K8sSignalVendor is struct for vendor signal
type K8sSignalVendor struct {
	Name    string
	Mode    string
	Content string
	Version string
}

type K8sSignalModifications struct {
	ModeDatadog     string
	ModeLabels      string
	ModeAnnotations string
}

// K8sSignalSpec is struct the spec for signal
type K8sSignalSpec struct {
	Version  string `json:"version"`
	Duration string `json:"duration"`
}

// K8sConfigSignalLabels is struct for labels
type K8sConfigSignalLabels struct {
	Namespaces  []K8sConfigSignalLabelsNamespace  `json:"namespaces"`
	Deployments []K8sConfigSignalLabelsDeployment `json:"deployments"`
}

// K8sConfigSignalLabelsNamespace is struct for actions
// the add labels
type K8sConfigSignalLabelsNamespace struct {
	Name string   `json:"name"`
	Add  []string `json:"add"`
}

// K8sConfigStateLabelsDeployment is struct for actions
// the add labels when deployments
type K8sConfigSignalLabelsDeployment struct {
	Name string   `json:"name"`
	Add  []string `json:"add"`
}

// K8sConfigSignalAnnotations is struct for annotations action
type K8sConfigSignalAnnotations struct {
	Namespaces  []K8sConfigSignalAnnotationsNamespace  `json:"namespaces"`
	Deployments []K8sConfigSignalAnnotationsDeployment `json:"deployments"`
}

// K8sConfigSignalAnnotationsNamespace is struct for actions
// the add annotations
type K8sConfigSignalAnnotationsNamespace struct {
	Name string   `json:"name"`
	Add  []string `json:"add"`
}

// K8sConfigSignalAnnotationsDeployment is struct for actions
// the add annotations when deployments
type K8sConfigSignalAnnotationsDeployment struct {
	Name string   `json:"name"`
	Add  []string `json:"add"`
}

// K8sRegister is struct for register payload
type K8sRegister struct {
	ClusterName string              `json:"cluster_name"`
	NodeNames   []string            `json:"node_names"`
	Namespaces  []string            `json:"namespaces"`
	Tags        []string            `json:"tags"`
	Metadata    K8sRegisterMetadata `json:"metadata"`
}

// K8sRegisterMetadata is struct for metadata the register
type K8sRegisterMetadata struct {
	ComputeInfo K8sRegisterPlataform `json:"compute_info"`
}

// K8sRegisterPlataform is struct for platform the metadata register
type K8sRegisterPlataform struct {
	ComputeName  string `json:"compute_name"`
	PlatformArch string `json:"platform_arch"`
}

// K8sRegisterResponse is struct for response the register
type K8sRegisterResponse struct {
	AccessToken string `json:"access_token"`
}

// K8sStateCheckPayload is struct for payload to state check
type K8sStateCheckPayload struct{}

// K8sAuthPayload is struct for auth payload
type K8sAuthPayload struct {
	ApiKey    string `json:"api_key"`
	ComputeId string `json:"compute_id"`
}

// K8sAuthResponse is struct for auth response
type K8sAuthResponse struct {
	AccessToken string `json:"access_token"`
}

// FactorySignalDTO is dto for factory signal
type FactorySignalDTO struct {
	Signal                     *K8sSignal
	Namespace                  string
	DatadogNamespace           string
	ConfigMapConfigurationName string
}

// ctxKey is type for key context
type ctxKey string

// ContextTransactionStatus is const for context transaction
const ContextTransactionStatus ctxKey = "transactionStatus"

// TransactionStatus is struct for transaction status
type TransactionStatus struct {
	ID        string `json:"id"`
	TraceID   string `json:"trace_id"`
	UlidEvent string `json:"ulid_event"`
	TypeEvent string `json:"type"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// AuthTokenClaims is struct for auth token claims
type AuthTokenClaims struct {
	OryaOrgId int    `json:"docp_org_id"`
	ComputeId string `json:"compute_id"`
}

// VendorInstalled is struct for data the vendors
type VendorInstalled struct {
	Installed bool           `json:"installed"`
	Namespace string         `json:"namespace"`
	Mode      string         `json:"mode"`
	Configs   map[string]any `json:"configs"`
}
