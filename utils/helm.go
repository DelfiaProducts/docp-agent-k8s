package utils

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type HelmClient struct {
	Settings       *cli.EnvSettings
	RepositoryName string
	logger         *K8sLogger
	kubeClient     *KubeClient
}

func NewHelmClient(logger *K8sLogger) *HelmClient {
	settings := cli.New()
	kubeClient := NewKubeClient()
	return &HelmClient{
		Settings:   settings,
		logger:     logger,
		kubeClient: kubeClient,
	}
}

// Setup initializes the Helm client
func (hc *HelmClient) Setup() error {
	//configure kube client
	if err := hc.kubeClient.LoadConfigKube(); err != nil {
		return err
	}
	repoName, err := hc.GetHelmRepositoryName()
	if err != nil {
		return err
	}
	hc.RepositoryName = repoName
	return nil
}

// UpdateChartRepository execute update chart repository
func (hc *HelmClient) UpdateChartRepository(repositoryName, repositoryURL string) error {
	hc.logger.Debug("updating helm chart repository", "repositoryName", repositoryName, "repositoryURL", repositoryURL)

	helmDriver := os.Getenv("HELM_DRIVER")
	hc.logger.Debug("helm driver", "helmDriver", helmDriver)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), hc.Settings.Namespace(), helmDriver, hc.logger.Debug); err != nil {
		hc.logger.Error("failed to initialize helm action config for repo update", "error", err.Error())
		return err
	}

	repoFile := hc.Settings.RepositoryConfig

	repoEntry := &repo.Entry{
		Name: repositoryName,
		URL:  repositoryURL,
	}
	repoFileObj, err := repo.LoadFile(repoFile)
	if err != nil && os.IsNotExist(err) {
		hc.logger.Error("failed to load helm repo file", "error", err.Error())
	}
	if repoFileObj.Has(repositoryName) {
		// Update the repo URL if needed
		for i, r := range repoFileObj.Repositories {
			if r.Name == repositoryName {
				repoFileObj.Repositories[i].URL = repositoryURL
			}
		}
	} else {
		repoFileObj.Add(repoEntry)
	}

	// Update the specific repo
	chartRepo, err := repo.NewChartRepository(repoEntry, getter.All(hc.Settings))
	if err != nil {
		hc.logger.Error("failed to create chart repository", "error", err.Error())
		return err
	}
	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		hc.logger.Error("failed to update helm repository index", "error", err.Error())
		return err
	}

	if err := repoFileObj.WriteFile(repoFile, 0644); err != nil {
		hc.logger.Error("failed to write helm repo file", "error", err.Error())
		return err
	}

	hc.logger.Debug("helm chart repository updated successfully", "repositoryName", repositoryName)
	return nil
}

// GetLatestHelmVersion busca a versão mais atual do chart Helm instalado no cluster
func (hc *HelmClient) GetLatestHelmVersion(namespace, releaseName string) (string, error) {
	hc.logger.Debug("getting latest helm version from cluster", "namespace", namespace, "releaseName", releaseName)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), hc.logger.Debug); err != nil {
		hc.logger.Error("failed to initialize helm action config for getting latest version", "error", err.Error())
		return "", err
	}

	historyAction := action.NewHistory(actionConfig)
	historyAction.Max = 1
	releases, err := historyAction.Run(releaseName)
	if err != nil {
		hc.logger.Error("failed to get release history", "error", err.Error())
		return "", err
	}
	hc.logger.Debug("helm release history retrieved successfully", "releases", releases)

	if len(releases) == 0 {
		hc.logger.Error("release not found in cluster", "releaseName", releaseName)
		return "", fmt.Errorf("release %s not found in namespace %s", releaseName, namespace)
	}

	latestVersion := releases[0].Chart.Metadata.Version
	hc.logger.Debug("latest helm version found in cluster", "version", latestVersion, "releaseName", releaseName)

	return latestVersion, nil
}

