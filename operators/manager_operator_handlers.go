package operators

import (
	"context"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	pkg "github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/templates"
	"github.com/OryaHub/agent-k8s/utils"
)

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
	go m.NotifyStatus("update_orya_received", pkg.TransactionEventOpen, "update orya received", ctx, factorySignal)

	existedVersion, err := m.GetCurrentHelmChartVersion(m.namespace, utils.GetOryaReleaseName())
	if err != nil {
		m.logger.Error("failed to get current helm chart version", "error", err.Error())
		go m.NotifyStatus("update_orya_error", pkg.TransactionEventClose, "failed update agent", ctx, factorySignal)
		return err
	}

	m.logger.Debug("signal update", "existedVersion", existedVersion, "applyVersion", applyVersion)

	alreadyUpdated := m.ValidateVersionHelmAlreadyUpdated(existedVersion, applyVersion)

	m.logger.Debug("already updated", "alreadyUpdated", alreadyUpdated)

	if alreadyUpdated {
		go m.NotifyStatus("update_orya_complete", pkg.TransactionEventClose, "agent already updated with last version", ctx, factorySignal)
		return nil
	}
	m.logger.Debug("execute update version agent", "timestamp", time.Now())
	go m.NotifyStatus("update_orya_processing", pkg.TransactionEventUpdate, "update orya processing", ctx, factorySignal)

	//update repository
	job := templates.TemplateJobAutoUpdateHelmRelease(factorySignal.Namespace, utils.GetOryaReleaseName(), utils.GetHelmRepository(), applyVersion, utils.GetOryaUpdaterRepositoryName(applyVersion))
	if err := m.CreateJob(factorySignal.Namespace, &job); err != nil {
		m.logger.Error("failed to create job", "error", err.Error())
		go m.NotifyStatus("update_orya_error", pkg.TransactionEventClose, "failed update orya", ctx, factorySignal)
		return err
	}
	go m.NotifyStatus("update_orya_completed", pkg.TransactionEventClose, "update orya completed", ctx, factorySignal)

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
		go m.NotifyStatus("update_vendor_received", pkg.TransactionEventOpen, "update vendor received", ctx, factorySignal)
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
	go m.NotifyStatus("uninstall_orya_received", pkg.TransactionEventOpen, "uninstall orya received", ctx, factorySignal)
	m.logger.Debug("execute signal uninstall", "signal type", factorySignal.Signal.TypeSignal)
	if len(factorySignal.Signal.RemoveOtherVendors) > 0 {
		if m.removeAllVendors(factorySignal.Signal.RemoveOtherVendors) {
			go m.NotifyStatus("uninstall_orya_initiate", pkg.TransactionEventUpdate, "all vendors initialize uninstall", ctx, factorySignal)
			// remove all
			m.logger.Debug("signal uninstall remove all vendors", "timestamp", time.Now())
			for _, fn := range m.mapFactoryVendorsSignalUninstall {
				if err := fn(ctx, factorySignal); err != nil {
					return err
				}
			}
		} else {
			// not remove all
			go m.NotifyStatus("uninstall_orya_initiate", pkg.TransactionEventUpdate, "other vendors initialize uninstall", ctx, factorySignal)
			m.logger.Debug("signal uninstall remove olther vendors", "timestamp", time.Now())
			for _, vendor := range factorySignal.Signal.RemoveOtherVendors {
				fn, ok := m.mapFactoryVendorsSignalUninstall[vendor]
				if ok {
					if err := fn(ctx, factorySignal); err != nil {
						return err
					}
				} else {
					m.logger.Debug("uninstall_orya_vendor_not_found", "vendor", vendor)
				}
			}
		}
	}

	time.Sleep(20 * time.Second)
	go m.NotifyStatus("uninstall_orya_update", pkg.TransactionEventUpdate, "auto uninstall update", ctx, factorySignal)

	if err := m.RemoveConfigMaps(factorySignal.Namespace); err != nil {
		m.logger.Error("execute signal auto uninstall remove config maps", "error", err.Error())
		go m.NotifyStatus("uninstall_orya_error", pkg.TransactionEventClose, "failed remove config maps", ctx, factorySignal)
		return err
	}
	if err := m.AutoUninstall(factorySignal.Namespace); err != nil {
		m.logger.Error("execute signal auto uninstall", "error", err.Error())
		go m.NotifyStatus("uninstall_orya_error", pkg.TransactionEventClose, "failed auto uninstall", ctx, factorySignal)
		return err
	}
	if err := m.RemoveOryaMutatingAgent(utils.GetMutatingWebhookName(), factorySignal.Namespace); err != nil {
		m.logger.Error("execute signal auto uninstall remove agent mutate", "error", err.Error())
		go m.NotifyStatus("uninstall_orya_error", pkg.TransactionEventClose, "failed remove agent mutate", ctx, factorySignal)
		return err
	}

	if err := m.RemoveOryaNamespace(factorySignal.Namespace); err != nil {
		go m.NotifyStatus("uninstall_orya_error", pkg.TransactionEventClose, "failed remove orya namespace", ctx, factorySignal)
		return err
	}

	go m.NotifyStatus("uninstall_orya_completed", pkg.TransactionEventClose, "uninstall orya completed", ctx, factorySignal)
	return nil
}

// SignalDebugSession execute signal debug session
func (m *ManagerOperator) SignalDebugSession(ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	return nil
}

