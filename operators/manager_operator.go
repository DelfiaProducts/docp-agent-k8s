package operators

import (
	"bytes"
	"context"
	"encoding/json"
	defaultErrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/DelfiaProducts/docp-agent-k8s/dto"
	"github.com/DelfiaProducts/docp-agent-k8s/interfaces"
	"github.com/DelfiaProducts/docp-agent-k8s/internal"
	"github.com/DelfiaProducts/docp-agent-k8s/mocks"
	"github.com/DelfiaProducts/docp-agent-k8s/services"
	"github.com/DelfiaProducts/docp-agent-k8s/templates"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbcav1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type FuncFactory func(ctx context.Context, factorySignal *dto.FactorySignalDTO) error

// ManagerOperator is struct for manager operator
type ManagerOperator struct {
	wg                               *sync.WaitGroup
	done                             chan struct{}
	logger                           *utils.K8sLogger
	kubeClient                       *utils.KubeClient
	helmClient                       *utils.HelmClient
	registerService                  interfaces.IRegisterService
	stateCheckService                interfaces.IStateCheckService
	authService                      interfaces.IAuthService
	configMapStateName               string
	configMapConfigurationsName      string
	namespace                        string
	mapFactorySignal                 map[string]FuncFactory
	mapFactoryVendorsSignalUninstall map[string]FuncFactory
	mapFactoryVendorsSignalUpdate    map[string]FuncFactory
	pendingTransactionEvents         []dto.TransactionStatus
	LockedEvents                     bool
}

// NewManagerOperator return instance of manager operator
func NewManagerOperator(logger *utils.K8sLogger) *ManagerOperator {
	return &ManagerOperator{
		logger:                           logger,
		helmClient:                       utils.NewHelmClient(logger),
		kubeClient:                       utils.NewKubeClient(),
		wg:                               &sync.WaitGroup{},
		done:                             make(chan struct{}),
		mapFactorySignal:                 make(map[string]FuncFactory),
		mapFactoryVendorsSignalUninstall: make(map[string]FuncFactory),
		mapFactoryVendorsSignalUpdate:    make(map[string]FuncFactory),
		namespace:                        utils.GetDocpNamespace(),
		configMapStateName:               utils.GetDocpConfiMapStateName(),
		configMapConfigurationsName:      utils.GetDocpConfigMapConfigurationsName(),

		pendingTransactionEvents: make([]dto.TransactionStatus, 0),
		LockedEvents:             false,
	}
}

// populateFactorySignals populate map the signals
func (m *ManagerOperator) populateFactorySignals() error {
	m.mapFactorySignal["update_agent"] = m.SignalUpdateAgent
	m.mapFactorySignal["update_vendor"] = m.SignalUpdateVendor
	m.mapFactorySignal["uninstall"] = m.SignalUninstall
	m.mapFactorySignal["debug-session"] = m.SignalDebugSession
	return nil
}

// populateFactoryVendorsSignalsUninstall populate map the vendors signals uninstall
func (m *ManagerOperator) populateFactoryVendorsSignalsUninstall() error {
	m.mapFactoryVendorsSignalUninstall["datadog"] = m.DatadogUninstall
	return nil
}

// populateFactoryVendorsSignalsUpdate populate map the vendors signals update
func (m *ManagerOperator) populateFactoryVendorsSignalsUpdate() error {
	m.mapFactoryVendorsSignalUpdate["datadog"] = m.DatadogUpdate
	return nil
}

// removeAllVendors verify if all remove vendors
func (m *ManagerOperator) removeAllVendors(vendors []string) bool {
	for _, vendor := range vendors {
		if strings.Contains(vendor, "all") {
			return true
		}
	}
	return false
}

// Setup configure operator
func (m *ManagerOperator) Setup() error {
	if err := m.populateFactorySignals(); err != nil {
		return err
	}

	if err := m.populateFactoryVendorsSignalsUninstall(); err != nil {
		return err
	}

	if err := m.populateFactoryVendorsSignalsUpdate(); err != nil {
		return err
	}

	if err := m.helmClient.Setup(); err != nil {
		return err
	}

	if err := m.kubeClient.LoadConfigKube(); err != nil {
		return err
	}

	mockable := os.Getenv("MOCKABLE")
	var registerService interfaces.IRegisterService
	var authService interfaces.IAuthService
	var stateCheckService interfaces.IStateCheckService
	if mockable == "true" {
		authService = mocks.NewAuthService(m.logger)
		registerService = mocks.NewRegisterService(m.logger)
		stateCheckService = mocks.NewStateCheckService(m.logger)
	} else {
		authService = services.NewAuthService(m.logger)
		registerService = services.NewRegisterService(m.logger)
		stateCheckService = services.NewStateCheckService(m.logger)
	}
	if err := authService.Setup(); err != nil {
		return err
	}
	if err := registerService.Setup(); err != nil {
		return err
	}
	if err := stateCheckService.Setup(); err != nil {
		return err
	}
	m.authService = authService
	m.registerService = registerService
	m.stateCheckService = stateCheckService
	return nil
}

// IsLocked return if operator locked for send transactions events
func (m *ManagerOperator) IsLockedEvents() bool {
	return m.LockedEvents
}

// verifyAndUpdateLockedEvents verify and update locked events variable
func (m *ManagerOperator) verifyAndUpdateLockedEvents() error {
	m.logger.Debug("verify and update locked events", "pendingTransactionEvents", m.pendingTransactionEvents)
	if len(m.pendingTransactionEvents) == 0 {
		m.LockedEvents = false
	}
	return nil
}

// NotifyStatus execute notify the status to state check
func (m *ManagerOperator) NotifyStatus(status string, typeEvent string, message string, ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	configMapConfiguration, err := m.GetConfigMap(factorySignal.ConfigMapConfigurationName, factorySignal.Namespace)
	if err != nil {
		m.logger.Error("notify status get config map", "error", err.Error())
		return err
	}
	if accessToken, ok := configMapConfiguration.Data["access_token"]; ok {
		transactionStatus := utils.GetTransactionFromContext(ctx)
		if len(transactionStatus.ID) > 0 {
			transactionStatus.Status = status
			transactionStatus.Message = message
			transactionStatus.TypeEvent = typeEvent
			transactionStatus.UlidEvent = utils.GetUlid()
			if m.LockedEvents {
				m.pendingTransactionEvents = append(m.pendingTransactionEvents, transactionStatus)
			} else {
				m.logger.Debug("notify status send status", "transactionStatus", transactionStatus)
				res, statusCode, err := m.StateCheckSendStatus(transactionStatus, accessToken)
				if err != nil {
					m.logger.Error("notify status send status", "error", err.Error())
					return err
				}
				if statusCode == 500 {
					m.LockedEvents = true
					m.pendingTransactionEvents = append(m.pendingTransactionEvents, transactionStatus)
				}
				m.logger.Debug("notify status", "timestamp", time.Now(), "response", string(res), "statusCode", statusCode)
			}
		}
	}

	return nil
}

// ValidateVersionHelmAlreadyUpdated return if helm chart version is already updated
func (m *ManagerOperator) ValidateVersionHelmAlreadyUpdated(currentVersion, targetVersion string) bool {
	return targetVersion == currentVersion
}

// GetCurrentHelmChartVersion return the current helm chart version
func (m *ManagerOperator) GetCurrentHelmChartVersion(namespace, releaseName string) (string, error) {
	currentRelease, err := m.helmClient.GetCurrentRelease(namespace, releaseName)
	if err != nil {
		return "", err
	}
	version := currentRelease.Chart.Metadata.Version
	return version, nil
}

// GetLatestHelmChartVersion return the latest helm chart version
func (m *ManagerOperator) GetLatestHelmChartVersion(namespace, releaseName string) (string, error) {
	latestVersion, err := m.helmClient.GetLatestHelmVersion(namespace, releaseName)
	if err != nil {
		return "", err
	}
	return latestVersion, nil
}

// UpdateChartRepository update the helm repository
func (m *ManagerOperator) UpdateChartRepository() error {
	m.logger.Debug("update helm repository")
	if err := m.helmClient.UpdateChartRepository(m.helmClient.RepositoryName, utils.GetHelmRepository()); err != nil {
		m.logger.Error("failed to update helm repository", "error", err.Error())
		return err
	}
	return nil
}

// GetReleaseName return release name
func (m *ManagerOperator) GetReleaseName(namespace, chartName string) (string, error) {
	return m.helmClient.GetReleaseName(namespace, chartName)
}

// GetReleaseModeDatadog return release mode from datadog
func (m *ManagerOperator) GetReleaseModeDatadog(namespace string) (string, error) {
	return m.helmClient.GetReleaseModeDatadog(namespace)
}

// ExecuteSignal execute functions the signals
func (m *ManagerOperator) ExecuteSignal(factorySignal *dto.FactorySignalDTO) error {
	transaction := utils.NewTransactionStatus()
	ctx := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)
	fn, ok := m.mapFactorySignal[factorySignal.Signal.TypeSignal]
	if ok {
		return fn(ctx, factorySignal)
	} else {
		m.logger.Debug("signal not implemented")
		return nil
	}
}

