package operators

import (
	"context"
	"os"
	"sync"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/interfaces"
	"github.com/OryaHub/agent-k8s/mocks"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/services"
	"github.com/OryaHub/agent-k8s/templates"
	"github.com/OryaHub/agent-k8s/utils"
)

type FuncFactory func(ctx context.Context, factorySignal *dto.FactorySignalDTO) error

// ManagerOperator is struct for manager operator
type ManagerOperator struct {
	wg                               *sync.WaitGroup
	done                             chan struct{}
	logger                           *utils.K8sLogger
	kubeClient                       *utils.KubeClient
	helmClient                       *utils.HelmClient
	json                             *pkg.JsonClient
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
		json:                             pkg.NewJsonClient(),
		kubeClient:                       utils.NewKubeClient(),
		wg:                               &sync.WaitGroup{},
		done:                             make(chan struct{}),
		mapFactorySignal:                 make(map[string]FuncFactory),
		mapFactoryVendorsSignalUninstall: make(map[string]FuncFactory),
		mapFactoryVendorsSignalUpdate:    make(map[string]FuncFactory),
		namespace:                        utils.GetOryaNamespace(),
		configMapStateName:               utils.GetOryaConfiMapStateName(),
		configMapConfigurationsName:      utils.GetOryaConfigMapConfigurationsName(),

		pendingTransactionEvents: make([]dto.TransactionStatus, 0),
		LockedEvents:             false,
	}
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

// UpdateChartRepository update the helm repository
func (m *ManagerOperator) UpdateChartRepository() error {
	m.logger.Debug("update helm repository")
	if err := m.helmClient.UpdateChartRepository(utils.GetHelmRepositoryName(), utils.GetHelmRepository()); err != nil {
		m.logger.Error("failed to update helm repository", "error", err.Error())
		return err
	}
	return nil
}

// AutoUninstall execute auto uninstall the orya agent
func (m *ManagerOperator) AutoUninstall(namespace string) error {
	job := templates.TemplateJobAutoUninstall(namespace)
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

// Start execute running the goroutines operator
func (m *ManagerOperator) Start() error {
	m.logger.Debug("start goroutines operator")
	m.wg.Add(1)
	go m.handleLockedEvents()
	return nil
}