// GetLatestHelmChartVersion busca a ultima versão do chart Helm instalado no cluster
func (hc *HelmClient) GetLatestHelmChartVersion(repoName, chartName string) (string, error) {
	hc.logger.Debug("getting helm chart version by version", "chartName", chartName, "repositoryName", repoName)
	var helmVersion string
	helmDriver := os.Getenv("HELM_DRIVER")
	hc.logger.Debug("helm driver", "helmDriver", helmDriver)

	repoFile := hc.Settings.RepositoryConfig
	repoConfig, err := repo.LoadFile(repoFile)
	if err != nil {
		hc.logger.Error("failed to load helm repo file", "error", err.Error())
		return helmVersion, err
	}

	// 2. Encontrar a entrada do repositório desejado.
	var repoEntry *repo.Entry
	for _, entry := range repoConfig.Repositories {
		if entry.Name == repoName {
			repoEntry = entry
			break
		}
	}

	if repoEntry == nil {
		hc.logger.Error("repository not found", "repositoryName", repoName)
		return helmVersion, ErrNotFoundChartVersion()
	}

	repoIndexFile := fmt.Sprintf("%s/%s-index.yaml", hc.Settings.RepositoryCache, repoName)
	index, err := repo.LoadIndexFile(repoIndexFile)
	if err != nil {
		hc.logger.Error("failed to load helm repo index file", "error", err.Error())
		return helmVersion, err
	}

	chartVersion, err := index.Get(chartName, "")
	if err != nil {
		hc.logger.Error("failed to search chart in helm repo index", "error", err.Error())
		return helmVersion, err
	}

	if chartVersion == nil {
		hc.logger.Error("chart not found in index", "chartName", chartName)
		return helmVersion, ErrNotFoundChartVersion()
	}
	return chartVersion.Version, nil
}

// GetReleaseName busca o nome da release no namespace especificado
func (hc *HelmClient) GetReleaseName(namespace, chartName string) (string, error) {
	var releaseName string
	helmDriver := os.Getenv("HELM_DRIVER")

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), namespace, helmDriver, hc.logger.Debug); err != nil {
		hc.logger.Error("falha ao inicializar a configuração do Helm", "error", err)
		return releaseName, err
	}

	listAction := action.NewList(actionConfig)
	listAction.Deployed = true
	listAction.All = true
	listAction.AllNamespaces = true

	releases, err := listAction.Run()
	if err != nil {
		hc.logger.Error("falha ao listar as releases do Helm", "error", err)
		return releaseName, err
	}

	for _, rel := range releases {
		hc.logger.Debug("verificando release", "release", rel.Chart.Metadata.Name, "namespace", namespace)
		if rel.Chart != nil && rel.Chart.Metadata.Name == chartName {
			return rel.Name, nil
		}
	}
	hc.logger.Debug("release encontrada", "chartName", chartName, "namespace", namespace)
	return releaseName, ErrNotFoundReleaseName()
}

// GetReleaseModeDatadog busca o mode da release no namespace especificado
func (hc *HelmClient) GetReleaseModeDatadog(namespace string) (string, error) {
	var mode string
	helmDriver := os.Getenv("HELM_DRIVER")

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), namespace, helmDriver, hc.logger.Debug); err != nil {
		hc.logger.Error("falha ao inicializar a configuração do Helm", "error", err)
		return mode, err
	}

	listAction := action.NewList(actionConfig)
	listAction.Deployed = true
	listAction.All = true
	listAction.AllNamespaces = true

	releases, err := listAction.Run()
	if err != nil {
		hc.logger.Error("falha ao listar as releases do Helm", "error", err)
		return mode, err
	}

	for _, rel := range releases {
		hc.logger.Debug("verificando release", "release", rel.Chart.Metadata.Name, "namespace", namespace)
		if rel.Chart != nil && strings.Contains(rel.Chart.Metadata.Name, "datadog") {
			chartName := rel.Chart.Metadata.Name
			switch chartName {
			case "datadog":
				mode = "helm"
				return mode, nil
			case "datadog-operator":
				mode = "operator"
				return mode, nil
			}
		}
	}
	hc.logger.Debug("release encontrada", "mode", mode, "namespace", namespace)
	return mode, ErrNotFoundReleaseName()
}

