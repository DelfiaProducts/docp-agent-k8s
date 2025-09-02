package tests

import (
	"testing"

	"github.com/DelfiaProducts/docp-agent-k8s/bdd"
	"github.com/DelfiaProducts/docp-agent-k8s/mocks"
	"github.com/DelfiaProducts/docp-agent-k8s/utils"
)

func TestDecodeJwt(t *testing.T) {
	bdd.Feature(t, "Decode jwt", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("decodificar jwt", func(s *bdd.Scenario) {
			var token string
			var claims any
			var err error
			s.Given("quando eu tenho um token", func() {
				token = mocks.GetMockJwt("delfia@mail.com")
			})

			s.When("eu decodifico o jwt", func() {
				claims, err = utils.DecodeJwt(token)
			})

			s.Then("o error deve ser nulo no decode", func(t *testing.T) {
				bdd.AssertNoError(t, err, "espera o erro seja nulo")
			})
			bdd.Printf("CLAIMS: %+v\n", claims)
		})
	})
}
