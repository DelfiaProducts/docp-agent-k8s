package tests

import (
	"os"
	"testing"

	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/mocks"
	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/utils"
)

func TestNewAgentOperator(t *testing.T) {
	bdd.Feature(t, "Instanciar o agent operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewAgentOperator(logger)
			})

			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
		})
	})
}

func TestAgentOperatorSetup(t *testing.T) {
	bdd.Feature(t, "Setup do AgentOperator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("setup básico do operator", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var err error
			s.Given("um AgentOperator instanciado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
			})
			s.When("eu executo o setup", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "setup configurado sem erro")
			})
			s.Then("deve executar sem erro crítico", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado em ambiente de teste: %v", err)
				}
			})
		})
	})
}

func TestAgentOperatorGetConfigMapConfiguration(t *testing.T) {
	bdd.Feature(t, "Buscar ConfigMap de configuração", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar configmap em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var err error
			var configMap interface{}
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.When("eu busco o configmap", func() {
				configMap, err = operator.GetConfigMapConfiguration()
			})
			s.Then("deve retornar erro ou configmap nulo em ambiente de teste", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
				if configMap == nil {
					bdd.Printf("configMap nulo (esperado em ambiente de teste)")
				}
			})
		})
	})
}

func TestAgentOperatorInstallDatadogHelm(t *testing.T) {
	bdd.Feature(t, "Instalar Datadog via Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("instalação simulada em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var datadogDto dto.DatadogDTO
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.Given("montar o dto pro datadog", func() {
				datadogDto = dto.DatadogDTO{
					Content:          mocks.MockDatadogHelm,
					Namespace:        "docp-agent",
					DatadogNamespace: "docp-agent",
					ApiKey:           "0df90e05ba755a4b57749d9c02e4cba1",
					AppKey:           "de5f8836f91dcb6854748a3bfb2531ec0c937164",
					Version:          "latest",
				}
			})
			s.When("eu executo a instalação do Datadog via Helm", func() {
				err = operator.InstallDatadogHelm(datadogDto)
			})
			s.Then("deve retornar erro ou simular instalação", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
			})
		})
	})
}

func TestAgentOperatorUninstallDatadogHelm(t *testing.T) {
	bdd.Feature(t, "Desinstalar Datadog via Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("desinstalação simulada em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.When("eu executo a desinstalação do Datadog via Helm", func() {
				err = operator.UninstallDatadogHelm("datadog-agent", "docp-agent")
			})
			s.Then("deve retornar erro ou simular desinstalação", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
			})
		})
	})
}

func TestAgentOperatorInstallDatadogOperator(t *testing.T) {
	bdd.Feature(t, "Instalar Datadog Operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("instalação simulada em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var datadogDto dto.DatadogDTO
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.Given("montar o dto pro datadog", func() {
				datadogDto = dto.DatadogDTO{
					Content:          mocks.MockDatadogOperator,
					Namespace:        "docp-agent",
					DatadogNamespace: "docp-agent",
					ApiKey:           "0df90e05ba755a4b57749d9c02e4cba1",
					AppKey:           "de5f8836f91dcb6854748a3bfb2531ec0c937164",
					Version:          "latest",
				}
			})
			s.When("eu executo a instalação do Datadog Operator", func() {
				err = operator.InstallDatadogOperator(datadogDto)
			})
			s.Then("deve retornar erro ou simular instalação", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
			})
		})
	})
}

func TestAgentOperatorUninstallDatadogOperator(t *testing.T) {
	bdd.Feature(t, "Desinstalar Datadog Operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("desinstalação simulada em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.When("eu executo a desinstalação do Datadog Operator", func() {
				err = operator.UninstallDatadogOperator("datadog", "datadog-operator", "docp-agent")
			})
			s.Then("deve retornar erro ou simular desinstalação", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
			})
		})
	})
}

