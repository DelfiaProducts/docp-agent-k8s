package agents

// populateVersion populate version the webhook
func (k *K8sWebhook) populateVersion() error {
	configMapConfiguration, err := k.operator.GetConfigMap(k.configMapConfigurationsName, k.namespace)
	if err != nil {
		return err
	}
	k.version = configMapConfiguration.Data["version"]
	return nil
}
