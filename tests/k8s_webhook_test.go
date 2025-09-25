package tests

import (
	"os"
	"testing"

	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

func TestNewK8sWebhook(t *testing.T) {
	bdd.Feature(t, "Instanciar o pkg webhook", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro pkg webhook", func(s *bdd.Scenario) {
			var port string
			var logger *utils.K8sLogger
			var webhook *pkg.K8sWebhook
			s.Given("que eu tenho uma configuração da porta", func() {
				port = ""
			})
			s.Given("que eu tenho uma configuração de logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})

			s.When("eu instancio o pkg webhook com essa configuração", func() {
				webhook = pkg.NewK8sWebhook(port, logger)
			})

			s.Then("o valor da porta nao pode ser vazia", func(t *testing.T) {
				bdd.AssertTrue(t, len(port) > 0, "O valor da porta tem que existir")
			})

			s.Then("um logger deve ser criado e associado ao webhook", func(t *testing.T) {
				bdd.AssertNoError(t, nil, "Espera-se que o K8sWebhook não seja nulo")
				bdd.AssertTrue(t, webhook != nil, "O K8sWebhook instanciado deve ser diferente de nil")
			})
		})
	})
}
