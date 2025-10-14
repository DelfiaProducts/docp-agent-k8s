package tests

import (
	"os"
	"testing"

	"github.com/OryaHub/agent-k8s/agents"
	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/utils"
)

func TestNewK8sAgent(t *testing.T) {
	bdd.Feature(t, "Instanciar o pkg agent", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro pkg agent", func(s *bdd.Scenario) {
			port := "8080"
			var logger *utils.K8sLogger
			var agent *agents.K8sAgent
			s.Given("que eu tenho uma configuração de logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})

			s.When("eu instancio o pkg agent com essa configuração", func() {
				agent = agents.NewK8sAgent(port, logger)
			})

			s.Then("um logger deve ser criado e associado ao agente", func(t *testing.T) {
				bdd.AssertNoError(t, nil, "Espera-se que o K8sAgent não seja nulo")
				bdd.AssertTrue(t, agent != nil, "O K8sAgent instanciado deve ser diferente de nil")
			})
		})
	})

}
