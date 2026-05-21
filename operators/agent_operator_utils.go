package operators

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func (a *AgentOperator) applyDatadogOperatorYml(datadogDto dto.DatadogDTO) error {
	clientset, err := dynamic.NewForConfig(a.kubeClient.Config)
	if err != nil {
		return err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(
		strings.NewReader(datadogDto.Content), 4096)

	var datadogAgent unstructured.Unstructured
	if err := decoder.Decode(&datadogAgent); err != nil {
		return err
	}
	applyHostTagsOperator(&datadogAgent, datadogDto.HostTags)
	a.logger.Debug("datadog agent after apply host tags", "datadogAgent", datadogAgent)
	gvr := schema.GroupVersionResource{
		Group:    "datadoghq.com",
		Version:  "v2alpha1",
		Resource: "datadogagents", // Plural do `kind` em lowercase
	}
	_, err = clientset.Resource(gvr).Namespace(datadogDto.DatadogNamespace).Get(context.Background(), datadogAgent.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return err
		}
	}
	_, err = clientset.Resource(gvr).Namespace(datadogDto.DatadogNamespace).Apply(context.Background(), datadogAgent.GetName(), &datadogAgent, metav1.ApplyOptions{
		FieldManager: "orya-operator-datadog-install",
		Force:        true,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// updateHelmChartOperator execute update chart datadog with operator mode
func (a *AgentOperator) updateHelmChartOperator(datadogDto dto.DatadogDTO) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), datadogDto.DatadogNamespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}
	var version string
	if datadogDto.Version == "latest" {
		versionChart, err := a.helmClient.GetLatestHelmChartVersion("datadog", "datadog-operator")
		if err != nil {
			return err
		}
		version = versionChart
	} else {
		version = datadogDto.Version
	}
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = datadogDto.DatadogNamespace
	upgrade.Version = version
	upgrade.Timeout = 5 * time.Minute
	upgrade.ResetValues = false
	upgrade.ReuseValues = true
	upgrade.DisableHooks = true

	chartPath, err := upgrade.ChartPathOptions.LocateChart("datadog/datadog-operator", a.helmClient.Settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(datadogDto.Content), &values)
	if err != nil {
		return err
	}

	_, err = upgrade.Run("datadog-operator", chart, values)
	if err != nil {
		return err
	}
	return nil
}

// applyDatadogUpdateConfigOperator execute update config datadog with operator mode
func (a *AgentOperator) applyDatadogUpdateConfigOperator(datadogDto dto.DatadogDTO) error {
	clientset, err := dynamic.NewForConfig(a.kubeClient.Config)
	if err != nil {
		return err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(
		strings.NewReader(datadogDto.Content), 4096)

	var datadogAgent unstructured.Unstructured
	if err := decoder.Decode(&datadogAgent); err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "datadoghq.com",
		Version:  "v2alpha1",
		Resource: "datadogagents", // Plural do `kind` em lowercase
	}

	resource, err := clientset.Resource(gvr).Namespace(datadogDto.DatadogNamespace).Get(context.Background(), datadogAgent.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("not found\n")
			return nil
		}
	}

	// Busca as tags que já estão no CRD atual e as carrega no novo objeto,
	// garantindo que tags locais tenham prioridade sobre as incoming (host_tags).
	existingTags, _, _ := unstructured.NestedStringSlice(resource.Object, "spec", "global", "tags")
	if len(existingTags) > 0 {
		out := make([]interface{}, len(existingTags))
		for i, t := range existingTags {
			out[i] = t
		}
		_ = unstructured.SetNestedSlice(datadogAgent.Object, out, "spec", "global", "tags")
	}
	a.logger.Debug("existing tags", "tags", existingTags)
	a.logger.Debug("host tags", "tags", datadogDto.HostTags)
	a.logger.Debug("datadog agent before apply host tags", "datadogAgent", datadogAgent)

	applyHostTagsOperator(&datadogAgent, datadogDto.HostTags)
	a.logger.Debug("datadog agent after apply host tags", "datadogAgent", datadogAgent)

	datadogAgent.SetResourceVersion(resource.GetResourceVersion())
	_, errUpdate := clientset.Resource(gvr).Namespace(datadogDto.DatadogNamespace).Update(context.Background(), &datadogAgent, metav1.UpdateOptions{})
	if errUpdate != nil {
		return err
	}
	return nil
}

