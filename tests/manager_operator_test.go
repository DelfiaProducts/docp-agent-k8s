package tests

import (
	"context"
	"os"
	"testing"

	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/utils"
	corev1 "k8s.io/api/core/v1"
	rbcav1 "k8s.io/api/rbac/v1"
)

func TestNewManagerOperator(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})

			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
		})
	})
}

func TestManagerOperatorAutoUninstall(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("execute auto uninstall", func() {
				err = operator.AutoUninstall("docp-agent")
			})
			s.Then("o operator deve executar o auto uninstall sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
			s.When("execute remove config maps", func() {
				err = operator.RemoveConfigMaps("docp-agent")
			})
			s.Then("o operator deve executar o remove config maps sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
			s.When("execute remove config maps", func() {
				err = operator.RemoveConfigMaps("docp-agent")
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("execute update deployment image", func() {
				err = operator.UpdateDeploymentImage("docp-agent", "k8s-manager", "manager", "k8s-manager:v1.1")
			})
			s.Then("o operator deve executar o update deployment sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
		})
	})
}

func TestManagerOperatorUpdateDeploymentImage(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("execute update deployment image", func() {
				err = operator.UpdateDeploymentImage("docp-agent", "k8s-manager", "manager", "k8s-manager:v1.1")
			})
			s.Then("o operator deve executar o update deployment sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
			s.When("execute remove config maps", func() {
				err = operator.RemoveConfigMaps("docp-agent")
			})
			s.Then("o operator deve executar o remove config maps sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
		})
	})
}

func TestManagerOperatorExecuteAuthCall(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		var err error
		var logger *utils.K8sLogger
		var operator *operators.ManagerOperator
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
		})
		scenario("executar request for auth", func(s *bdd.Scenario) {
			var k8sAuthPayload dto.K8sAuthPayload
			var resp []byte
			var statusCode int
			var err error
			s.Given("pegar api key", func() {
				k8sAuthPayload = dto.K8sAuthPayload{
					ApiKey: os.Getenv("API_KEY"),
				}
				bdd.AssertTrue(t, len(k8sAuthPayload.ApiKey) > 0, "a api key deve existir")
			})
			s.When("eu executo a request", func() {
				resp, statusCode, err = operator.ExecuteAuthCall(k8sAuthPayload)
			})
			s.Then("a resposta nao pode ser nula", func(t *testing.T) {
				bdd.AssertIsNotNil(t, resp, "response from request diferente de nulo")
			})
			s.Then("o status code tem que ser valido", func(t *testing.T) {
				bdd.AssertTrue(t, statusCode != 0, "status code precisa ser valido")
			})
			s.Then("o error deve ser nulo", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro é nulo")
			})
			bdd.Printf("RESP: %v\n", string(resp))
			bdd.Printf("API_KEY: %v\n", k8sAuthPayload.ApiKey)
			bdd.Printf("STATUS CODE: %v\n", statusCode)
		})
	})
}

func TestManagerOperatorVerifyDatadogResourceExists(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			var exists bool
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("execute verification the datadog resource exist", func() {
				exists, err = operator.VerifyDatadogResourceExists("datadog", "docp-agent")
			})
			s.Then("o operator deve executar a verificação sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
			bdd.Printf("exists: %v\n", exists)
		})
	})
}

func TestManagerOperatorGetClusterRole(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			var clusterRole *rbcav1.ClusterRole
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("execute get cluster role", func() {
				clusterRole, err = operator.GetClusterRole("docp-agent-datadog-orch-exp-dca")
			})
			s.Then("o operator deve executar a verificação sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
			bdd.Printf("cluster role: %v\n", clusterRole)
		})
	})
}

func TestManagerOperatorListClusterRole(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			var clusterRoles *rbcav1.ClusterRoleList
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("execute list cluster role", func() {
				clusterRoles, err = operator.ListClusterRole()
			})
			s.Then("o operator deve executar a verificação sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
			for _, cr := range clusterRoles.Items {
				bdd.Printf("cluster role: name: %s namespace: %s\n", cr.Name, cr.Namespace)
			}
		})
	})
}

