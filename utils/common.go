package utils

import (
	"fmt"
	"os"

	"github.com/OryaHub/agent-k8s/internal"
)

// GetDomainUrl return domain url
func GetDomainUrl() (string, error) {
	oryaDomain := os.Getenv("ORYA_DOMAIN")
	if len(oryaDomain) != 0 {
		return oryaDomain, nil
	}
	return internal.ORYA_DOMAIN, nil
}

// GetDatadogHelmRepository returns the Datadog Helm repository URL
func GetDatadogHelmRepository() string {
	return internal.DATADOG_HELM_REPOSITORY
}

// GetDatadogDeploymentOperatorName returns the Datadog deployment operator name
func GetDatadogDeploymentOperatorName() string {
	return internal.DATADOG_DEPLOYMENT_OPERATOR_NAME
}

// GetDatadogDeploymentClusterAgentName returns the Datadog deployment cluster agent name
func GetDatadogDeploymentClusterAgentName() string {
	return internal.DATADOG_DEPLOYMENT_CLUSTER_AGENT_NAME
}

// GetDatadogDeploymentClusterAgentHelmName returns the Datadog deployment cluster agent Helm name
func GetDatadogDeploymentClusterAgentHelmName() string {
	return internal.DATADOG_DEPLOYMENT_CLUSTER_AGENT_HELM_NAME
}

// GetHelmRepository returns the Helm repository URL
func GetHelmRepository() string {
	return internal.ORYA_HELM_REPOSITORY
}

// GetHelmRepositoryName returns chart name the orya helm repository
func GetHelmRepositoryName() string {
	return internal.ORYA_HELM_REPOSITORY_NAME
}

// GetHelmChartName returns the chart name for the orya helm
func GetHelmChartName() string {
	return internal.ORYA_HELM_CHART_NAME
}

// GetOryaNamespace return orya namespace
func GetOryaNamespace() string {
	return internal.ORYA_NAMESPACE
}

// GetMutatingWebhookName return orya mutating webhook name
func GetMutatingWebhookName() string {
	return internal.ORYA_MUTATING_WEBHOOK_NAME
}

// GetClusterRoleName return orya cluster role name
func GetClusterRoleName() string {
	return internal.ORYA_CLUSTER_ROLE_NAME
}

// GetClusterRoleBindingName return orya cluster role binding name
func GetClusterRoleBindingName() string {
	return internal.ORYA_CLUSTER_ROLE_BINDING_NAME
}

// GetOryaConfiMapStateName return orya config map state name
func GetOryaConfiMapStateName() string {
	return internal.ORYA_CONFIG_MAP_STATE_NAME
}

// GetOryaConfigMapConfigurationsName return orya config map configurations name
func GetOryaConfigMapConfigurationsName() string {
	return internal.ORYA_CONFIG_MAP_CONFIGURATIONS_NAME
}

// GetServiceAccountName return service account name
func GetServiceAccountName() string {
	return internal.SERVICE_ACCOUNT_NAME
}

// GetOryaReleaseName return release name the orya
func GetOryaReleaseName() string {
	return os.Getenv("RELEASE_NAME")
}

// GetOryaUpdaterRepositoryName return orya updater repository name
func GetOryaUpdaterRepositoryName(version string) string {
	name := fmt.Sprintf("%s/%s:%s", internal.ORYA_REPOSITORY_IMAGE_NAME, internal.ORYA_DEPLOYMENT_UPDATER_NAME, version)
	return name
}

// GetRepositoryImage return name of repository image
func GetRepositoryImage(name string) string {
	switch name {
	case "manager":
		managerImage := os.Getenv("MANAGER_IMAGE_NAME")
		if len(managerImage) > 0 {
			return managerImage
		} else {
			return fmt.Sprintf("%s/%s", internal.ORYA_REPOSITORY_IMAGE_NAME, "k8s-manager")
		}
	case "agent":
		agentImage := os.Getenv("AGENT_IMAGE_NAME")
		if len(agentImage) > 0 {
			return agentImage
		} else {
			return fmt.Sprintf("%s/%s", internal.ORYA_REPOSITORY_IMAGE_NAME, "k8s-agent")
		}
	case "webhook":
		webhookImage := os.Getenv("WEBHOOK_IMAGE_NAME")
		if len(webhookImage) > 0 {
			return webhookImage
		} else {
			return fmt.Sprintf("%s/%s", internal.ORYA_REPOSITORY_IMAGE_NAME, "k8s-webhook")
		}
	}
	return ""
}

// GetOryaDeploymentName return name of deployment
func GetOryaDeploymentName(name string) string {
	switch name {
	case "manager":
		managerDeployNamme := os.Getenv("MANAGER_DEPLOYMENT_NAME")
		if len(managerDeployNamme) > 0 {
			return managerDeployNamme
		} else {
			return internal.ORYA_DEPLOYMENT_MANAGER_NAME
		}
	case "agent":
		agentDeployName := os.Getenv("AGENT_DEPLOYMENT_NAME")
		if len(agentDeployName) > 0 {
			return agentDeployName
		} else {
			return internal.ORYA_DEPLOYMENT_AGENT_NAME
		}
	case "webhook":
		webhookDeployName := os.Getenv("WEBHOOK_DEPLOYMENT_NAME")
		if len(webhookDeployName) > 0 {
			return webhookDeployName
		} else {
			return internal.ORYA_DEPLOYMENT_WEBHOOK_NAME
		}
	default:
		return ""
	}
}

// GetOryaContainerName return name of container pods
func GetOryaContainerName(name string) string {
	switch name {
	case "manager":
		return internal.CONTAINER_MANAGER_NAME
	case "agent":
		return internal.CONTAINER_AGENT_NAME
	case "webhook":
		return internal.CONTAINER_WEBHOOK_NAME
	default:
		return ""
	}
}

// ErrorOryaApiKeyNotFound return error the api key orya not found
func ErrorOryaApiKeyNotFound() error {
	return internal.OryaApiKeyNotFound
}

// ErrAuthTokenClaimsInvalid return error the invalid claims token
func ErrAuthTokenClaimsInvalid() error {
	return internal.ErrAuthTokenClaimsInvalid
}

// ErrNotFoundChart return error the chart not found
func ErrNotFoundChart() error {
	return internal.ErrNotFoundChart
}

// ErrNotFoundChartVersion return error the chart version not found
func ErrNotFoundChartVersion() error {
	return internal.ErrNotFoundChartVersion
}

// ErrNotFoundReleaseName return error the not found release name
func ErrNotFoundReleaseName() error {
	return internal.ErrNotFoundReleaseName
}

// ErrInvalidDatadogMode return error the invalid datadog mode
func ErrInvalidDatadogMode() error {
	return internal.ErrInvalidDatadogMode
}
