package tests

import (
	"os"
	"testing"

	"github.com/DelfiaProducts/docp-agent-k8s/bdd"
	"github.com/DelfiaProducts/docp-agent-k8s/services"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"
)

func TestNewAuthService(t *testing.T) {
	bdd.Feature(t, "Instanciar o auth service", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var auth *services.AuthService
			var logger *utils.K8sLogger
			var err error

			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)

			})
			s.When("eu instancio o auth", func() {
				auth = services.NewAuthService(logger)
			})

			s.When("eu realizo o setup auth", func() {
				err = auth.Setup()
			})

			s.Then("o auth deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, auth, "o auth deve ser diferente de nil")
			})

			s.Then("o setup do auth nao deve ter erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "o setup não deve conter erro")
			})
		})
	})
}