// DatadogUninstall execute uninstall the datadog
func (m *ManagerOperator) DatadogUninstall(ctx context.Context, factorySignal *dto.FactorySignalDTO) error {
	m.logger.Debug("execute datadog uninstall", "timestamp", time.Now())
	go m.NotifyStatus("uninstall_orya_vendor_received", pkg.TransactionEventUpdate, "uninstall orya vendor received", ctx, factorySignal)
	configMapConfiguration, err := m.GetConfigMap(factorySignal.ConfigMapConfigurationName, factorySignal.Namespace)
	if err != nil {
		m.logger.Error("execute signal get config map on datadog uninstall", "error", err.Error())
		go m.NotifyStatus("uninstall_orya_vendor_error", pkg.TransactionEventClose, "failed uninstall orya vendor", ctx, factorySignal)
		return err
	}
	exists, err := m.VerifyDatadogResourceExists("datadog", factorySignal.DatadogNamespace)
	if err != nil {
		m.logger.Error("verify datadog resource exists", "error", err.Error())
		go m.NotifyStatus("uninstall_orya_vendor_error", pkg.TransactionEventClose, "failed uninstall orya vendor", ctx, factorySignal)
		return err
	}
	if exists {
		modeDatadog, ok := configMapConfiguration.Data["datadog_mode"]
		if ok {
			switch modeDatadog {
			case "helm":
				go m.NotifyStatus("uninstall_orya_vendor_processing", pkg.TransactionEventUpdate, "uninstall orya vendor processing", ctx, factorySignal)
				if err := m.UninstallDatadogCall("helm", dto.DatadogDTO{DatadogNamespace: factorySignal.DatadogNamespace}); err != nil {
					m.logger.Error("execute uninstall datadog with helm", "error", err.Error())
					go m.NotifyStatus("uninstall_orya_vendor_error", pkg.TransactionEventClose, "failed uninstall orya vendor", ctx, factorySignal)
					return err
				}
				go m.NotifyStatus("uninstall_orya_vendor_complete", pkg.TransactionEventClose, "uninstall orya vendo completed", ctx, factorySignal)
			case "operator":
				go m.NotifyStatus("uninstall_orya_vendor_processing", pkg.TransactionEventUpdate, "uninstall orya vendor processing", ctx, factorySignal)
				if err := m.UninstallDatadogCall("operator", dto.DatadogDTO{DatadogNamespace: factorySignal.DatadogNamespace}); err != nil {
					m.logger.Error("execute uninstall datadog with operator", "error", err.Error())
					go m.NotifyStatus("uninstall_orya_vendor_error", pkg.TransactionEventClose, "failed uninstall orya vendor", ctx, factorySignal)
					return err
				}
				go m.NotifyStatus("uninstall_orya_vendor_complete", pkg.TransactionEventClose, "uninstall orya vendor completed", ctx, factorySignal)
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
		go m.NotifyStatus("update_orya_vendor_error", pkg.TransactionEventClose, "failed update configurations vendor", ctx, factorySignal)
		return err
	}
	if exists {
		modeDatadog, ok := configMapConfiguration.Data["datadog_mode"]
		modeVendor := factorySignal.Signal.Vendor.Mode
		if ok {
			switch modeDatadog {
			case "helm":
				if modeVendor == "helm" {
					go m.NotifyStatus("update_orya_vendor_processing", pkg.TransactionEventUpdate, "update orya vendor processing", ctx, factorySignal)
					if err := m.UpdateDatadogConfigurations("helm", dto.DatadogDTO{DatadogNamespace: factorySignal.DatadogNamespace, Content: factorySignal.Signal.Vendor.Content, Version: factorySignal.Signal.Vendor.Version}); err != nil {
						m.logger.Error("execute update datadog with helm", "error", err.Error())
						go m.NotifyStatus("uninstall_orya_vendor_error", pkg.TransactionEventClose, "failed update vendor configurations", ctx, factorySignal)
						return err
					}
					configMapConfiguration.Data["datadog_hash"] = utils.GenerateHashMd5([]byte(factorySignal.Signal.Vendor.Content))
					if err := m.UpdateConfigMap(m.configMapConfigurationsName, m.namespace, configMapConfiguration.Data); err != nil {
						go m.NotifyStatus("uninstall_orya_vendor_error", pkg.TransactionEventClose, "failed update vendor configurations", ctx, factorySignal)
						return err
					}
					go m.NotifyStatus("update_orya_vendor_complete", pkg.TransactionEventClose, "update orya vendor completed", ctx, factorySignal)
				}
			case "operator":
				if modeVendor == "operator" {
					go m.NotifyStatus("update_orya_vendor_processing", pkg.TransactionEventUpdate, "update orya vendor processing", ctx, factorySignal)
					if err := m.UpdateDatadogConfigurations("operator", dto.DatadogDTO{DatadogNamespace: factorySignal.DatadogNamespace, Content: factorySignal.Signal.Vendor.Content, Version: factorySignal.Signal.Vendor.Version}); err != nil {
						m.logger.Error("execute update datadog with operator", "error", err.Error())
						go m.NotifyStatus("update_orya_vendor__error", pkg.TransactionEventClose, "failed update vendor configurations", ctx, factorySignal)
						return err
					}
					configMapConfiguration.Data["datadog_hash"] = utils.GenerateHashMd5([]byte(factorySignal.Signal.Vendor.Content))
					if err := m.UpdateConfigMap(m.configMapConfigurationsName, m.namespace, configMapConfiguration.Data); err != nil {
						go m.NotifyStatus("uninstall_orya_vendor_error", pkg.TransactionEventClose, "failed update vendor configurations", ctx, factorySignal)
						return err
					}
					go m.NotifyStatus("update_orya_vendor_complete", pkg.TransactionEventClose, "update orya vendor completed", ctx, factorySignal)
				}
			}
		}
	} else {
		go m.NotifyStatus("update_orya_vendor_error", pkg.TransactionEventClose, "failed verify vendor exists", ctx, factorySignal)
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
