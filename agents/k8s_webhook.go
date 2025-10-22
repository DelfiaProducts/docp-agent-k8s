package agents

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/utils"
	v1 "k8s.io/api/admission/v1"
)

// K8sWebhook is struct for webhook the k8s
type K8sWebhook struct {
	port                        string
	logger                      *utils.K8sLogger
	operator                    *operators.WebhookOperator
	configMapStateName          string
	configMapConfigurationsName string
	namespace                   string
	version                     string
}

// NewK8sWebhook return instance of k8s webhook
func NewK8sWebhook(port string, logger *utils.K8sLogger) *K8sWebhook {
	return &K8sWebhook{
		port:                        port,
		logger:                      logger,
		configMapStateName:          utils.GetOryaConfiMapStateName(),
		configMapConfigurationsName: utils.GetOryaConfigMapConfigurationsName(),
		namespace:                   utils.GetOryaNamespace(),
	}
}

// Initialize execute initialization
func (k *K8sWebhook) Initialize() error {
	operator := operators.NewWebhookOperator(k.logger)
	if err := operator.Setup(); err != nil {
		return err
	}
	k.operator = operator
	if err := k.populateVersion(); err != nil {
		return err
	}
	return nil
}

// ApplyAdmission execute apply for admission
func (k *K8sWebhook) ApplyAdmission(body []byte) (v1.AdmissionReview, error) {
	admissionReview, err := k.operator.GetAdmissionReview(body)
	if err != nil {
		k.logger.Error("get admission review error", "error", err.Error())
		return v1.AdmissionReview{}, err
	}
	configState, err := k.operator.GetConfigMap(k.configMapStateName, k.namespace)
	if err != nil {
		k.logger.Error("get config map error", "error", err.Error())
		return v1.AdmissionReview{}, err
	}
	k.logger.Debug("apply admission", "configStateData", configState.Data)
	var k8sConfigState dto.K8sConfig
	stateReceived := configState.Data["received"]
	reader := strings.NewReader(stateReceived)
	if err := json.NewDecoder(reader).Decode(&k8sConfigState); err != nil {
		k.logger.Error("decode config state", "error", err.Error())
		return v1.AdmissionReview{}, err

	}

	k.logger.Debug("apply admission", "k8sConfigState", k8sConfigState)
	labels := k8sConfigState.Signal.Labels
	annotations := k8sConfigState.Signal.Annotations
	k.logger.Debug("apply admission", "labels", labels, "annotations", annotations)

	populatedLabels, populatedAnnotations, modified, err := k.operator.PopulateLabelsAndAnnotations(admissionReview, labels, annotations)
	if err != nil {
		return v1.AdmissionReview{}, err
	}
	k.logger.Debug("apply admission", "populatedLabels", populatedLabels, "populatedAnnotations", populatedAnnotations, "modified", modified)
	response, err := k.operator.ApplyPatchForAdmissionReview(admissionReview, populatedLabels, populatedAnnotations, modified)
	if err != nil {
		return v1.AdmissionReview{}, err
	}
	k.logger.Debug("apply admission", "response", response)
	return response, nil
}

func (k *K8sWebhook) Listen() error {
	certFile := "/etc/webhook/tls/tls.crt"
	keyFile := "/etc/webhook/tls/tls.key"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to load cert and key: %v", err)
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/", k.handler)
	mux.HandleFunc("/health", k.health)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", k.port),
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		return err
	}
	return nil
}

func (k *K8sWebhook) Start() error {
	k.logger.Info("Orya Webhook Kubernetes Running", "port", k.port)
	if err := k.Initialize(); err != nil {
		return err
	}
	if err := k.Listen(); err != nil {
		return err
	}
	return nil
}