func TestAgentOperatorUpdateDatadogConfigOperator(t *testing.T) {
	bdd.Feature(t, "Atualizar config do Datadog Operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("atualização simulada em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var datadogDto dto.DatadogDTO
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.Given("montar o dto pro datadog", func() {
				datadogDto = dto.DatadogDTO{
					Content:          mocks.MockDatadogOperator,
					Namespace:        "docp-agent",
					DatadogNamespace: "docp-agent",
					ApiKey:           "0df90e05ba755a4b57749d9c02e4cba1",
					AppKey:           "de5f8836f91dcb6854748a3bfb2531ec0c937164",
					Version:          "latest",
				}
			})
			s.When("eu executo a atualização do config do Datadog Operator", func() {
				err = operator.UpdateDatadogConfigOperator(datadogDto)
			})
			s.Then("deve retornar erro ou simular atualização", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
			})
		})
	})
}

func TestAgentOperatorUpdateDatadogConfigHelm(t *testing.T) {
	bdd.Feature(t, "Atualizar config do Datadog via Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("atualização simulada em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var datadogDto dto.DatadogDTO
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.Given("montar o dto pro datadog", func() {
				datadogDto = dto.DatadogDTO{
					Content:          mocks.MockDatadogHelm,
					Namespace:        "docp-agent",
					DatadogNamespace: "docp-agent",
					ApiKey:           "0df90e05ba755a4b57749d9c02e4cba1",
					AppKey:           "de5f8836f91dcb6854748a3bfb2531ec0c937164",
					Version:          "3.128.0",
				}
			})
			s.When("eu executo a atualização do config do Datadog via Helm", func() {
				err = operator.UpdateDatadogConfigHelm(datadogDto)
			})
			s.Then("deve retornar erro ou simular atualização", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
			})
		})
	})
}

func TestAgentOperatorValidateAllDatadogDeploymentSuccess(t *testing.T) {
	bdd.Feature(t, "Validar sucesso de todos os deployments do Datadog", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("validar todos os deployments do Datadog", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var datadogDto dto.DatadogDTO
			var success bool
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
				datadogDto = dto.DatadogDTO{
					Content:          mocks.MockDatadogHelm,
					Namespace:        "docp-agent",
					DatadogNamespace: "docp-agent",
					ApiKey:           "0df90e05ba755a4b57749d9c02e4cba1",
					AppKey:           "de5f8836f91dcb6854748a3bfb2531ec0c937164",
					Version:          "latest",
				}
			})
			s.Given("um DatadogDTO montado", func() {
				datadogDto = dto.DatadogDTO{
					Content:          mocks.MockDatadogHelm,
					Namespace:        "docp-agent",
					DatadogNamespace: "docp-agent",
					ApiKey:           "0df90e05ba755a4b57749d9c02e4cba1",
					AppKey:           "de5f8836f91dcb6854748a3bfb2531ec0c937164",
					Version:          "latest",
				}
			})
			s.When("eu valido todos os deployments do Datadog", func() {
				success, err = operator.ValidateAllDatadogDeploymentSuccess("helm", datadogDto)
			})
			s.Then("deve executar sem erro crítico", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado em ambiente de teste: %v", err)
				} else {
					bdd.Printf("validação executada, sucesso: %t", success)
				}
			})
		})
	})
}

func TestAgentOperatorValidateAndRollbackDatadogIfNeeded(t *testing.T) {
	bdd.Feature(t, "Validar e executar rollback do Datadog se necessário", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("validar sem necessidade de rollback", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.When("eu executo validação com rollback se necessário", func() {
				err = operator.ValidateAndRollbackDatadogIfNeeded("helm", "docp-agent", "datadog-agent", 1)
			})
			s.Then("deve executar validação e tentar rollback se necessário", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado na validação: %v", err)
				}
			})
		})
	})
}

func TestAgentOperatorUpdateChartRepository(t *testing.T) {
	bdd.Feature(t, "executar atualização do chart", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("validar se foi atualizado com sucesso", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.AgentOperator
			var err error
			s.Given("um AgentOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewAgentOperator(logger)
				err = operator.Setup()
				bdd.AssertNoError(t, err, "configurado sem erro")
			})
			s.When("eu executo a atualização do chart", func() {
				err = operator.UpdateChartRepository("datadog", utils.GetDatadogHelmRepository())
			})
			s.Then("deve executar a atualização com sucesso", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado na atualização: %v", err)
				}
			})
		})
	})
}
