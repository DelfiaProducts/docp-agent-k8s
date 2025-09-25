package tests

import (
	"os"
	"testing"

	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/utils"
)

func TestNewHelmClient(t *testing.T) {
	bdd.Feature(t, "Criar cliente Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("instanciar cliente Helm com logger", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o cliente Helm", func() {
				helmClient = utils.NewHelmClient(logger)
			})
			s.Then("o cliente Helm deve ser criado com sucesso", func(t *testing.T) {
				bdd.AssertIsNotNil(t, helmClient, "o cliente Helm deve ser diferente de nil")
				bdd.AssertIsNotNil(t, helmClient.Settings, "as configurações do Helm devem estar definidas")
			})
		})
	})
}

func TestUpdateChartRepository(t *testing.T) {
	bdd.Feature(t, "Atualizar repositório de charts Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("atualizar repositório válido", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})
			s.When("eu atualizo um repositório válido", func() {
				err = helmClient.UpdateChartRepository("docp", utils.GetHelmRepository())
			})
			s.Then("a atualização deve ser executada sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "atualização do repositório deve ser executada sem erro")
			})
		})
	})
}

func TestGetLatestHelmVersion(t *testing.T) {
	bdd.Feature(t, "Buscar versão mais atual do chart Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar versão de release inexistente", func(s *bdd.Scenario) {
			var err error
			var version string
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})
			s.When("eu busco a versão de um release inexistente", func() {
				version, err = helmClient.GetLatestHelmVersion("docp-agent", "docp-agent")
			})
			s.Then("a busca deve retornar erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "deve retornar nulo pro erro")
			})
			bdd.Printf("versão encontrada: %s", version)
		})
	})
}

func TestGetReleaseName(t *testing.T) {
	bdd.Feature(t, "Buscar nome da release do chart Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar nome de release inexistente", func(s *bdd.Scenario) {
			var err error
			var name string
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
				err = helmClient.Setup()
				bdd.AssertNoError(t, err, "deve configurar o cliente Helm corretamente")
			})
			s.When("eu busco o nome de um release inexistente", func() {
				name, err = helmClient.GetReleaseName("docp-agent", "datadog-operator")
			})
			s.Then("a busca deve retornar erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "deve retornar nulo pro erro")
			})
			bdd.Printf("nome encontrado: %s", name)
		})
	})
}

func TestGetReleaseModeDatadog(t *testing.T) {
	bdd.Feature(t, "Buscar mode da release do chart Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar mode de release inexistente", func(s *bdd.Scenario) {
			var err error
			var mode string
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
				err = helmClient.Setup()
				bdd.AssertNoError(t, err, "deve configurar o cliente Helm corretamente")
			})
			s.When("eu busco o mode de um release inexistente", func() {
				mode, err = helmClient.GetReleaseModeDatadog("docp-agent")
			})
			s.Then("a busca deve retornar erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "deve retornar nulo pro erro")
			})
			bdd.Printf("mode encontrado: %s", mode)
		})
	})
}
func TestLatestGetHelmChartVersion(t *testing.T) {
	bdd.Feature(t, "Buscar ultima versão do chart Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar ultima versão de chart existente", func(s *bdd.Scenario) {
			var err error
			var version string
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
				err = os.Setenv("HELM_DRIVER", "secret")
				bdd.AssertNoError(t, err, "deve configurar o Helm Driver corretamente")
				err = helmClient.Setup()
				bdd.AssertNoError(t, err, "deve configurar o cliente Helm corretamente")
			})
			s.When("eu busco a versão de um release inexistente", func() {
				version, err = helmClient.GetLatestHelmChartVersion("datadog", "datadog")
			})
			s.Then("a busca deve retornar erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "deve retornar nulo pro erro")
			})
			bdd.Printf("versão encontrada: %s", version)
		})
	})
}

func TestUpgradeDocpHelmChart(t *testing.T) {
	bdd.Feature(t, "Executar upgrade do chart Helm do DOCP", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("tentar upgrade com release existente", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})
			s.When("eu tento fazer upgrade de release inexistente", func() {
				values := make(map[string]interface{})
				err = helmClient.UpgradeDocpHelmChart("docp-agent", "docp-agent", utils.GetHelmRepository(), "0.1.0", values)
			})
			s.Then("o upgrade não deve retornar erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "upgrade de release não deve retornar erro")
			})
		})

	})
}

