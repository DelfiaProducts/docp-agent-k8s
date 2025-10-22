package operators

import (
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
)

// GetLatestHelmVersion busca a versão mais atual do chart Helm do DOCP
func (uo *UpdaterOperator) GetLatestHelmVersion(namespace, releaseName string) (string, error) {
	latestVersion, err := uo.helmClient.GetLatestHelmVersion(namespace, releaseName)
	if err != nil {
		uo.logger.Error("failed to get latest helm version", "error", err.Error())
		return "", err
	}

	return latestVersion, nil
}

// GetReleaseByVersion busca a versão do release do DOCP
func (uo *UpdaterOperator) GetReleaseByVersion(namespace, releaseName, version string) (*release.Release, error) {
	latestRelease, err := uo.helmClient.GetReleaseByVersion(namespace, releaseName, version)
	if err != nil {
		uo.logger.Error("failed to get latest release", "error", err.Error())
		return nil, err
	}

	return latestRelease, nil
}

// GetConfigMap retrieves a config map by name and namespace
func (uo *UpdaterOperator) GetConfigMap(configMapName string, namespace string) (*corev1.ConfigMap, error) {
	return uo.kubeClient.GetConfigMap(configMapName, namespace)
}
