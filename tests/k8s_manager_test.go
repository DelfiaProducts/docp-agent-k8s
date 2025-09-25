package tests

import (
	"os"
	"testing"

	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/pkg"
	"github.com/OryaHub/agent-k8s/utils"
)

func TestNewK8sManager(t *testing.T) {
	bdd.Feature(t, "Instanciar o pkg manager", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro pkg manager", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var manager *pkg.K8sManager
			s.Given("que eu tenho uma configuração de logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})

			s.When("eu instancio o pkg manager com essa configuração", func() {
				manager = pkg.NewK8sManager(logger)
			})

			s.Then("um logger deve ser criado e associado ao manager", func(t *testing.T) {
				bdd.AssertNoError(t, nil, "Espera-se que o K8sManager não seja nulo")
				bdd.AssertTrue(t, manager != nil, "O K8sManager instanciado deve ser diferente de nil")
			})
		})
	})
}

func TestK8sManagerAutoUpdateDocp(t *testing.T) {
	bdd.Feature(t, "Auto update docp", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro pkg manager", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var manager *pkg.K8sManager
			var err error
			s.Given("que eu tenho uma configuração de logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})

			s.When("eu instancio o pkg manager com essa configuração", func() {
				manager = pkg.NewK8sManager(logger)
			})

			s.When("eu inicializo o pkg manager", func() {
				err = manager.Initialize()
				bdd.AssertNoError(t, err, "Espera-se que a inicialização do K8sManager não contenha erro")
			})

			s.When("eu auto atualizo o docp", func() {
				err = manager.AutoUpdateDocp()
			})

			s.Then("o auto update do docp não deve conter erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "Espera-se que o K8sManager não seja nulo")
			})
		})
	})
}

func TestK8sManagerAutoUpdateDatadog(t *testing.T) {
	bdd.Feature(t, "Auto update datadog", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro pkg manager", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var manager *pkg.K8sManager
			var err error
			s.Given("que eu tenho uma configuração de logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})

			s.When("eu instancio o pkg manager com essa configuração", func() {
				manager = pkg.NewK8sManager(logger)
			})

			s.When("eu inicializo o pkg manager", func() {
				err = manager.Initialize()
				bdd.AssertNoError(t, err, "Espera-se que a inicialização do K8sManager não contenha erro")
			})

			s.When("eu auto atualizo o datadog", func() {
				err = manager.AutoUpdateDatadog()
			})

			s.Then("o auto update do datadog não deve conter erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "Espera-se que o K8sManager não seja nulo")
			})
		})
	})
}
