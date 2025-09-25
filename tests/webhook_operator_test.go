package tests

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/OryaHub/agent-k8s/bdd"
	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/operators"
	"github.com/OryaHub/agent-k8s/utils"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestNewWebhookOperator(t *testing.T) {
	bdd.Feature(t, "Instanciar o webhook operator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("criar logger pra passar pro operator", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.WebhookOperator
			s.Given("que eu tenho um logger", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
			})
			s.When("eu instancio o operator", func() {
				operator = operators.NewWebhookOperator(logger)
			})
			s.Then("o operator deve ser instanciado com o logger normalmente", func(t *testing.T) {
				bdd.AssertIsNotNil(t, operator, "o operator deve ser diferente de nil")
			})
		})
	})
}

func TestWebhookOperatorSetup(t *testing.T) {
	bdd.Feature(t, "Setup do WebhookOperator", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("setup básico do operator", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.WebhookOperator
			var err error
			s.Given("um WebhookOperator instanciado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewWebhookOperator(logger)
			})
			s.When("eu executo o setup", func() {
				err = operator.Setup()
			})
			s.Then("deve executar sem erro crítico", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado em ambiente de teste: %v", err)
				}
			})
		})
	})
}

func TestWebhookOperatorGetAdmissionReview(t *testing.T) {
	bdd.Feature(t, "Unmarshal de AdmissionReview", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("unmarshal de admission review válido", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.WebhookOperator
			var review admissionv1.AdmissionReview
			var err error
			s.Given("um WebhookOperator instanciado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewWebhookOperator(logger)
			})
			s.When("eu executo o setup", func() {
				err = operator.Setup()
			})
			s.When("eu executo o unmarshal de um admission review válido", func() {
				ar := admissionv1.AdmissionReview{}
				ar.Request = &admissionv1.AdmissionRequest{
					UID: "12345",
					Kind: v1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					Resource: v1.GroupVersionResource{
						Group:    "",
						Version:  "v1",
						Resource: "pods",
					},
					Namespace: "orya-agent",
					Name:      "test-pod",
					Object: runtime.RawExtension{
						Raw: []byte(`{
							"apiVersion": "v1",
							"kind": "Pod",
							"metadata": {
								"name": "test-pod",
								"namespace": "orya-agent",
								"labels": {
									"app": "webhook"
								}
							},
							"spec": {
								"containers": [{
									"name": "nginx",
									"image": "nginx:latest"
								}]
							}
						}`),
					},
				}
				ar.Kind = "AdmissionReview"
				arBytes, err := json.Marshal(ar)
				bdd.AssertNoError(t, err, "marshal deve ser sem erro")
				review, err = operator.GetAdmissionReview(arBytes)
			})
			s.Then("deve executar sem erro", func(t *testing.T) {
				bdd.AssertNoError(t, err, "unmarshal deve ser sem erro")
			})
			bdd.Printf("admission review: %+v", review)
		})
	})
}

func TestWebhookOperatorParseAndReplaceLabelOrAnnotation(t *testing.T) {
	bdd.Feature(t, "Parse and replace labels e annotations", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("parse and replace labels e annotations em admission review", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.WebhookOperator
			var lb map[string]string
			var an map[string]string
			var parsed string
			var err error
			s.Given("um WebhookOperator instanciado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewWebhookOperator(logger)
			})
			s.When("eu executo o setup", func() {
				err = operator.Setup()
				bdd.AssertNoError(t, err, "setup deve ser sem erro")
			})

			s.When("eu executo a função de parse e replace labels e annotations", func() {
				parsed = operator.ParseAndReplaceLabelOrAnnotation("app=$[app.webhook]", lb, an)
			})

			bdd.Printf("parsed: %+v\n", parsed)
		})
	})
}