// UpgradeDocpHelmChart executa o upgrade do chart Helm do DOCP
func (hc *HelmClient) UpgradeDocpHelmChart(namespace, releaseName, repositoryURL, targetVersion string, values map[string]interface{}) error {
	chartName := fmt.Sprintf("%s-%s.tgz", "k8s-docp", targetVersion)

	hc.logger.Debug("upgrading DOCP helm chart",
		"namespace", namespace,
		"releaseName", releaseName,
		"chartName", chartName,
		"targetVersion", targetVersion)

	// Configurar o cliente de ação do Helm
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), hc.logger.Debug); err != nil {
		hc.logger.Error("failed to initialize helm action config", "error", err.Error())
		return err
	}

	// Criar ação de upgrade
	upgradeAction := action.NewUpgrade(actionConfig)
	upgradeAction.Namespace = namespace
	upgradeAction.Wait = true
	upgradeAction.Timeout = 5 * time.Minute
	upgradeAction.ResetValues = false
	upgradeAction.ReuseValues = true
	upgradeAction.Atomic = true
	upgradeAction.DisableHooks = true

	// Se uma versão específica foi fornecida, usar ela
	if targetVersion != "" {
		upgradeAction.Version = targetVersion
	}
	// Localizar o chart
	chartPath, err := upgradeAction.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", repositoryURL, chartName), hc.Settings)
	if err != nil {
		hc.logger.Error("failed to locate chart", "error", err.Error())
		return err
	}

	hc.logger.Debug("chart located successfully", "chartPath", chartPath)

	// Carregar o chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		hc.logger.Error("failed to load chart", "error", err.Error())
		return err
	}

	// Verificar se o release existe
	historyAction := action.NewHistory(actionConfig)
	historyAction.Max = 1
	releases, err := historyAction.Run(releaseName)
	if err != nil {
		hc.logger.Error("failed to get release history", "error", err.Error())
		return err
	}

	if len(releases) == 0 {
		hc.logger.Error("release not found", "releaseName", releaseName)
		return fmt.Errorf("release %s not found in namespace %s", releaseName, namespace)
	}

	// Executar o upgrade
	hc.logger.Debug("executing helm upgrade", "releaseName", releaseName, "chartVersion", chart.Metadata.Version)
	release, err := upgradeAction.Run(releaseName, chart, values)
	if err != nil {
		hc.logger.Error("failed to upgrade helm chart", "error", err.Error())
		return err
	}

	hc.logger.Debug("helm upgrade completed successfully",
		"releaseName", release.Name,
		"version", release.Chart.Metadata.Version,
		"status", release.Info.Status)

	return nil
}

// UpgradeDocpToLatestVersion busca a versão mais atual e executa o upgrade do DOCP
func (hc *HelmClient) UpgradeDocpToLatestVersion(namespace, releaseName, repositoryURL string, values map[string]interface{}) error {
	hc.logger.Debug("upgrading DOCP to latest version", "namespace", namespace, "releaseName", releaseName)

	// Buscar a versão mais atual
	latestVersion, err := hc.GetLatestHelmVersion(namespace, releaseName)
	if err != nil {
		hc.logger.Error("failed to get latest helm version", "error", err.Error())
		return err
	}

	// Verificar se já está na versão mais atual
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), hc.logger.Debug); err != nil {
		hc.logger.Error("failed to initialize helm action config for version check", "error", err.Error())
		return err
	}

	getAction := action.NewGet(actionConfig)
	currentRelease, err := getAction.Run(releaseName)
	if err != nil {
		hc.logger.Error("failed to get current release", "error", err.Error())
		return err
	}

	currentVersion := currentRelease.Chart.Metadata.Version
	if currentVersion == latestVersion {
		hc.logger.Debug("DOCP is already at the latest version",
			"currentVersion", currentVersion,
			"latestVersion", latestVersion)
		return nil
	}

	hc.logger.Debug("upgrading from current version to latest",
		"currentVersion", currentVersion,
		"latestVersion", latestVersion)

	// Executar o upgrade para a versão mais atual
	return hc.UpgradeDocpHelmChart(namespace, releaseName, repositoryURL, latestVersion, values)
}