// SignalUpdateAgent execute signal update the agent
func (m *ManagerOperator) SignalUpdateAgent(ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	m.logger.Debug("execute signal", "signal type", factorySignal.Signal.TypeSignal)
	var applyVersion string
	versionSignal := factorySignal.Signal.Version
	if len(versionSignal) > 0 {
		applyVersion = versionSignal
	}
	//verify if latest version and bypass
	if applyVersion == "latest" {
		//bypass update
		m.logger.Debug("update execution for auto update", "signal type", factorySignal.Signal.TypeSignal)
		return nil
	}

	//execute update agent
	go m.NotifyStatus("update_docp_received", internal.TransactionEventOpen, "update docp received", ctx, factorySignal)

	existedVersion, err := m.GetCurrentHelmChartVersion(m.namespace, utils.GetDocpReleaseName())
	if err != nil {
		m.logger.Error("failed to get current helm chart version", "error", err.Error())
		go m.NotifyStatus("update_docp_error", internal.TransactionEventClose, "failed update agent", ctx, factorySignal)
		return err
	}

	m.logger.Debug("signal update", "existedVersion", existedVersion, "applyVersion", applyVersion)

	alreadyUpdated := m.ValidateVersionHelmAlreadyUpdated(existedVersion, applyVersion)

	m.logger.Debug("already updated", "alreadyUpdated", alreadyUpdated)

	if alreadyUpdated {
		go m.NotifyStatus("update_docp_complete", internal.TransactionEventClose, "agent already updated with last version", ctx, factorySignal)
		return nil
	}
	m.logger.Debug("execute update version agent", "timestamp", time.Now())
	go m.NotifyStatus("update_docp_processing", internal.TransactionEventUpdate, "update docp processing", ctx, factorySignal)

	//update repository
	job := templates.TemplateJobAutoUpdateHelmRelease(factorySignal.Namespace, utils.GetDocpReleaseName(), utils.GetHelmRepository(), applyVersion, utils.GetDocpUpdaterRepositoryName(applyVersion))
	if err := m.CreateJob(factorySignal.Namespace, &job); err != nil {
		m.logger.Error("failed to create job", "error", err.Error())
		go m.NotifyStatus("update_docp_error", internal.TransactionEventClose, "failed update docp", ctx, factorySignal)
		return err
	}
	go m.NotifyStatus("update_docp_completed", internal.TransactionEventClose, "update docp completed", ctx, factorySignal)

	return nil
}