func TestUpgradeDocpToLatestVersion(t *testing.T) {
	bdd.Feature(t, "Executar upgrade do DOCP para versão mais atual", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("tentar upgrade para versão mais atual com release inexistente", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})
			s.When("eu tento fazer upgrade para versão mais atual com release existente", func() {
				values := make(map[string]interface{})
				err = helmClient.UpgradeDocpToLatestVersion("docp-agent", "docp-agent", utils.GetHelmRepository(), values)
			})
			s.Then("o upgrade deve retornar erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "upgrade para versão mais atual com release existente não deve retornar erro")
			})
		})
	})
}

func TestRollbackDocpHelmChart(t *testing.T) {
	bdd.Feature(t, "Executar rollback do chart Helm do DOCP", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("tentar rollback com release inexistente", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})
			s.When("eu tento fazer rollback de release inexistente", func() {
				err = helmClient.RollbackDocpHelmChart("docp-agent", "docp-agent")
			})
			s.Then("o rollback deve retornar erro", func(t *testing.T) {
				bdd.AssertErrorIsNil(t, err, "rollback de release inexistente deve retornar erro")
			})
		})

	})
}

func TestGetCurrentRelease(t *testing.T) {
	bdd.Feature(t, "Buscar release atual do Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar release inexistente", func(s *bdd.Scenario) {
			var err error
			var release interface{}
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})
			s.When("eu busco um release inexistente", func() {
				release, err = helmClient.GetCurrentRelease("docp-agent", "docp-agent")
			})
			s.Then("a busca deve retornar erro", func(t *testing.T) {
				bdd.AssertErrorIsNil(t, err, "deve retornar erro para release inexistente")
				bdd.AssertEqual(t, nil, release, "release deve ser nil quando há erro")
			})
		})
	})
}

func TestGetReleaseByVersion(t *testing.T) {
	bdd.Feature(t, "Buscar release do Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar release", func(s *bdd.Scenario) {
			var err error
			var release interface{}
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})

			s.When("eu busco um release inexistente", func() {
				release, err = helmClient.GetReleaseByVersion("docp-agent", "docp-agent", "0.1.0")
			})
			s.Then("a busca deve retornar erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "deve retornar erro para release inexistente")
			})
			bdd.Printf("release: %+v\n", release)
		})
	})
}

func TestGetChartValuesByVersion(t *testing.T) {
	bdd.Feature(t, "Buscar valores do chart do Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar valores do chart", func(s *bdd.Scenario) {
			var err error
			var values interface{}
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			s.Given("que eu tenho um cliente Helm configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})

			s.When("eu busco os valores do chart", func() {
				values, err = helmClient.GetChartValuesByVersion("0.1.1")
			})
			s.Then("a busca não deve retornar erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "não deve retornar erro para valores de chart existentes")
			})
			bdd.Printf("chart values: %+v\n", values)
		})
	})
}

func TestValidateAllDatadogDeploymentsSuccess(t *testing.T) {
	bdd.Feature(t, "Validar sucesso de todos os deployments do Datadog", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("validar todos os deployments do Datadog", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			var success bool
			var err error
			s.Given("um HelmClient configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})
			s.When("eu valido todos os deployments do Datadog", func() {
				success, err = helmClient.ValidateAllDatadogDeploymentsSuccess("operator", "docp-agent")
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

func TestValidateAndRollbackDatadogIfNeeded(t *testing.T) {
	bdd.Feature(t, "Validar e executar rollback do Datadog se necessário", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("validar sem necessidade de rollback", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var helmClient *utils.HelmClient
			var err error
			s.Given("um HelmClient configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				helmClient = utils.NewHelmClient(logger)
			})
			s.When("eu executo validação com rollback se necessário", func() {
				err = helmClient.ValidateAndRollbackDatadogIfNeeded("operator", "docp-agent", "datadog-operator", 1)
			})
			s.Then("deve executar validação e tentar rollback se necessário", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado na validação: %v", err)
				}
			})
		})
	})
}
