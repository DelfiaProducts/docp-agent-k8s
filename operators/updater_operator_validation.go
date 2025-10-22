package operators

// ValidateDeploymentSuccess valida se o último deployment teve sucesso
func (uo *UpdaterOperator) ValidateDeploymentSuccess(namespace, deploymentName string) (bool, error) {
	return uo.helmClient.ValidateDeploymentSuccess(namespace, deploymentName)
}

// ValidateAllOryaDeploymentsSuccess valida se todos os deployments do DOCP tiveram sucesso
func (uo *UpdaterOperator) ValidateAllOryaDeploymentsSuccess(namespace string) (bool, error) {
	return uo.helmClient.ValidateAllOryaDeploymentsSuccess(namespace)
}

// ValidateAndRollbackIfNeeded valida o deployment e executa rollback se necessário
func (m *UpdaterOperator) ValidateAndRollbackIfNeeded(namespace, releaseName string, maxRetries int) error {
	return m.helmClient.ValidateAndRollbackIfNeeded(namespace, releaseName, maxRetries)
}