// RollbackDocpHelmChart executa o rollback do chart Helm do DOCP para a versão anterior
func (hc *HelmClient) RollbackDocpHelmChart(namespace, releaseName string) error {
	hc.logger.Debug("rolling back DOCP helm chart",
		"namespace", namespace,
		"releaseName", releaseName)

	// Configurar o cliente de ação do Helm
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), hc.logger.Debug); err != nil {
		hc.logger.Error("failed to initialize helm action config for rollback", "error", err.Error())
		return err
	}

	// Verificar se o release existe
	historyAction := action.NewHistory(actionConfig)
	historyAction.Max = 256 // Buscar mais histórico para encontrar revisões disponíveis
	releases, err := historyAction.Run(releaseName)
	if err != nil {
		hc.logger.Error("failed to get release history for rollback", "error", err.Error())
		return err
	}

	if len(releases) == 0 {
		hc.logger.Error("release not found for rollback", "releaseName", releaseName)
		return fmt.Errorf("release %s not found in namespace %s", releaseName, namespace)
	}

	if len(releases) < 2 {
		hc.logger.Error("insufficient release history for rollback", "releaseName", releaseName, "historyCount", len(releases))
		return fmt.Errorf("insufficient release history for rollback, only %d release(s) found", len(releases))
	}

	// Buscar a revisão anterior que não seja a atual
	length := len(releases)
	hc.logger.Debug("length releases", "length", length)
	currentRevision := length
	targetRevision := length - 1
	hc.logger.Debug("current revision determined", "currentRevision", currentRevision)

	// Validar se a revisão existe no histórico
	var targetExists bool
	for _, release := range releases {
		if release.Version == targetRevision {
			targetExists = true
			break
		}
	}

	if !targetExists {
		hc.logger.Error("target revision not found in release history",
			"targetRevision", targetRevision,
			"releaseName", releaseName)
		return fmt.Errorf("revision %d not found in release history for %s", targetRevision, releaseName)
	}

	// Criar ação de rollback
	rollbackAction := action.NewRollback(actionConfig)
	rollbackAction.Wait = true
	rollbackAction.Timeout = 5 * time.Minute
	rollbackAction.Version = targetRevision

	// Executar o rollback
	hc.logger.Debug("executing helm rollback",
		"releaseName", releaseName,
		"targetRevision", targetRevision,
		"currentRevision", currentRevision)

	err = rollbackAction.Run(releaseName)
	if err != nil {
		hc.logger.Error("failed to rollback helm chart", "error", err.Error())
		return err
	}

	// Verificar o status após rollback
	getAction := action.NewGet(actionConfig)
	rolledBackRelease, err := getAction.Run(releaseName)
	if err != nil {
		hc.logger.Error("failed to get release status after rollback", "error", err.Error())
		return err
	}

	hc.logger.Debug("helm rollback completed successfully",
		"releaseName", rolledBackRelease.Name,
		"newRevision", rolledBackRelease.Version,
		"status", rolledBackRelease.Info.Status)

	return nil
}

// GetCurrentRelease return current release from helm
func (hc *HelmClient) GetCurrentRelease(namespace, releaseName string) (*release.Release, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), hc.logger.Debug); err != nil {
		hc.logger.Error("failed to initialize helm action config for getting current release", "error", err.Error())
		return nil, err
	}

	getAction := action.NewGet(actionConfig)
	currentRelease, err := getAction.Run(releaseName)
	if err != nil {
		hc.logger.Error("failed to get current release", "error", err.Error())
		return nil, err
	}

	return currentRelease, nil
}