// applyDatadogUpdateConfigHelm execute update config datadog with helm mode
func (a *AgentOperator) applyDatadogUpdateConfigHelm(datadogDto dto.DatadogDTO) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), datadogDto.DatadogNamespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}
	var version string
	if datadogDto.Version == "latest" {
		versionChart, err := a.helmClient.GetLatestHelmChartVersion("datadog", "datadog")
		if err != nil {
			return err
		}
		version = versionChart
	} else {
		version = datadogDto.Version
	}
	namespace := ""
	if len(datadogDto.DatadogNamespace) > 0 {
		namespace = datadogDto.DatadogNamespace
	} else {
		namespace = datadogDto.Namespace
	}
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = namespace
	upgrade.Install = true
	upgrade.Atomic = true
	upgrade.Version = version
	upgrade.Timeout = 5 * time.Minute
	upgrade.ResetValues = false
	upgrade.ReuseValues = true
	upgrade.DisableHooks = true

	chartPath, err := upgrade.ChartPathOptions.LocateChart("datadog/datadog", a.helmClient.Settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(datadogDto.Content), &values)
	if err != nil {
		return err
	}

	// Busca as tags que já estão no release atual e as carrega no values novo,
	// garantindo que tags locais tenham prioridade sobre as incoming (host_tags).
	getValues := action.NewGetValues(actionConfig)
	getValues.AllValues = true
	existingReleaseValues, err := getValues.Run("datadog-agent")
	if err == nil {
		existingTags := extractTagsFromHelmValues(existingReleaseValues)
		if len(existingTags) > 0 {
			setTagsInHelmValues(values, existingTags)
		}
	}

	applyHostTagsHelm(values, datadogDto.HostTags)

	_, err = upgrade.Run("datadog-agent", chart, values)
	if err != nil {
		return err
	}
	return nil
}

// createOrUpdateSecret execute creation or update the secret
func (a *AgentOperator) createOrUpdateSecret(datadogDto dto.DatadogDTO, secretName string) error {
	clientset, err := kubernetes.NewForConfig(a.kubeClient.Config)
	if err != nil {
		a.logger.Error("erro ao criar clientset", "error", err.Error())
		return err
	}
	apiKey := datadogDto.ApiKey
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: datadogDto.DatadogNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"api-key": apiKey,
		},
	}

	_, err = clientset.CoreV1().Secrets(datadogDto.DatadogNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			_, err = clientset.CoreV1().Secrets(datadogDto.DatadogNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update existing secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return err
}

func (a *AgentOperator) installHelmChart(releaseName string, datadogDto dto.DatadogDTO) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), datadogDto.DatadogNamespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}
	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = datadogDto.DatadogNamespace
	install.CreateNamespace = false
	install.Timeout = 1 * time.Minute

	var installVersion string
	if len(datadogDto.Version) > 0 && strings.Contains(datadogDto.Version, "latest") {
		latestVersion, err := a.helmClient.GetLatestHelmChartVersion("datadog", "datadog")
		if err != nil {
			return err
		}
		installVersion = latestVersion
	} else {
		installVersion = datadogDto.Version
	}
	a.logger.Debug("install version", "installVersion", installVersion)

	install.Version = installVersion

	chartPath, err := install.ChartPathOptions.LocateChart("datadog/datadog", a.helmClient.Settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(datadogDto.Content), &values)
	if err != nil {
		return err
	}
	applyHostTagsHelm(values, datadogDto.HostTags)

	_, err = install.Run(chart, values)
	return err
}

