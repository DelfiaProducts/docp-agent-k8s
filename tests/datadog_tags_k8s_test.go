package tests

import (
	"testing"

	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/utils"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ---------------------------------------------------------------------------
// utils.ExtractTagKey
// ---------------------------------------------------------------------------

func TestExtractTagKey(t *testing.T) {
	bdd.Feature(t, "Extrair chave de uma tag", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("tag no formato chave:valor", func(s *bdd.Scenario) {
			var result string
			s.When("eu extraio a chave de 'env:prod'", func() {
				result = utils.ExtractTagKey("env:prod")
			})
			s.Then("deve retornar 'env'", func(t *testing.T) {
				bdd.AssertEqual(t, "env", result, "chave deve ser 'env'")
			})
		})

		scenario("tag sem separador de dois pontos", func(s *bdd.Scenario) {
			var result string
			s.When("eu extraio a chave de 'standalone'", func() {
				result = utils.ExtractTagKey("standalone")
			})
			s.Then("deve retornar a tag inteira como chave", func(t *testing.T) {
				bdd.AssertEqual(t, "standalone", result, "chave deve ser a tag completa")
			})
		})
	})
}

// ---------------------------------------------------------------------------
// utils.MergeTagSlices
// ---------------------------------------------------------------------------

func TestMergeTagSlicesAppend(t *testing.T) {
	bdd.Feature(t, "Fazer merge de tags — adicionar nova tag", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("incoming tem chave nova, não presente em existing", func(s *bdd.Scenario) {
			var existing, incoming, result []string
			s.Given("existing com tag 'env:prod'", func() {
				existing = []string{"env:prod"}
			})
			s.Given("incoming com tag nova 'team:platform'", func() {
				incoming = []string{"team:platform"}
			})
			s.When("faço o merge", func() {
				result = utils.MergeTagSlices(existing, incoming)
			})
			s.Then("resultado deve conter ambas as tags", func(t *testing.T) {
				bdd.AssertEqual(t, 2, len(result), "deve ter 2 tags")
				bdd.AssertEqual(t, "env:prod", result[0], "primeira tag preservada")
				bdd.AssertEqual(t, "team:platform", result[1], "nova tag adicionada")
			})
		})
	})
}

func TestMergeTagSlicesReplace(t *testing.T) {
	bdd.Feature(t, "Fazer merge de tags — tag local tem prioridade sobre incoming", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("incoming tem mesma chave que uma tag de existing — existing vence", func(s *bdd.Scenario) {
			var existing, incoming, result []string
			s.Given("existing com 'env:prod'", func() {
				existing = []string{"env:prod", "team:platform"}
			})
			s.Given("incoming com 'env:staging' (mesma chave 'env')", func() {
				incoming = []string{"env:staging"}
			})
			s.When("faço o merge", func() {
				result = utils.MergeTagSlices(existing, incoming)
			})
			s.Then("o valor local deve ser mantido e o incoming descartado", func(t *testing.T) {
				bdd.AssertEqual(t, 2, len(result), "deve continuar com 2 tags")
				bdd.AssertEqual(t, "env:prod", result[0], "tag local preservada (não substituída)")
				bdd.AssertEqual(t, "team:platform", result[1], "segunda tag preservada")
			})
		})
	})
}

func TestMergeTagSlicesEmptyIncoming(t *testing.T) {
	bdd.Feature(t, "Fazer merge de tags — incoming vazio", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("nenhuma tag incoming", func(s *bdd.Scenario) {
			var existing, result []string
			s.Given("existing com tags", func() {
				existing = []string{"env:prod", "team:platform"}
			})
			s.When("faço merge com incoming vazio", func() {
				result = utils.MergeTagSlices(existing, []string{})
			})
			s.Then("existing é retornado sem alteração", func(t *testing.T) {
				bdd.AssertEqual(t, 2, len(result), "deve ter as mesmas 2 tags")
				bdd.AssertEqual(t, "env:prod", result[0], "primeira tag preservada")
				bdd.AssertEqual(t, "team:platform", result[1], "segunda tag preservada")
			})
		})
	})
}

func TestMergeTagSlicesEmptyExisting(t *testing.T) {
	bdd.Feature(t, "Fazer merge de tags — existing vazio", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("nenhuma tag existente, incoming tem tags", func(s *bdd.Scenario) {
			var incoming, result []string
			s.Given("incoming com tags", func() {
				incoming = []string{"env:prod", "team:platform"}
			})
			s.When("faço merge com existing vazio", func() {
				result = utils.MergeTagSlices([]string{}, incoming)
			})
			s.Then("todas as tags incoming são adicionadas", func(t *testing.T) {
				bdd.AssertEqual(t, 2, len(result), "deve ter 2 tags")
				bdd.AssertEqual(t, "env:prod", result[0], "primeira tag incoming")
				bdd.AssertEqual(t, "team:platform", result[1], "segunda tag incoming")
			})
		})
	})
}

// ---------------------------------------------------------------------------
// applyHostTagsOperator (via unstructured.Unstructured direto)
// ---------------------------------------------------------------------------

