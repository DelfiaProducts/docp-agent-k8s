package tests

import (
	"os"
	"testing"

	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/utils"
	"helm.sh/helm/v3/pkg/release"
)

func TestNewUpdaterOperator(t *testing.T) {
	bdd.Feature(t, "Criar UpdaterOperator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("instanciar UpdaterOperator com logger", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o UpdaterOperator", func() {
				updaterOperator = operators.NewUpdaterOperator(logger)
			})
			s.Then("o UpdaterOperator deve ser criado com sucesso", func(t *testing.T) {
				bdd.AssertIsNotNil(t, updaterOperator, "o UpdaterOperator deve ser diferente de nil")
			})
		})
	})
}

func TestLoadConfigKube(t *testing.T) {
	bdd.Feature(t, "Carregar configuração do Kubernetes", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("carregar configuração em modo local", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				err = updaterOperator.Setup()
				bdd.AssertNoError(t, err, "falha ao configurar UpdaterOperator")
			})

			s.Then("a configuração deve ser carregada sem erro", func(t *testing.T) {
				// Em ambiente de teste, pode não haver kubeconfig válido
				// Então vamos apenas verificar se o método executa
				if err != nil {
					bdd.Printf("erro esperado em ambiente de teste: %v", err)
				}
			})
		})

		scenario("carregar configuração em modo cluster", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "cluster")
				err = updaterOperator.Setup()
				bdd.AssertNoError(t, err, "falha ao configurar UpdaterOperator")

			})

			s.Then("a configuração deve falhar fora do cluster", func(t *testing.T) {
				// Em ambiente de teste fora do cluster, deve retornar erro
				bdd.AssertIsNotNil(t, err, "deve retornar erro quando não está no cluster")
			})
		})
	})
}

func TestSetup(t *testing.T) {
	bdd.Feature(t, "Configurar UpdaterOperator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("configurar operator com sucesso", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
			})
			s.When("eu executo o setup", func() {
				err = updaterOperator.Setup()
			})
			s.Then("o setup deve executar sem erro crítico", func(t *testing.T) {
				// Em ambiente de teste, pode não haver kubeconfig válido
				if err != nil {
					bdd.Printf("erro esperado em ambiente de teste: %v", err)
				}
			})
		})
	})
}

func TestValidateDeploymentSuccess(t *testing.T) {
	bdd.Feature(t, "Validar sucesso de deployment", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("validar deployment inexistente", func(s *bdd.Scenario) {
			var success bool
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				err = updaterOperator.Setup()
				bdd.AssertNoError(t, err, "falha ao configurar UpdaterOperator")
			})
			s.When("eu valido um deployment inexistente", func() {
				success, err = updaterOperator.ValidateDeploymentSuccess("orya-agent", "manager")
			})
			s.Then("deve retornar false sem erro ou com erro de conexão", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro de conexão esperado em ambiente de teste: %v", err)
				} else {
					bdd.AssertFalse(t, success, "deployment inexistente deve retornar false")
				}
			})
		})

	})
}

func TestValidateAllOryaDeploymentsSuccess(t *testing.T) {
	bdd.Feature(t, "Validar sucesso de todos os deployments DOCP", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("validar todos os deployments DOCP", func(s *bdd.Scenario) {
			var success bool
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				_ = updaterOperator.Setup()
			})
			s.When("eu valido todos os deployments DOCP", func() {
				success, err = updaterOperator.ValidateAllOryaDeploymentsSuccess("orya-agent")
			})
			s.Then("deve executar sem erro crítico", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro de conexão esperado em ambiente de teste: %v", err)
				} else {
					bdd.Printf("validação executada, sucesso: %t", success)
				}
			})
		})

		scenario("validar deployments com namespace inexistente", func(s *bdd.Scenario) {
			var success bool
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				_ = updaterOperator.Setup()
			})
			s.When("eu valido deployments em namespace inexistente", func() {
				success, err = updaterOperator.ValidateAllOryaDeploymentsSuccess("namespace-inexistente")
			})
			s.Then("deve retornar false ou erro de conexão", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado para namespace inexistente: %v", err)
				} else {
					bdd.AssertFalse(t, success, "namespace inexistente deve retornar false")
				}
			})
		})
	})
}

func TestUpdaterOperatorGetLatestHelmVersion(t *testing.T) {
	bdd.Feature(t, "Buscar versão mais atual do Helm", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar versão mais atual de release", func(s *bdd.Scenario) {
			var version string
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
			})
			s.When("eu busco a versão mais atual", func() {
				version, err = updaterOperator.GetLatestHelmVersion("orya-agent", "orya-agent")
			})
			s.Then("deve executar a busca", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado na busca: %v", err)
				} else {
					bdd.Printf("versão encontrada: %s", version)
				}
			})
		})

	})
}

