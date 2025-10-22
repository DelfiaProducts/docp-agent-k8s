package operators

import "github.com/OryaHub/agent-k8s/dto"

// ValidateAllDatadogDeploymentSuccess verifica se a implantação foi bem-sucedida
func (a *AgentOperator) ValidateAllDatadogDeploymentSuccess(mode string, datadogDto dto.DatadogDTO) (bool, error) {
	var namespace string
	if len(datadogDto.DatadogNamespace) > 0 {
		namespace = datadogDto.DatadogNamespace
	} else {
		namespace = datadogDto.Namespace
	}

	return a.helmClient.ValidateAllDatadogDeploymentsSuccess(mode, namespace)
}

// ValidateAndRollbackDatadogIfNeeded valida o deployment datadog e executa rollback se necessário
func (a *AgentOperator) ValidateAndRollbackDatadogIfNeeded(mode, namespace, releaseName string, maxRetries int) error {
	return a.helmClient.ValidateAndRollbackDatadogIfNeeded(mode, namespace, releaseName, maxRetries)
}