func TestManagerOperatorExecuteSignalUpdate(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			var factory *dto.FactorySignalDTO
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("eu crio um factory signal", func() {
				factory = &dto.FactorySignalDTO{
					Namespace:                  "docp-agent",
					ConfigMapConfigurationName: utils.GetDocpConfigMapConfigurationsName(),
					Signal: &dto.K8sSignal{
						TypeSignal: "update_agent",
						Version:    "0.1.0",
					},
				}
			})
			s.When("execute signal", func() {
				err = operator.ExecuteSignal(factory)
			})
			s.Then("o erro da execução do signal precisa ser null", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
		})
	})
}

func TestManagerOperatorExecuteSignalUninstall(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			var factory *dto.FactorySignalDTO
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("eu crio um factory signal", func() {
				factory = &dto.FactorySignalDTO{
					Namespace:                  "docp-agent",
					ConfigMapConfigurationName: utils.GetDocpConfigMapConfigurationsName(),
					Signal: &dto.K8sSignal{
						TypeSignal: "uninstall",
						RemoveOtherVendors: []string{
							"datadog",
						},
					},
				}
			})
			s.When("execute signal", func() {
				err = operator.ExecuteSignal(factory)
			})
			s.Then("o erro da execução do signal precisa ser null", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
		})
	})
}

func TestManagerOperatorNotifyStatus(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var err error
			var logger *utils.K8sLogger
			var operator *operators.ManagerOperator
			var factory *dto.FactorySignalDTO
			var ctx context.Context

			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
			s.When("executo o setup do operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "o erro deve ser nulo no setup")
			})
			s.When("crio um contexto com a transaction como value", func() {
				transactionStatus := utils.NewTransactionStatus()
				ctx = context.WithValue(context.Background(), dto.ContextTransactionStatus, transactionStatus)
			})
			s.When("eu crio um factory signal", func() {
				factory = &dto.FactorySignalDTO{
					Namespace:                  "docp-agent",
					ConfigMapConfigurationName: utils.GetDocpConfigMapConfigurationsName(),
				}
			})
			s.When("execute signal", func() {
				err = operator.NotifyStatus("update_docp_test", "", "", ctx, factory)
			})
			s.Then("o erro da execução do signal precisa ser null", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o erro deve ser nulo")
			})
		})
	})
}

func TestManagerOperatorGetNamespaces(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var err error
			var operator *operators.ManagerOperator
			var namespaces []corev1.Namespace
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
				bdd.AssertIsNotNil(t, operator, "operator não pode ser nullo")
			})

			s.When("configurar o operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "operator configurado sem erro")
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})

			s.When("eu busco namespaces", func() {
				namespaces, err = operator.GetNamespaces()
				bdd.AssertNoError(t, err, "não tivemos erro na busca de namespaces")
			})
			s.Then("os namespaces precisam existir", func(t *testing.T) {
				bdd.AssertTrue(t, len(namespaces) > 0, "o tamanho do slice de namespace precisa ser maior que zero")
			})
			bdd.Printf("OPERATOR: %+v\n", operator)
			bdd.Printf("NAMESPACES: %+v\n", namespaces)
		})
	})
}

func TestManagerOperatorDatadogAlreadyInstalled(t *testing.T) {
	bdd.Feature(t, "Instanciar o manager operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var err error
			var operator *operators.ManagerOperator
			var namespaces []corev1.Namespace
			var vendorInstalled dto.VendorInstalled

			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewManagerOperator(logger)
				bdd.AssertIsNotNil(t, operator, "operator não pode ser nullo")
			})

			s.When("configurar o operator", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "operator configurado sem erro")
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})

			s.When("eu busco namespaces", func() {
				namespaces, err = operator.GetNamespaces()
				bdd.AssertNoError(t, err, "não tivemos erro na busca de namespaces")
			})
			s.Then("os namespaces precisam existir", func(t *testing.T) {
				bdd.AssertTrue(t, len(namespaces) > 0, "o tamanho do slice de namespace precisa ser maior que zero")
			})

			s.When("eu verifico se o datadog esta instalado", func() {
				vendorInstalled, err = operator.DatadogAlreadyInstalled("datadog", namespaces)
				bdd.AssertNoError(t, err, "não tivemos erro na busca de instalação do datadog")
			})
			bdd.Printf("OPERATOR: %+v\n", operator)
			bdd.Printf("NAMESPACES: %+v\n", namespaces)
			bdd.Printf("VENDOR DATADOG: %+v\n", vendorInstalled)
		})
	})
}
