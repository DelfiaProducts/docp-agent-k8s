package agents

import (
	"fmt"
	"net/http"
	"os"

	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/utils"
)

// K8sAgent is struct for agent the k8s
type K8sAgent struct {
	port     string
	logger   *utils.K8sLogger
	operator *operators.AgentOperator
	version  string
}

// NewK8sAgent return instance of k8s agent
func NewK8sAgent(port string, logger *utils.K8sLogger) *K8sAgent {
	return &K8sAgent{
		port:   port,
		logger: logger,
	}
}

// Initialize execute initialization the agent
func (k *K8sAgent) Initialize() error {
	operator := operators.NewAgentOperator(k.logger)
	if err := operator.Setup(); err != nil {
		return err
	}
	k.operator = operator
	configMapConfiguration, err := k.operator.GetConfigMapConfiguration()
	if err != nil {
		return err
	}
	k.version = configMapConfiguration.Data["version"]
	return nil
}

func (k *K8sAgent) Listen() error {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "12012"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/datadog/update/operator", k.updateDatadogConfigOperator)
	mux.HandleFunc("/datadog/update/helm", k.updateDatadogConfigHelm)
	mux.HandleFunc("/datadog/install/helm", k.installDatadogHelm)
	mux.HandleFunc("/datadog/uninstall/helm", k.uninstallDatadogHelm)
	mux.HandleFunc("/datadog/install/operator", k.installDatadogOperator)
	mux.HandleFunc("/datadog/uninstall/operator", k.uninstallDatadogOperator)
	mux.HandleFunc("/health", k.health)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (k *K8sAgent) Start() error {
	k.logger.Info("Orya Agent Kubernetes Running", "port", k.port)
	if err := k.Initialize(); err != nil {
		return err
	}
	if err := k.Listen(); err != nil {
		return err
	}
	return nil
}