// extractTagsFromHelmValues reads datadog.tags from a Helm values map,
// returning them as []string. Handles both []string and []interface{} formats.
func extractTagsFromHelmValues(values map[string]interface{}) []string {
	datadogRaw, ok := values["datadog"]
	if !ok {
		return nil
	}
	datadogMap, ok := datadogRaw.(map[string]interface{})
	if !ok {
		return nil
	}
	tagsRaw, ok := datadogMap["tags"]
	if !ok {
		return nil
	}
	var tags []string
	switch v := tagsRaw.(type) {
	case []string:
		tags = v
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				tags = append(tags, s)
			}
		}
	}
	return tags
}

// setTagsInHelmValues writes tags into values["datadog"]["tags"] as []interface{},
// creating the datadog map if it does not exist.
func setTagsInHelmValues(values map[string]interface{}, tags []string) {
	datadogRaw, ok := values["datadog"]
	if !ok {
		datadogRaw = map[string]interface{}{}
	}
	datadogMap, ok := datadogRaw.(map[string]interface{})
	if !ok {
		datadogMap = map[string]interface{}{}
	}
	out := make([]interface{}, len(tags))
	for i, t := range tags {
		out[i] = t
	}
	datadogMap["tags"] = out
	values["datadog"] = datadogMap
}

// applyHostTagsHelm injects hostTags into the helm values map under datadog.tags,
// merging with any existing tags using key-priority rules.
func applyHostTagsHelm(values map[string]interface{}, hostTags []string) {
	if len(hostTags) == 0 {
		return
	}
	datadogRaw, ok := values["datadog"]
	if !ok {
		datadogRaw = map[string]interface{}{}
	}
	datadogMap, ok := datadogRaw.(map[string]interface{})
	if !ok {
		datadogMap = map[string]interface{}{}
	}

	var existing []string
	if tagsRaw, ok := datadogMap["tags"]; ok {
		switch v := tagsRaw.(type) {
		case []string:
			existing = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					existing = append(existing, s)
				}
			}
		}
	}

	merged := utils.MergeTagSlices(existing, hostTags)

	// Store as []interface{} to stay compatible with yaml/helm unmarshal output.
	out := make([]interface{}, len(merged))
	for i, t := range merged {
		out[i] = t
	}
	datadogMap["tags"] = out
	values["datadog"] = datadogMap
}

// applyHostTagsOperator injects hostTags into a DatadogAgent unstructured object
// at spec.global.tags, merging with any existing tags using key-priority rules.
func applyHostTagsOperator(obj *unstructured.Unstructured, hostTags []string) {
	if len(hostTags) == 0 {
		return
	}
	existing, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "global", "tags")
	merged := utils.MergeTagSlices(existing, hostTags)
	// SetNestedSlice requires []interface{}.
	out := make([]interface{}, len(merged))
	for i, t := range merged {
		out[i] = t
	}
	_ = unstructured.SetNestedSlice(obj.Object, out, "spec", "global", "tags")
}

func (a *AgentOperator) installHelmChartOperatorDatadog(releaseName string, datadogDto dto.DatadogDTO) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(a.helmClient.Settings.RESTClientGetter(), datadogDto.DatadogNamespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}

	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = datadogDto.DatadogNamespace
	install.CreateNamespace = false
	install.Atomic = true
	install.Timeout = 2 * time.Minute

	if len(datadogDto.Version) > 0 && strings.Contains(datadogDto.Version, "latest") {
		versionChart, err := a.helmClient.GetLatestHelmChartVersion("datadog", "datadog-operator")
		if err != nil {
			return err
		}
		install.Version = versionChart
	} else {
		install.Version = datadogDto.Version
	}
	chartPath, err := install.ChartPathOptions.LocateChart("datadog/datadog-operator", a.helmClient.Settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(datadogDto.Content), &values)
	if err != nil {
		return err
	}

	_, err = install.Run(chart, values)
	return err
}