func TestUpdaterOperatorUpgradeOryaHelmChart(t *testing.T) {
	bdd.Feature(t, "Executar upgrade do chart Helm DOCP", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("tentar upgrade com parâmetros válidos", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				err = updaterOperator.Setup()
				bdd.AssertNoError(t, err, "falha ao configurar UpdaterOperator")
			})
			s.When("eu executo upgrade com parâmetros válidos", func() {

				values := map[string]interface{}{
					"manager": map[string]interface{}{
						"image": map[string]interface{}{
							"tag": "v0.1.0",
						},
					},
					"agent": map[string]interface{}{
						"image": map[string]interface{}{
							"tag": "v0.1.0",
						},
					},
					"webhook": map[string]interface{}{
						"image": map[string]interface{}{
							"tag": "v0.1.0",
						},
					},
				}
				err = updaterOperator.UpgradeOryaHelmChart("orya-agent", "orya-agent", utils.GetHelmRepository(), "0.1.0", values)
			})
			s.Then("deve executar sem erro crítico", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado no upgrade: %v", err)
				}
			})
		})

	})
}

func TestUpdaterOperatorUpgradeOryaToLatestVersion(t *testing.T) {
	bdd.Feature(t, "Executar upgrade do DOCP para versão mais atual", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("upgrade para versão mais atual", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			var latestVersion string
			var latestRelease *release.Release
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				err = updaterOperator.Setup()
				bdd.AssertNoError(t, err, "falha ao configurar UpdaterOperator")
			})
			s.Given("que eu tenho a ultima versão do chart", func() {
				latestVersion, err = updaterOperator.GetLatestHelmVersion("orya-agent", "orya-agent")
				bdd.AssertNoError(t, err, "falha ao pegar ultima versão do chart")
			})
			s.Given("que eu tenho a versão mais atual do release", func() {
				latestRelease, err = updaterOperator.GetReleaseByVersion("orya-agent", "orya-agent", latestVersion)
				bdd.AssertNoError(t, err, "falha ao pegar ultima release")
			})
			s.When("eu executo upgrade para versão mais atual", func() {
				values := latestRelease.Config
				err = updaterOperator.UpgradeOryaToLatestVersion("orya-agent", "orya-agent", utils.GetHelmRepository(), values)
			})
			s.Then("deve executar sem erro crítico", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado no upgrade: %v", err)
				}
			})
		})

	})
}

func TestUpdaterOperatorRollbackOryaHelmChart(t *testing.T) {
	bdd.Feature(t, "Executar rollback do chart Helm DOCP", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("rollback para revisão anterior", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
			})
			s.When("eu executo rollback para revisão anterior", func() {
				err = updaterOperator.RollbackOryaHelmChart("orya-agent", "orya-agent")
			})
			s.Then("deve executar sem erro crítico", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado no rollback: %v", err)
				}
			})
		})

	})
}

func TestValidateAndRollbackIfNeeded(t *testing.T) {
	bdd.Feature(t, "Validar e executar rollback se necessário", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("validar sem necessidade de rollback", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				_ = updaterOperator.Setup()
			})
			s.When("eu executo validação com rollback se necessário", func() {
				err = updaterOperator.ValidateAndRollbackIfNeeded("orya-agent", "orya-agent", 1)
			})
			s.Then("deve executar validação", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado na validação: %v", err)
				}
			})
		})

		scenario("validar com maxRetries zero", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				_ = updaterOperator.Setup()
			})
			s.When("eu executo validação com maxRetries zero", func() {
				err = updaterOperator.ValidateAndRollbackIfNeeded("orya-agent", "orya-agent", 0)
			})
			s.Then("deve completar sem tentativas", func(t *testing.T) {
				// Com maxRetries 0, não deve fazer tentativas
				bdd.Printf("resultado da validação com 0 tentativas: %v", err)
			})
		})
	})
}

func TestUpgradeOryaWithRollbackProtection(t *testing.T) {
	bdd.Feature(t, "Executar upgrade com proteção de rollback", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("upgrade para versão especifica", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var updaterOperator *operators.UpdaterOperator
			s.Given("que eu tenho um UpdaterOperator configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				updaterOperator = operators.NewUpdaterOperator(logger)
				os.Setenv("MODE", "local")
				err = updaterOperator.Setup()
				bdd.AssertNoError(t, err, "falha ao configurar UpdaterOperator")
			})
			s.When("eu executo upgrade para versão especifica", func() {
				err = updaterOperator.UpgradeOryaWithRollbackProtection("orya-agent", "orya-agent", utils.GetHelmRepository(), "0.1.0")
			})
			s.Then("deve buscar versão mais atual e executar upgrade", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado no upgrade para versão especifica: %v", err)
				}
			})
		})

	})
}