// GetChartValuesByVersion retorna os valores do chart para uma versão específica
func (hc *HelmClient) GetChartValuesByVersion(version string) (map[string]interface{}, error) {
	var values map[string]interface{}
	chartName := fmt.Sprintf("%s-%s.tgz", GetHelmChartName(), version)

	helmDriver := os.Getenv("HELM_DRIVER")
	hc.logger.Debug("helm driver", "helmDriver", helmDriver)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), hc.Settings.Namespace(), helmDriver, hc.logger.Debug); err != nil {
		hc.logger.Error("failed to initialize helm action config for repo update", "error", err.Error())
		return values, err
	}

	repoFile := hc.Settings.RepositoryConfig

	repoFileObj, err := repo.LoadFile(repoFile)
	if err != nil && os.IsNotExist(err) {
		hc.logger.Error("failed to load helm repo file", "error", err.Error())
	}
	entry := repoFileObj.Get(hc.RepositoryName)
	if entry != nil {
		chartPathOptions := &action.ChartPathOptions{}
		chartPath, err := chartPathOptions.LocateChart(fmt.Sprintf("%s/%s", entry.URL, chartName), hc.Settings)
		if err != nil {
			hc.logger.Error("failed to locate chart", "error", err.Error())
			return values, err
		}

		chart, err := loader.Load(chartPath)
		if err != nil {
			hc.logger.Error("failed to load chart", "error", err.Error())
			return values, err
		}

		if chart.Values != nil {
			values = chart.Values
		}
		return values, nil
	}

	return values, ErrNotFoundChart()
}

// GetReleaseByVersion retorna a release do Helm com a versão do chart, mesmo que não seja a release current
func (hc *HelmClient) GetReleaseByVersion(namespace, releaseName, version string) (*release.Release, error) {
	hc.logger.Debug("getting helm release by version",
		"namespace", namespace,
		"releaseName", releaseName,
		"version", version)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(hc.Settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), hc.logger.Debug); err != nil {
		hc.logger.Error("failed to initialize helm action config for getting latest release", "error", err.Error())
		return nil, err
	}

	historyAction := action.NewHistory(actionConfig)
	historyAction.Max = 256 // Busca até 256 releases no histórico
	releases, err := historyAction.Run(releaseName)
	if err != nil {
		hc.logger.Error("failed to get release history", "error", err.Error())
		return nil, err
	}
	hc.logger.Debug("release history retrieved", "releases", releases)

	if len(releases) == 0 {
		hc.logger.Error("release not found", "releaseName", releaseName)
		return nil, fmt.Errorf("release %s not found in namespace %s", releaseName, namespace)
	}

	// Encontrar a release com a versão do chart mais nova
	var latestRelease *release.Release
	for _, r := range releases {
		hc.logger.Debug("release version", "chart version", r.Chart.Metadata.Version, "version", version)
		if r != nil && r.Chart.Metadata.Version == version {
			hc.logger.Debug("chart versions", "chart version", r.Chart.Metadata.Version, "latest version", version)
			latestRelease = r
		}
	}

	hc.logger.Debug("release with newest chart version found",
		"releaseName", releaseName,
		"chartVersion", latestRelease.Chart.Metadata.Version,
		"releaseRevision", latestRelease.Version)

	return latestRelease, nil
}