// SignalUpdateVendor execute signal update the vendor
func (m *ManagerOperator) SignalUpdateVendor(ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	goNext := false
	vendor := factorySignal.Signal.Vendor.Name
	switch vendor {
	case "datadog":
		configMapConfiguration, err := m.GetConfigMap(m.configMapConfigurationsName, m.namespace)
		if err != nil {
			return err
		}
		exists, err := m.VerifyDatadogResourceExists("datadog", factorySignal.DatadogNamespace)
		if err != nil {
			return err
		}
		datadogHash := configMapConfiguration.Data["datadog_hash"]
		datadogSignalContentHash := utils.GenerateHashMd5([]byte(factorySignal.Signal.Vendor.Content))
		if exists && datadogHash != datadogSignalContentHash {
			goNext = true
		}
	}

	if goNext {
		go m.NotifyStatus("update_vendor_received", internal.TransactionEventOpen, "update vendor received", ctx, factorySignal)
		fn, ok := m.mapFactoryVendorsSignalUpdate[factorySignal.Signal.Vendor.Name]
		if ok {
			return fn(ctx, factorySignal)
		} else {
			m.logger.Debug("signal update vendor not implemented")
			return nil
		}
	}
	return nil
}

// SignalUninstall execute signal uninstall
func (m *ManagerOperator) SignalUninstall(ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	go m.NotifyStatus("uninstall_docp_received", internal.TransactionEventOpen, "uninstall docp received", ctx, factorySignal)
	m.logger.Debug("execute signal uninstall", "signal type", factorySignal.Signal.TypeSignal)
	if len(factorySignal.Signal.RemoveOtherVendors) > 0 {
		if m.removeAllVendors(factorySignal.Signal.RemoveOtherVendors) {
			go m.NotifyStatus("uninstall_docp_initiate", internal.TransactionEventUpdate, "all vendors initialize uninstall", ctx, factorySignal)
			// remove all
			m.logger.Debug("signal uninstall remove all vendors", "timestamp", time.Now())
			for _, fn := range m.mapFactoryVendorsSignalUninstall {
				if err := fn(ctx, factorySignal); err != nil {
					return err
				}
			}
		} else {
			// not remove all
			go m.NotifyStatus("uninstall_docp_initiate", internal.TransactionEventUpdate, "other vendors initialize uninstall", ctx, factorySignal)
			m.logger.Debug("signal uninstall remove olther vendors", "timestamp", time.Now())
			for _, vendor := range factorySignal.Signal.RemoveOtherVendors {
				fn, ok := m.mapFactoryVendorsSignalUninstall[vendor]
				if ok {
					if err := fn(ctx, factorySignal); err != nil {
						return err
					}
				} else {
					m.logger.Debug("uninstall_docp_vendor_not_found", "vendor", vendor)
				}
			}
		}
	}

	time.Sleep(20 * time.Second)
	go m.NotifyStatus("uninstall_docp_update", internal.TransactionEventUpdate, "auto uninstall update", ctx, factorySignal)

	if err := m.RemoveConfigMaps(factorySignal.Namespace); err != nil {
		m.logger.Error("execute signal auto uninstall remove config maps", "error", err.Error())
		go m.NotifyStatus("uninstall_docp_error", internal.TransactionEventClose, "failed remove config maps", ctx, factorySignal)
		return err
	}
	if err := m.AutoUninstall(factorySignal.Namespace); err != nil {
		m.logger.Error("execute signal auto uninstall", "error", err.Error())
		go m.NotifyStatus("uninstall_docp_error", internal.TransactionEventClose, "failed auto uninstall", ctx, factorySignal)
		return err
	}
	if err := m.RemoveDocpMutatingAgent(utils.GetMutatingWebhookName(), factorySignal.Namespace); err != nil {
		m.logger.Error("execute signal auto uninstall remove agent mutate", "error", err.Error())
		go m.NotifyStatus("uninstall_docp_error", internal.TransactionEventClose, "failed remove agent mutate", ctx, factorySignal)
		return err
	}

	if err := m.RemoveDocpNamespace(factorySignal.Namespace); err != nil {
		go m.NotifyStatus("uninstall_docp_error", internal.TransactionEventClose, "failed remove docp namespace", ctx, factorySignal)
		return err
	}

	go m.NotifyStatus("uninstall_docp_completed", internal.TransactionEventClose, "uninstall docp completed", ctx, factorySignal)
	return nil
}

