package agents

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OryaHub/agent-k8s/dto"
)

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