// ValidateDeploymentSuccess valida se o último deployment teve sucesso
func (hc *HelmClient) ValidateDeploymentSuccess(namespace, deploymentName string) (bool, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(hc.kubeClient.Config)
	if err != nil {
		return false, err
	}

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			hc.logger.Debug("deployment not found", "namespace", namespace, "deploymentName", deploymentName)
			return false, nil
		}
		return false, err
	}

	// Verifica as condições do deployment
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing {
			if condition.Status == corev1.ConditionTrue && condition.Reason == "NewReplicaSetAvailable" {
				// Verifica se todas as réplicas estão prontas
				if deployment.Status.ReadyReplicas == deployment.Status.Replicas &&
					deployment.Status.UpdatedReplicas == deployment.Status.Replicas &&
					deployment.Status.UnavailableReplicas == 0 {
					hc.logger.Debug("deployment validation success",
						"namespace", namespace,
						"deploymentName", deploymentName,
						"readyReplicas", deployment.Status.ReadyReplicas,
						"totalReplicas", deployment.Status.Replicas)
					return true, nil
				}
			}
		}

		// Se há uma condição de falha, o deployment não teve sucesso
		if condition.Type == appsv1.DeploymentReplicaFailure {
			if condition.Status == corev1.ConditionTrue {
				hc.logger.Debug("deployment validation failed",
					"namespace", namespace,
					"deploymentName", deploymentName,
					"reason", condition.Reason,
					"message", condition.Message)
				return false, nil
			}
		}
	}

	// Se chegou até aqui, verifica o status geral
	isSuccessful := deployment.Status.ReadyReplicas == deployment.Status.Replicas &&
		deployment.Status.UpdatedReplicas == deployment.Status.Replicas &&
		deployment.Status.UnavailableReplicas == 0

	hc.logger.Debug("deployment validation result",
		"namespace", namespace,
		"deploymentName", deploymentName,
		"isSuccessful", isSuccessful,
		"readyReplicas", deployment.Status.ReadyReplicas,
		"totalReplicas", deployment.Status.Replicas,
		"unavailableReplicas", deployment.Status.UnavailableReplicas)

	return isSuccessful, nil
}

// ValidateAllDocpDeploymentsSuccess valida se todos os deployments do DOCP tiveram sucesso
func (hc *HelmClient) ValidateAllDocpDeploymentsSuccess(namespace string) (bool, error) {
	deployments := []string{
		GetDocpDeploymentName("manager"),
		GetDocpDeploymentName("agent"),
		GetDocpDeploymentName("webhook"),
	}

	for _, deploymentName := range deployments {
		success, err := hc.ValidateDeploymentSuccess(namespace, deploymentName)
		if err != nil {
			hc.logger.Error("error validating deployment",
				"deploymentName", deploymentName,
				"namespace", namespace,
				"error", err.Error())
			return false, err
		}

		if !success {
			hc.logger.Debug("deployment validation failed",
				"deploymentName", deploymentName,
				"namespace", namespace)
			return false, nil
		}
	}

	hc.logger.Debug("all DOCP deployments validation successful", "namespace", namespace)
	return true, nil
}

// ValidateAllDatadogDeploymentsSuccess valida se todos os deployments do Datadog tiveram sucesso
func (hc *HelmClient) ValidateAllDatadogDeploymentsSuccess(mode, namespace string) (bool, error) {
	var deployments []string
	switch mode {
	case "helm":
		deployments = append(deployments,
			GetDatadogDeploymentClusterAgentHelmName(),
		)
	case "operator":
		deployments = append(deployments,
			GetDatadogDeploymentOperatorName(),
			GetDatadogDeploymentClusterAgentName(),
		)
	}

	if len(deployments) == 0 {
		hc.logger.Debug("no deployments to validate", "namespace", namespace)
		return true, nil
	}

	for _, deploymentName := range deployments {
		success, err := hc.ValidateDeploymentSuccess(namespace, deploymentName)
		if err != nil {
			hc.logger.Error("error validating deployment",
				"deploymentName", deploymentName,
				"namespace", namespace,
				"error", err.Error())
			return false, err
		}

		if !success {
			hc.logger.Debug("deployment validation failed",
				"deploymentName", deploymentName,
				"namespace", namespace)
			return false, nil
		}
	}

	hc.logger.Debug("all Datadog Operator deployments validation successful", "namespace", namespace)
	return true, nil
}