func TestApplyHostTagsOperatorInjectsTagsInEmptySpec(t *testing.T) {
	bdd.Feature(t, "Injetar host tags no DatadogAgent (operator mode)", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("spec.global.tags não existe ainda", func(s *bdd.Scenario) {
			var obj unstructured.Unstructured
			var hostTags []string
			s.Given("um DatadogAgent sem spec.global.tags", func() {
				obj = unstructured.Unstructured{Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"global": map[string]interface{}{},
					},
				}}
				hostTags = []string{"env:prod", "team:platform"}
			})
			s.When("injeto as host tags manualmente via SetNestedSlice", func() {
				out := make([]interface{}, len(hostTags))
				for i, tag := range hostTags {
					out[i] = tag
				}
				_ = unstructured.SetNestedSlice(obj.Object, out, "spec", "global", "tags")
			})
			s.Then("spec.global.tags deve conter as tags injetadas", func(t *testing.T) {
				got, found, err := unstructured.NestedStringSlice(obj.Object, "spec", "global", "tags")
				bdd.AssertNoError(t, err, "não deve dar erro ao ler tags")
				bdd.AssertTrue(t, found, "spec.global.tags deve existir")
				bdd.AssertEqual(t, 2, len(got), "deve ter 2 tags")
				bdd.AssertEqual(t, "env:prod", got[0], "primeira tag")
				bdd.AssertEqual(t, "team:platform", got[1], "segunda tag")
			})
		})
	})
}

func TestApplyHostTagsOperatorMergesExistingTags(t *testing.T) {
	bdd.Feature(t, "Injetar host tags — merge com tags existentes no operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("spec.global.tags já tem tags, incoming é descartado quando chave já existe", func(s *bdd.Scenario) {
			var obj unstructured.Unstructured
			var hostTags, result []string
			s.Given("um DatadogAgent com spec.global.tags existente", func() {
				obj = unstructured.Unstructured{Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"global": map[string]interface{}{
							"tags": []interface{}{"env:prod", "region:us-east-1"},
						},
					},
				}}
				hostTags = []string{"env:staging", "team:platform"}
			})
			s.When("faço merge das tags (simulate applyHostTagsOperator)", func() {
				existing, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "global", "tags")
				merged := utils.MergeTagSlices(existing, hostTags)
				out := make([]interface{}, len(merged))
				for i, t := range merged {
					out[i] = t
				}
				_ = unstructured.SetNestedSlice(obj.Object, out, "spec", "global", "tags")
				result, _, _ = unstructured.NestedStringSlice(obj.Object, "spec", "global", "tags")
			})
			s.Then("env local é preservado, env:staging descartado, team:platform adicionado", func(t *testing.T) {
				bdd.AssertEqual(t, 3, len(result), "deve ter 3 tags no total")
				bdd.AssertEqual(t, "env:prod", result[0], "env local preservado")
				bdd.AssertEqual(t, "region:us-east-1", result[1], "region preservado")
				bdd.AssertEqual(t, "team:platform", result[2], "team adicionado (chave nova)")
			})
		})
	})
}

// ---------------------------------------------------------------------------
// applyHostTagsHelm (via mapa de values)
// ---------------------------------------------------------------------------

func TestApplyHostTagsHelmInjectsTagsInEmptyDatadog(t *testing.T) {
	bdd.Feature(t, "Injetar host tags no values do Helm (helm mode)", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("datadog.tags não existe", func(s *bdd.Scenario) {
			var values map[string]interface{}
			var hostTags []string
			s.Given("values sem datadog.tags", func() {
				values = map[string]interface{}{
					"datadog": map[string]interface{}{
						"clusterName": "my-cluster",
					},
				}
				hostTags = []string{"env:prod", "team:infra"}
			})
			s.When("injeto as tags (simulate applyHostTagsHelm)", func() {
				datadogMap := values["datadog"].(map[string]interface{})
				existing := []string{}
				merged := utils.MergeTagSlices(existing, hostTags)
				out := make([]interface{}, len(merged))
				for i, tag := range merged {
					out[i] = tag
				}
				datadogMap["tags"] = out
				values["datadog"] = datadogMap
			})
			s.Then("datadog.tags deve conter as tags injetadas", func(t *testing.T) {
				datadogMap := values["datadog"].(map[string]interface{})
				tags := datadogMap["tags"].([]interface{})
				bdd.AssertEqual(t, 2, len(tags), "deve ter 2 tags")
				bdd.AssertEqual(t, "env:prod", tags[0].(string), "primeira tag")
				bdd.AssertEqual(t, "team:infra", tags[1].(string), "segunda tag")
			})
		})
	})
}

func TestApplyHostTagsHelmMergesExistingTags(t *testing.T) {
	bdd.Feature(t, "Injetar host tags — merge com tags existentes no helm values", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("datadog.tags já tem tags, incoming é descartado quando chave já existe", func(s *bdd.Scenario) {
			var values map[string]interface{}
			var hostTags []string
			var result []interface{}
			s.Given("values com datadog.tags existente", func() {
				values = map[string]interface{}{
					"datadog": map[string]interface{}{
						"tags": []interface{}{"env:prod", "region:us-east-1"},
					},
				}
				hostTags = []string{"env:staging", "team:platform"}
			})
			s.When("faço merge (simulate applyHostTagsHelm)", func() {
				datadogMap := values["datadog"].(map[string]interface{})
				var existing []string
				for _, item := range datadogMap["tags"].([]interface{}) {
					existing = append(existing, item.(string))
				}
				merged := utils.MergeTagSlices(existing, hostTags)
				out := make([]interface{}, len(merged))
				for i, tag := range merged {
					out[i] = tag
				}
				datadogMap["tags"] = out
				values["datadog"] = datadogMap
				result = values["datadog"].(map[string]interface{})["tags"].([]interface{})
			})
			s.Then("env local é preservado, env:staging descartado, team:platform adicionado", func(t *testing.T) {
				bdd.AssertEqual(t, 3, len(result), "deve ter 3 tags no total")
				bdd.AssertEqual(t, "env:prod", result[0].(string), "env local preservado")
				bdd.AssertEqual(t, "region:us-east-1", result[1].(string), "region preservado")
				bdd.AssertEqual(t, "team:platform", result[2].(string), "team adicionado (chave nova)")
			})
		})
	})
}
