package pkg

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/DelfiaProducts/docp-agent-k8s/dto"
	"github.com/DelfiaProducts/docp-agent-k8s/operators"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"
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

func (k *K8sAgent) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (k *K8sAgent) updateDatadogConfigOperator(w http.ResponseWriter, r *http.Request) {
	k.logger.Info("Requisição recebida", "remoteAddr", r.RemoteAddr, "path", r.URL.Path)
	var datadogDto dto.DatadogDTO
	if err := json.NewDecoder(r.Body).Decode(&datadogDto); err != nil {
		k.logger.Error("update datadog config", "error", err.Error())
		return
	}
	response := struct{}{}
	go k.operator.UpdateDatadogConfigOperator(datadogDto)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		k.logger.Error("update datadog config failed to encode response", "error", err.Error())
	}
	k.logger.Debug("response sent successfully", "status", "ok")
}

func (k *K8sAgent) updateDatadogConfigHelm(w http.ResponseWriter, r *http.Request) {
	k.logger.Info("Requisição recebida", "remoteAddr", r.RemoteAddr, "path", r.URL.Path)
	var datadogDto dto.DatadogDTO
	if err := json.NewDecoder(r.Body).Decode(&datadogDto); err != nil {
		k.logger.Error("update datadog config", "error", err.Error())
		return
	}
	response := struct{}{}
	go k.operator.UpdateDatadogConfigHelm(datadogDto)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		k.logger.Error("update datadog config failed to encode response", "error", err.Error())
	}
	k.logger.Debug("response sent successfully", "status", "ok")
}

// installDatadogHelm is handler for install datadog with helm
func (k *K8sAgent) installDatadogHelm(w http.ResponseWriter, r *http.Request) {
	k.logger.Info("Requisição recebida", "remoteAddr", r.RemoteAddr, "path", r.URL.Path)
	var datadogDto dto.DatadogDTO
	if err := json.NewDecoder(r.Body).Decode(&datadogDto); err != nil {
		k.logger.Error("install datadog helm", "error", err.Error())
		return
	}
	response := struct{}{}
	go k.operator.InstallDatadogHelm(datadogDto)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		k.logger.Error("install datadog helm failed to encode response", "error", err.Error())
	}
	k.logger.Debug("response sent successfully", "status", "ok")
}

// installDatadogOperator is handler for install datadog with operator
func (k *K8sAgent) installDatadogOperator(w http.ResponseWriter, r *http.Request) {
	k.logger.Info("Requisição recebida", "remoteAddr", r.RemoteAddr, "path", r.URL.Path)
	var datadogDto dto.DatadogDTO
	if err := json.NewDecoder(r.Body).Decode(&datadogDto); err != nil {
		k.logger.Error("install datadog helm", "error", err.Error())
		return
	}
	response := struct{}{}
	go k.operator.InstallDatadogOperator(datadogDto)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		k.logger.Error("install datadog operator failed to encode response", "error", err.Error())
	}
	k.logger.Debug("response sent successfully", "status", "ok")
}

// uninstallDatadogHelm is handler for uninstall datadog with helm
func (k *K8sAgent) uninstallDatadogHelm(w http.ResponseWriter, r *http.Request) {
	k.logger.Info("Requisição recebida", "remoteAddr", r.RemoteAddr, "path", r.URL.Path)
	var datadogDto dto.DatadogDTO
	if err := json.NewDecoder(r.Body).Decode(&datadogDto); err != nil {
		k.logger.Error("uninstall datadog helm", "error", err.Error())
		return
	}
	response := struct{}{}
	go k.operator.UninstallDatadogHelm("datadog-agent", datadogDto.DatadogNamespace)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		k.logger.Error("uninstall datadog helm failed to encode response", "error", err.Error())
	}
	k.logger.Debug("response sent successfully", "status", "ok")
}

// uninstallDatadogOperator is handler for uninstall datadog with operator
func (k *K8sAgent) uninstallDatadogOperator(w http.ResponseWriter, r *http.Request) {
	k.logger.Info("Requisição recebida", "remoteAddr", r.RemoteAddr, "path", r.URL.Path)
	var datadogDto dto.DatadogDTO
	if err := json.NewDecoder(r.Body).Decode(&datadogDto); err != nil {
		k.logger.Error("uninstall datadog operator", "error", err.Error())
		return
	}
	response := struct{}{}
	go k.operator.UninstallDatadogOperator("datadog", "datadog-operator", datadogDto.DatadogNamespace)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		k.logger.Error("uninstall datadog operator failed to encode response", "error", err.Error())
	}
	k.logger.Debug("response sent successfully", "status", "ok")
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
	k.logger.Info("Docp Agent Kubernetes Running", "port", k.port)
	if err := k.Initialize(); err != nil {
		return err
	}
	if err := k.Listen(); err != nil {
		return err
	}
	return nil
}