// ValidateAndRollbackIfNeeded valida o deployment e executa rollback se necessário
func (hc *HelmClient) ValidateAndRollbackIfNeeded(namespace, releaseName string, maxRetries int) error {
	hc.logger.Debug("validating deployment and checking if rollback is needed",
		"namespace", namespace,
		"releaseName", releaseName)

	// Aguardar um tempo para o deployment se estabilizar
	time.Sleep(30 * time.Second)

	// Tentar validar múltiplas vezes antes de decidir fazer rollback
	for attempt := 1; attempt <= maxRetries; attempt++ {
		hc.logger.Debug("validation attempt", "attempt", attempt, "maxRetries", maxRetries)

		// Validar todos os deployments do DOCP
		success, err := hc.ValidateAllDocpDeploymentsSuccess(namespace)
		if err != nil {
			hc.logger.Error("error during deployment validation", "error", err.Error(), "attempt", attempt)
			if attempt == maxRetries {
				hc.logger.Error("validation failed after max retries, initiating rollback")
				return hc.RollbackDocpHelmChart(namespace, releaseName)
			}
			// Aguardar antes da próxima tentativa
			time.Sleep(time.Duration(attempt*30) * time.Second)
			continue
		}

		if success {
			hc.logger.Debug("deployment validation successful", "attempt", attempt)
			return nil
		}

		hc.logger.Debug("deployment validation failed", "attempt", attempt)
		if attempt == maxRetries {
			hc.logger.Error("deployment validation failed after max retries, initiating rollback")
			return hc.RollbackDocpHelmChart(namespace, releaseName)
		}

		// Aguardar progressivamente mais tempo entre tentativas
		waitTime := time.Duration(attempt*30) * time.Second
		hc.logger.Debug("waiting before next validation attempt", "waitTime", waitTime)
		time.Sleep(waitTime)
	}

	return nil
}

// ValidateAndRollbackDatadogIfNeeded valida o deployment datadog e executa rollback se necessário
func (hc *HelmClient) ValidateAndRollbackDatadogIfNeeded(mode, namespace, releaseName string, maxRetries int) error {
	hc.logger.Debug("validating deployment datadog and checking if rollback is needed",
		"namespace", namespace,
		"releaseName", releaseName)

	// Aguardar um tempo para o deployment se estabilizar
	time.Sleep(30 * time.Second)

	// Tentar validar múltiplas vezes antes de decidir fazer rollback
	for attempt := 1; attempt <= maxRetries; attempt++ {
		hc.logger.Debug("validation attempt", "attempt", attempt, "maxRetries", maxRetries)

		// Validar todos os deployments do DOCP
		success, err := hc.ValidateAllDatadogDeploymentsSuccess(mode, namespace)
		if err != nil {
			hc.logger.Error("error during deployment validation", "error", err.Error(), "attempt", attempt)
			if attempt == maxRetries {
				hc.logger.Error("validation failed after max retries, initiating rollback")
				return hc.RollbackDocpHelmChart(namespace, releaseName)
			}
			// Aguardar antes da próxima tentativa
			time.Sleep(time.Duration(attempt*30) * time.Second)
			continue
		}

		if success {
			hc.logger.Debug("deployment validation successful", "attempt", attempt)
			return nil
		}

		hc.logger.Debug("deployment validation failed", "attempt", attempt)
		if attempt == maxRetries {
			hc.logger.Error("deployment validation failed after max retries, initiating rollback")
			return hc.RollbackDocpHelmChart(namespace, releaseName)
		}

		// Aguardar progressivamente mais tempo entre tentativas
		waitTime := time.Duration(attempt*30) * time.Second
		hc.logger.Debug("waiting before next validation attempt", "waitTime", waitTime)
		time.Sleep(waitTime)
	}

	return nil
}

// GetHelmRepositoryName returns the name of the Helm repository
func (hc *HelmClient) GetHelmRepositoryName() (string, error) {
	pattern := GetHelmRepository()
	repoFile := hc.Settings.RepositoryConfig
	repoConfig, err := repo.LoadFile(repoFile)
	if err != nil {
		hc.logger.Error("failed to load helm repo file", "error", err.Error())
		return "", err
	}

	for _, entry := range repoConfig.Repositories {
		if strings.Contains(entry.URL, pattern) {
			hc.logger.Debug("repository URL matches pattern", "name", entry.Name, "url", entry.URL, "pattern", pattern)
			return entry.Name, nil
		}
	}

	hc.logger.Debug("no repository URL matches pattern", "pattern", pattern)
	return "", ErrNotFoundRepositoryName()
}