// SignalDebugSession execute signal debug session
func (m *ManagerOperator) SignalDebugSession(ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	return nil
}

// DatadogUninstall execute uninstall the datadog
func (m *ManagerOperator) DatadogUninstall(ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	m.logger.Debug("execute datadog uninstall", "timestamp", time.Now())
	go m.NotifyStatus("uninstall_docp_vendor_received", internal.TransactionEventUpdate, "uninstall docp vendor received", ctx, factorySignal)
	configMapConfiguration, err := m.GetConfigMap(factorySignal.ConfigMapConfigurationName, factorySignal.Namespace)
	if err != nil {
		m.logger.Error("execute signal get config map on datadog uninstall", "error", err.Error())
		go m.NotifyStatus("uninstall_docp_vendor_error", internal.TransactionEventClose, "failed uninstall docp vendor", ctx, factorySignal)
		return err
	}
	exists, err := m.VerifyDatadogResourceExists("datadog", factorySignal.DatadogNamespace)
	if err != nil {
		m.logger.Error("verify datadog resource exists", "error", err.Error())
		go m.NotifyStatus("uninstall_docp_vendor_error", internal.TransactionEventClose, "failed uninstall docp vendor", ctx, factorySignal)
		return err
	}
	if exists {
		modeDatadog, ok := configMapConfiguration.Data["datadog_mode"]
		if ok {
			switch modeDatadog {
			case "helm":
				go m.NotifyStatus("uninstall_docp_vendor_processing", internal.TransactionEventUpdate, "uninstall docp vendor processing", ctx, factorySignal)
				if err := m.UninstallDatadogCall("helm", dto.DatadogDTO{DatadogNamespace: factorySignal.DatadogNamespace}); err != nil {
					m.logger.Error("execute uninstall datadog with helm", "error", err.Error())
					go m.NotifyStatus("uninstall_docp_vendor_error", internal.TransactionEventClose, "failed uninstall docp vendor", ctx, factorySignal)
					return err
				}
				go m.NotifyStatus("uninstall_docp_vendor_complete", internal.TransactionEventClose, "uninstall docp vendo completed", ctx, factorySignal)
			case "operator":
				go m.NotifyStatus("uninstall_docp_vendor_processing", internal.TransactionEventUpdate, "uninstall docp vendor processing", ctx, factorySignal)
				if err := m.UninstallDatadogCall("operator", dto.DatadogDTO{DatadogNamespace: factorySignal.DatadogNamespace}); err != nil {
					m.logger.Error("execute uninstall datadog with operator", "error", err.Error())
					go m.NotifyStatus("uninstall_docp_vendor_error", internal.TransactionEventClose, "failed uninstall docp vendor", ctx, factorySignal)
					return err
				}
				go m.NotifyStatus("uninstall_docp_vendor_complete", internal.TransactionEventClose, "uninstall docp vendor completed", ctx, factorySignal)
			}
		}
	}
	return nil
}

