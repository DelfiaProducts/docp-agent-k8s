package bdd

import (
	"fmt"
	"strings"
	"testing"
)

// Scenario define um cenário BDD.
type Scenario struct {
	t           *testing.T
	description string
}

// Given define a etapa "Dado que" de um cenário.
func (s *Scenario) Given(description string, step func()) *Scenario {
	s.runStep("Given", description, step)
	return s
}

// When define a etapa "Quando" de um cenário.
func (s *Scenario) When(description string, step func()) *Scenario {
	s.runStep("When", description, step)
	return s
}

// Then define a etapa "Então" de um cenário.
func (s *Scenario) Then(description string, assertion func(t *testing.T)) *Scenario {
	s.runStep("Then", description, func() {
		assertion(s.t)
	})
	return s
}

// runStep executa uma etapa do cenário.
func (s *Scenario) runStep(keyword, description string, step func()) {
	stepName := fmt.Sprintf("%s %s", keyword, description)
	s.t.Run(stepName, func(t *testing.T) {
		step()
		if t.Failed() {
			fmt.Printf("\t%s\n", stepName)
		} else {
			fmt.Printf("\t%s\n", stepName)
		}
	})
}

// Feature agrupa cenários relacionados.
func Feature(t *testing.T, description string, scenarios func(t *testing.T, f func(description string, steps func(s *Scenario)))) {
	t.Run(description, func(t *testing.T) {
		fmt.Printf("\nFuncionalidade: %s\n", description)
		runScenario := func(scenarioDescription string, steps func(s *Scenario)) {
			scenario := &Scenario{t: t, description: scenarioDescription}
			fmt.Printf("\tCenário: %s\n", scenarioDescription)
			steps(scenario)
		}
		scenarios(t, runScenario)
	})
}

//Printf execute print the any interface
func Printf(format string , a ...any) (int, error) {
	return fmt.Printf(format,a...)
}

// AssertEqual compara dois valores e falha no teste se não forem iguais.
func AssertEqual[T comparable](t *testing.T, expected, actual T, message string) {
	if expected != actual {
		t.Errorf("Falhou: %s\n\tEsperado: %v\n\tObtido:  %v", message, expected, actual)
	}
}

//AssertIsNotNil verifica se um valor é nulo e falha se for
func AssertIsNotNil(t *testing.T, expected any, message string){
if expected == nil{
		t.Errorf("Falhou: %s\n\tEsperado: !nil\n\tObtido:  %v", message, expected)
	}
}

// AssertTrue verifica se uma condição é verdadeira e falha no teste se não for.
func AssertTrue(t *testing.T, condition bool, message string) {
	if !condition {
		t.Errorf("Falhou: %s\n\tEsperado: true\n\tObtido:  false", message)
	}
}

// AssertFalse verifica se uma condição é falsa e falha no teste se não for.
func AssertFalse(t *testing.T, condition bool, message string) {
	if condition {
		t.Errorf("Falhou: %s\n\tEsperado: false\n\tObtido:  true", message)
	}
}

// AssertErrorContains verifica se um erro contém uma substring específica.
func AssertErrorContains(t *testing.T, err error, contains string, message string) {
	if err == nil || !strings.Contains(err.Error(), contains) {
		t.Errorf("Falhou: %s\n\tErro Esperado contendo: '%s'\n\tErro Obtido:         '%v'", message, contains, err)
	}
}

// AssertErrorIsNil verifica se um erro é nulo, e se for falha.
func AssertErrorIsNil(t *testing.T, err error, message string) {
	if err == nil {
		t.Errorf("Falhou: %s\n\tErro Não Esperado: %v", message, err)
	}
}

// AssertNoError verifica se um erro é nulo.
func AssertNoError(t *testing.T, err error, message string) {
	if err != nil {
		t.Errorf("Falhou: %s\n\tErro Não Esperado: %v", message, err)
	}
}