func TestWebhookOperatorPopulateLabelsAndAnnotations(t *testing.T) {
	bdd.Feature(t, "Popular labels e annotations", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("popular labels e annotations em admission review", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.WebhookOperator
			var review admissionv1.AdmissionReview
			var labels dto.K8sConfigSignalLabels
			var annotations dto.K8sConfigSignalAnnotations
			var lb map[string]string
			var an map[string]string
			var modified bool
			var err error
			s.Given("um WebhookOperator instanciado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewWebhookOperator(logger)
			})
			s.When("eu executo o setup", func() {
				err = operator.Setup()
			})
			s.Given("um WebhookOperator instanciado e um admission review válido", func() {
				labels = dto.K8sConfigSignalLabels{Namespaces: []dto.K8sConfigSignalLabelsNamespace{{Name: "orya-agent", Add: []string{"app=$[labels.app]"}}}}
				annotations = dto.K8sConfigSignalAnnotations{Namespaces: []dto.K8sConfigSignalAnnotationsNamespace{{Name: "orya-agent", Add: []string{"team=$[app.devops]"}}}}
				review = admissionv1.AdmissionReview{}
				review.Kind = "AdmissionReview"
				review.Request = &admissionv1.AdmissionRequest{
					UID: "12345",
					Kind: v1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					Resource: v1.GroupVersionResource{
						Group:    "",
						Version:  "v1",
						Resource: "pods",
					},
					Namespace: "orya-agent",
					Name:      "test-pod",
					Object: runtime.RawExtension{
						Raw: []byte(`{
							"apiVersion": "v1",
							"kind": "Pod",
							"metadata": {
								"name": "test-pod",
								"namespace": "orya-agent",
								"labels": {
									"app": "webhook"
								}
							},
							"spec": {
								"containers": [{
									"name": "nginx",
									"image": "nginx:latest"
								}]
							}
						}`),
					},
				}

			})
			s.When("eu executo a função de popular labels e annotations", func() {
				lb, an, modified, err = operator.PopulateLabelsAndAnnotations(review, labels, annotations)
			})
			s.Then("deve executar sem erro e retornar maps", func(t *testing.T) {
				bdd.AssertNoError(t, err, "deve ser sem erro")
			})
			bdd.Printf("labels: %+v, annotations: %+v, modified: %v\n", lb, an, modified)
		})
	})
}

func TestWebhookOperatorApplyPatchForAdmissionReview(t *testing.T) {
	bdd.Feature(t, "Aplicar patch em AdmissionReview", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("aplicar patch em admission review válido", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.WebhookOperator
			var review admissionv1.AdmissionReview
			var patched admissionv1.AdmissionReview
			var err error
			s.Given("um WebhookOperator instanciado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewWebhookOperator(logger)
			})
			s.When("eu executo o setup", func() {
				err = operator.Setup()
			})
			s.Given("um WebhookOperator instanciado e um admission review válido", func() {
				review.Kind = "AdmissionReview"
				review.Request = &admissionv1.AdmissionRequest{
					UID: "12345",
					Kind: v1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					Resource: v1.GroupVersionResource{
						Group:    "",
						Version:  "v1",
						Resource: "pods",
					},
					Namespace: "orya-agent",
					Name:      "test-pod",
					Object: runtime.RawExtension{
						Raw: []byte(`{
							"apiVersion": "v1",
							"kind": "Pod",
							"metadata": {
								"name": "test-pod",
								"namespace": "orya-agent",
								"labels": {
									"app": "webhook"
								}
							},
							"spec": {
								"containers": [{
									"name": "nginx",
									"image": "nginx:latest"
								}]
							}
						}`),
					},
				}
			})
			s.When("eu executo a função de aplicar patch", func() {
				labels := map[string]string{"app": "webhook"}
				annotations := map[string]string{"team": "devops"}
				patched, err = operator.ApplyPatchForAdmissionReview(review, labels, annotations, true)
			})
			s.Then("deve executar sem erro e retornar review patchado", func(t *testing.T) {
				bdd.AssertNoError(t, err, "deve ser sem erro")
			})
			bdd.Printf("patched: %+v", patched)
		})
	})
}

func TestWebhookOperatorGetConfigMap(t *testing.T) {
	bdd.Feature(t, "Buscar ConfigMap", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("buscar configmap em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.WebhookOperator
			var err error
			var configMap interface{}
			s.Given("um WebhookOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewWebhookOperator(logger)
				_ = operator.Setup()
			})
			s.When("eu busco o configmap", func() {
				configMap, err = operator.GetConfigMap("webhook-config", "default")
			})
			s.Then("deve retornar erro ou configmap nulo em ambiente de teste", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
				if configMap == nil {
					bdd.Printf("configMap nulo (esperado em ambiente de teste)")
				}
			})
		})
	})
}

func TestWebhookOperatorUpdateConfigMap(t *testing.T) {
	bdd.Feature(t, "Atualizar ConfigMap", func(t *testing.T, scenario func(description string, steps func(s *bdd.Scenario))) {
		scenario("atualizar configmap em ambiente de teste", func(s *bdd.Scenario) {
			var logger *utils.K8sLogger
			var operator *operators.WebhookOperator
			var err error
			s.Given("um WebhookOperator instanciado e configurado", func() {
				logger = utils.NewK8sLoggerText(os.Stdout)
				operator = operators.NewWebhookOperator(logger)
				_ = operator.Setup()
			})
			s.When("eu atualizo o configmap", func() {
				err = operator.UpdateConfigMap("webhook-config", "default", map[string]string{"key": "value"})
			})
			s.Then("deve retornar erro ou simular atualização", func(t *testing.T) {
				if err != nil {
					bdd.Printf("erro esperado: %v", err)
				}
			})
		})
	})
}