// DatadogUpdate execute update the datadog
func (m *ManagerOperator) DatadogUpdate(ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	m.logger.Debug("execute datadog update", "timestamp", time.Now())
	configMapConfiguration, err := m.GetConfigMap(factorySignal.ConfigMapConfigurationName, factorySignal.Namespace)
	if err != nil {
		return err
	}
	exists, err := m.VerifyDatadogResourceExists("datadog", factorySignal.DatadogNamespace)
	if err != nil {
		m.logger.Error("verify datadog resource exists", "error", err.Error())
		go m.NotifyStatus("update_docp_vendor_error", internal.TransactionEventClose, "failed update configurations vendor", ctx, factorySignal)
		return err
	}
	if exists {
		modeDatadog, ok := configMapConfiguration.Data["datadog_mode"]
		modeVendor := factorySignal.Signal.Vendor.Mode
		if ok {
			switch modeDatadog {
			case "helm":
				if modeVendor == "helm" {
					go m.NotifyStatus("update_docp_vendor_processing", internal.TransactionEventUpdate, "update docp vendor processing", ctx, factorySignal)
					if err := m.UpdateDatadogConfigurations("helm", dto.DatadogDTO{DatadogNamespace: factorySignal.DatadogNamespace, Content: factorySignal.Signal.Vendor.Content, Version: factorySignal.Signal.Vendor.Version}); err != nil {
						m.logger.Error("execute update datadog with helm", "error", err.Error())
						go m.NotifyStatus("uninstall_docp_vendor_error", internal.TransactionEventClose, "failed update vendor configurations", ctx, factorySignal)
						return err
					}
					configMapConfiguration.Data["datadog_hash"] = utils.GenerateHashMd5([]byte(factorySignal.Signal.Vendor.Content))
					if err := m.UpdateConfigMap(m.configMapConfigurationsName, m.namespace, configMapConfiguration.Data); err != nil {
						go m.NotifyStatus("uninstall_docp_vendor_error", internal.TransactionEventClose, "failed update vendor configurations", ctx, factorySignal)
						return err
					}
					go m.NotifyStatus("update_docp_vendor_complete", internal.TransactionEventClose, "update docp vendor completed", ctx, factorySignal)
				}
			case "operator":
				if modeVendor == "operator" {
					go m.NotifyStatus("update_docp_vendor_processing", internal.TransactionEventUpdate, "update docp vendor processing", ctx, factorySignal)
					if err := m.UpdateDatadogConfigurations("operator", dto.DatadogDTO{DatadogNamespace: factorySignal.DatadogNamespace, Content: factorySignal.Signal.Vendor.Content, Version: factorySignal.Signal.Vendor.Version}); err != nil {
						m.logger.Error("execute update datadog with operator", "error", err.Error())
						go m.NotifyStatus("update_docp_vendor__error", internal.TransactionEventClose, "failed update vendor configurations", ctx, factorySignal)
						return err
					}
					configMapConfiguration.Data["datadog_hash"] = utils.GenerateHashMd5([]byte(factorySignal.Signal.Vendor.Content))
					if err := m.UpdateConfigMap(m.configMapConfigurationsName, m.namespace, configMapConfiguration.Data); err != nil {
						go m.NotifyStatus("uninstall_docp_vendor_error", internal.TransactionEventClose, "failed update vendor configurations", ctx, factorySignal)
						return err
					}
					go m.NotifyStatus("update_docp_vendor_complete", internal.TransactionEventClose, "update docp vendor completed", ctx, factorySignal)
				}
			}
		}
	} else {
		go m.NotifyStatus("update_docp_vendor_error", internal.TransactionEventClose, "failed verify vendor exists", ctx, factorySignal)
	}

	return nil
}

// DatadogAlreadyInstalled execute validation if datadog exists and return data from instalation
func (m *ManagerOperator) DatadogAlreadyInstalled(resourceName string, namespaces []corev1.Namespace) (dto.VendorInstalled, error) {
	vendor := dto.VendorInstalled{
		Installed: false,
	}
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return vendor, err
	}

	for _, namespace := range namespaces {
		pods, err := clientset.CoreV1().Pods(namespace.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			continue
		}
		prefix := "datadog"
		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, prefix) {
				vendor.Installed = true
				vendor.Namespace = namespace.Name

				labels := pod.GetLabels()

				mode := "unknown"
				if managedBy, ok := labels["app.kubernetes.io/managed-by"]; ok && managedBy == "datadog-operator" {
					mode = "operator"
				} else if managedBy, ok := labels["app.kubernetes.io/managed-by"]; ok && managedBy == "Helm" {
					mode = "helm"
				}
				vendor.Mode = mode
				return vendor, nil
			}
		}

	}

	return vendor, nil
}

// VerifyDatadogResourceExists identify if exist resource datadog
func (m *ManagerOperator) VerifyDatadogResourceExists(resourceName, namespace string) (bool, error) {
	clientsetDynamic, err := dynamic.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return false, err
	}
	gvr := schema.GroupVersionResource{
		Group:    "datadoghq.com",
		Version:  "v2alpha1",
		Resource: "datadogagents",
	}

	objDatadog, err := clientsetDynamic.Resource(gvr).Namespace(namespace).Get(context.Background(), resourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return true, nil
		}
	}

	m.logger.Debug("verify datadog resource exists", "objDatadog", objDatadog)
	if objDatadog != nil {
		return true, nil
	}
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return false, err
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	prefix := "datadog"
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, prefix) {
			m.logger.Debug("verify datadog resource exists", "pod name datadog", pod.Name)
			return true, nil
		}
	}
	return false, nil
}

// CreateConfigMap execute creation the config map
func (m *ManagerOperator) CreateConfigMap(configMapName string, namespace string, data map[string]string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			newConfiMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
				},
				Data: data,
			}
			_, errCreate := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &newConfiMap, metav1.CreateOptions{})
			if errCreate != nil {
				return err
			}
		}
	}
	return nil
}

// GetConfigMap execute get the config map
func (m *ManagerOperator) GetConfigMap(configMapName, namespace string) (*corev1.ConfigMap, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return configMap, nil
}

// GetNamespaces execute get the namespaces
func (m *ManagerOperator) GetNamespaces() ([]corev1.Namespace, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	if len(namespaces.Items) == 0 {
		return nil, internal.ErrNotFound
	}
	return namespaces.Items, nil
}

// GetKeyFromConfigMap execute get key from the config map
func (m *ManagerOperator) GetKeyFromConfigMap(key, configMapName, namespace string) (string, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return "", err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", err
		}
	}
	data := configMap.Data
	if val, ok := data[key]; ok {
		return val, nil
	}

	return "", internal.ConfigMapKeyNotFound
}

// UpdateConfigMap execute update the config map
func (m *ManagerOperator) UpdateConfigMap(configMapName string, namespace string, data map[string]string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			newConfiMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
				},
				Data: data,
			}
			_, errCreate := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &newConfiMap, metav1.CreateOptions{})
			if errCreate != nil {
				return err
			}
		}
	}
	configMap.Data = data
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteConfigMap execute remove the config map
func (m *ManagerOperator) DeleteConfigMap(configMapName string, namespace string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	if err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

// GetClusterRole execute get the cluster role
func (m *ManagerOperator) GetClusterRole(clusterRoleName string) (*rbcav1.ClusterRole, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	clusterRole, err := clientset.RbacV1().ClusterRoles().Get(ctx, clusterRoleName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return clusterRole, nil
}

// ListClusterRole execute get the all cluster role
func (m *ManagerOperator) ListClusterRole() (*rbcav1.ClusterRoleList, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return clusterRoles, nil
}

// DeleteClusterRole execute remove the cluster role
func (m *ManagerOperator) DeleteClusterRole(clusterRoleName string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	if err := clientset.RbacV1().ClusterRoles().Delete(ctx, clusterRoleName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// GetClusterRoleBinding execute get the cluster role binding
func (m *ManagerOperator) GetClusterRoleBinding(clusterRoleBindingName string) (*rbcav1.ClusterRoleBinding, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBindingName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return clusterRoleBinding, nil
}

// ListClusterRoleBinding execute get the all cluster role bindings
func (m *ManagerOperator) ListClusterRoleBindig() (*rbcav1.ClusterRoleBindingList, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return clusterRoleBindings, nil
}

// DeleteClusterRoleBinding execute remove the cluster role binding
func (m *ManagerOperator) DeleteClusterRoleBinding(clusterRoleBindingName string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	if err := clientset.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBindingName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// DeleteNamespace execute remove the namespace
func (m *ManagerOperator) DeleteNamespace(namespace string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}

	if err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// DeleteMutatingWebhook execute remove the mutating webhook
func (m *ManagerOperator) DeleteMutatingWebhook(mutatingName string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	if err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, mutatingName, metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// CreateJob execute creation the job
func (m *ManagerOperator) CreateJob(namespace string, job *batchv1.Job) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return err
	}
	_, err = clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// AutoUninstall execute auto uninstall the docp agent
func (m *ManagerOperator) AutoUninstall(namespace string) error {
	job := templates.TemplateJobAutoUninstall(namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveConfigMaps execute remove config maps the docp agent
func (m *ManagerOperator) RemoveConfigMaps(namespace string) error {
	job := templates.TemplateJobRemoveConfiMaps(namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveDocpNamespace execute remove docp namespace
func (m *ManagerOperator) RemoveDocpNamespace(namespace string) error {
	job := templates.TemplateJobRemoveDocpNamespace(namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveDocpHelmRelease execute remove docp helm release
func (m *ManagerOperator) RemoveDocpHelmRelease(releaseName, namespace string) error {
	job := templates.TemplateJobRemoveDocpHelmRelease(releaseName, namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveDocpMutatingAgent execute remove docp mutating agent
func (m *ManagerOperator) RemoveDocpMutatingAgent(mutateName, namespace string) error {
	job := templates.TemplateJobRemoveMutatingWebhook(mutateName, namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveDocpClusterRole execute remove docp cluster role
func (m *ManagerOperator) RemoveDocpClusterRole(clusterRoleName, namespace string) error {
	job := templates.TemplateJobRemoveClusterRole(clusterRoleName, namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// RemoveDocpClusterRoleBinding execute remove docp cluster role binding
func (m *ManagerOperator) RemoveDocpClusterRoleBinding(clusterRoleBindingName, namespace string) error {
	job := templates.TemplateJobRemoveClusterRoleBinding(clusterRoleBindingName, namespace)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// UpdateDeploymentImage execute update the deployment image
func (m *ManagerOperator) UpdateDeploymentImage(namespace, deploymentName, containerName, imageName string) error {
	m.logger.Debug("update deployment image", "namespace", namespace, "deploymentName", deploymentName, "containerName", containerName, "imageName", imageName)
	job := templates.TemplateJobUpdateDeploymentImage(namespace, deploymentName, containerName, imageName)
	if err := m.CreateJob(namespace, &job); err != nil {
		return err
	}
	return nil
}

// InstallDatadog execute call for install datadog agent
func (m *ManagerOperator) InstallDatadogCall(mode string, datadogDto dto.DatadogDTO) error {
	switch mode {
	case "helm":
		if err := m.datadogCall("/datadog/install/helm", datadogDto); err != nil {
			return err
		}
	case "operator":
		if err := m.datadogCall("/datadog/install/operator", datadogDto); err != nil {
			return err
		}
	}
	return nil
}

// UnistallDatadog execute call for uninstall datadog agent
func (m *ManagerOperator) UninstallDatadogCall(mode string, datadogDto dto.DatadogDTO) error {
	switch mode {
	case "helm":
		if err := m.datadogCall("/datadog/uninstall/helm", datadogDto); err != nil {
			return err
		}
	case "operator":
		if err := m.datadogCall("/datadog/uninstall/operator", datadogDto); err != nil {
			return err
		}
	}
	return nil
}

// UpdateDatadogConfigurations execute call for update datadog agent
func (m *ManagerOperator) UpdateDatadogConfigurations(mode string, datadogDto dto.DatadogDTO) error {
	switch mode {
	case "helm":
		if err := m.datadogCall("/datadog/update/helm", datadogDto); err != nil {
			return err
		}
	case "operator":
		if err := m.datadogCall("/datadog/update/operator", datadogDto); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteRegisterCall execute call for service register
func (m *ManagerOperator) ExecuteRegisterCall(mode string, metadata dto.K8sRegister, apiKey string, token string) ([]byte, int, error) {
	transaction := utils.NewTransactionStatus()
	ctx := context.WithValue(context.Background(), dto.ContextTransactionStatus, transaction)
	factoryDto := dto.FactorySignalDTO{
		Namespace:                  utils.GetDocpNamespace(),
		ConfigMapConfigurationName: utils.GetDocpConfigMapConfigurationsName(),
	}
	switch mode {
	case "create":
		resp, statusCode, err := m.registerService.RegisterCall("compute/v1/docp", metadata, apiKey, token, true)
		if err != nil {
			return nil, 0, err
		}
		return resp, statusCode, err
	case "update":
		go m.NotifyStatus("update_metadata", internal.TransactionEventOpen, "update metadata", ctx, &factoryDto)
		resp, statusCode, err := m.registerService.RegisterCall("compute/v1/docp", metadata, apiKey, token, false)
		if err != nil {
			go m.NotifyStatus("update_metadata_error", internal.TransactionEventClose, "error on update metadata", ctx, &factoryDto)
		}
		go m.NotifyStatus("update_metadata_completed", internal.TransactionEventClose, "update metadata completed", ctx, &factoryDto)
		return resp, statusCode, err

	}
	return nil, 0, nil
}

// StateCheckCall execute call to state check service
func (m *ManagerOperator) StateCheckCall(payload dto.K8sStateCheckPayload, accessToken string) ([]byte, int, error) {
	return m.stateCheckService.GetState("compute/v1/status/info", payload, accessToken)
}

// StateCheckSendStatus execute call to state check and
// send status
func (m *ManagerOperator) StateCheckSendStatus(transactionStatus dto.TransactionStatus, accessToken string) ([]byte, int, error) {
	return m.stateCheckService.SendStatus("compute/transaction", transactionStatus, accessToken)
}

// ExecuteAuthCall execute call to auth
func (m *ManagerOperator) ExecuteAuthCall(payload dto.K8sAuthPayload) ([]byte, int, error) {
	return m.authService.AuthCall("agents/auth/api_key/token", payload)
}

// getServiceName return service name from envs
func (m *ManagerOperator) getServiceName() string {
	return os.Getenv("SERVICE_AGENT_NAME")
}

// getAgentPort return agent port from envs
func (m *ManagerOperator) getAgentPort() string {
	return os.Getenv("AGENT_PORT")
}

// marshaller execute marshal the obj to bytes
func (m *ManagerOperator) marshaller(obj any) ([]byte, error) {
	return json.Marshal(obj)
}

// executeRequestForAgent execute request for api service
func (m *ManagerOperator) executeRequestForAgent(urlService string, path string, data []byte) ([]byte, error) {
	client := http.Client{}
	urlRequest := fmt.Sprintf("%s%s", urlService, path)
	req, err := http.NewRequest(http.MethodPost, urlRequest, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return respBytes, nil
}

// datadogCall execute call for api agent service
func (m *ManagerOperator) datadogCall(pathDatadog string, datadogDto dto.DatadogDTO) error {
	datadogDtoBytes, err := m.marshaller(datadogDto)
	if err != nil {
		return err
	}
	urlService := fmt.Sprintf("http://%s:%s", m.getServiceName(), m.getAgentPort())
	resp, err := m.executeRequestForAgent(urlService, pathDatadog, datadogDtoBytes)
	if err != nil {
		return err
	}
	m.logger.Debug("datadogCall", "resp", string(resp))
	return nil
}

// getClusterName return cluster name
func (m *ManagerOperator) getClusterName() (string, error) {
	mode := os.Getenv("MODE")
	if mode == "local" {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			kubeconfig = filepath.Join(homeDir, ".kube", "config")
		}

		config, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			return "", err
		}
		currentContextName := config.CurrentContext

		if currentContextName == "" {
			return "", defaultErrors.New("no current context found in kubeconfig")
		}

		currentContext, ok := config.Contexts[currentContextName]
		if !ok {
			return "", defaultErrors.New("current context not found in kubeconfig")
		}
		clusterName := currentContext.Cluster
		if clusterName == "" {
			return "", defaultErrors.New("no cluster name associated with current context")
		}
		return clusterName, nil
	} else {
		var kubeClusterName string
		clusterName := os.Getenv("CLUSTER_NAME")
		kubernetesHost := os.Getenv("KUBERNETES_SERVICE_HOST")
		if len(clusterName) > 0 {
			kubeClusterName = clusterName
		} else if len(kubernetesHost) > 0 {
			kubeClusterName = kubernetesHost
		}
		return kubeClusterName, nil
	}
}

func (m *ManagerOperator) getNodeNames() ([]string, error) {
	var nodesNames []string
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		nodesNames = append(nodesNames, node.Name)
	}
	return nodesNames, nil
}

func (m *ManagerOperator) getNamespaces() ([]string, error) {
	var slcNamespaces []string
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return nil, err
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, namespace := range namespaces.Items {
		slcNamespaces = append(slcNamespaces, namespace.Name)
	}
	return slcNamespaces, nil
}

func (m *ManagerOperator) getClusterArch() (string, error) {
	var arch string
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(m.kubeClient.Config)
	if err != nil {
		return "", err
	}
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, node := range nodes.Items {
		arch = node.Status.NodeInfo.Architecture
		if len(arch) > 0 {
			return arch, nil
		}
	}
	return arch, nil
}

// CollectMetadataK8s return metadata from k8s
func (m *ManagerOperator) CollectMetadataK8s() (dto.K8sRegister, error) {
	registerData := dto.K8sRegister{}
	clusterName, err := m.getClusterName()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.ClusterName = clusterName
	nodeNames, err := m.getNodeNames()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.NodeNames = nodeNames
	namespaces, err := m.getNamespaces()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.Namespaces = namespaces
	arch, err := m.getClusterArch()
	if err != nil {
		return dto.K8sRegister{}, err
	}
	registerData.Metadata.ComputeInfo.PlatformArch = arch
	return registerData, nil
}

// CleanningDatadogLastInstalation execute cleanning the last instalation the datadog
func (m *ManagerOperator) CleanningDatadogLastInstalation() error {
	prefix := "datadog"
	clusterRoles, err := m.ListClusterRole()
	if err != nil {
		return err
	}
	for _, cr := range clusterRoles.Items {
		if strings.HasPrefix(cr.Name, prefix) {
			if err := m.DeleteClusterRole(cr.Name); err != nil {
				return err
			}
		}
	}
	clusterRoleBindings, err := m.ListClusterRoleBindig()
	if err != nil {
		return err
	}
	for _, crb := range clusterRoleBindings.Items {
		if strings.HasPrefix(crb.Name, prefix) {
			if err := m.DeleteClusterRoleBinding(crb.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

// executeSendLockedEvents execute send locked events
func (m *ManagerOperator) executeSendLockedEvents() error {
	m.logger.Debug("execute send locked events", "timestamp", time.Now())
	if len(m.pendingTransactionEvents) > 0 {
		m.logger.Debug("execute send locked events", "timestamp", time.Now(), "pendingTransactionEvents", m.pendingTransactionEvents)
		newPendingTransactionsEvents := []dto.TransactionStatus{}
		configMapConfiguration, err := m.GetConfigMap(m.configMapConfigurationsName, m.namespace)
		if err != nil {
			m.logger.Error("execute send locked events get config map", "error", err.Error())
			return err
		}
		if accessToken, ok := configMapConfiguration.Data["access_token"]; ok {
			for _, trs := range m.pendingTransactionEvents {
				res, statusCode, err := m.StateCheckSendStatus(trs, accessToken)
				if err != nil {
					m.logger.Error("execute send locked events notify status send status", "error", err.Error())
					return err
				}
				if statusCode == 500 {
					m.LockedEvents = true
					newPendingTransactionsEvents = append(newPendingTransactionsEvents, trs)
				}
				m.logger.Debug("notify status", "timestamp", time.Now(), "response", string(res), "statusCode", statusCode)
			}
			m.pendingTransactionEvents = newPendingTransactionsEvents
		}
	}
	return nil
}

// handleLockedEvents execute send locked transactions events
func (m *ManagerOperator) handleLockedEvents() error {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.verifyAndUpdateLockedEvents(); err != nil {
				m.logger.Error("verify and update locked events", "timestamp", time.Now(), "error", err.Error())
			}
			if err := m.executeSendLockedEvents(); err != nil {
				m.logger.Error("execute send locked", "timestamp", time.Now(), "error", err.Error())
			}
		case <-m.done:
			close(m.done)
			return nil
		}
	}
}

// Start execute running the goroutines operator
func (m *ManagerOperator) Start() error {
	m.logger.Debug("start goroutines operator")
	m.wg.Add(1)
	go m.handleLockedEvents()
	return nil
}
