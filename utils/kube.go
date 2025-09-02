package utils

import (
	"context"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient is a client for interacting with Kubernetes
type KubeClient struct {
	Config *rest.Config
}

// NewKubeClient creates a new KubeClient
func NewKubeClient() *KubeClient {
	return &KubeClient{}
}

// LoadConfigKube loading config the kube
func (k *KubeClient) LoadConfigKube() error {
	mode := os.Getenv("MODE")
	if mode == "local" {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			kubeconfig = filepath.Join(homeDir, ".kube", "config")
		}

		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}
		k.Config = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		k.Config = config
	}
	return nil
}

// CreateConfigMap execute creation the config map
func (k *KubeClient) CreateConfigMap(configMapName string, namespace string, data map[string]string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			newConfiMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
				},
				Data: data,
			}
			_, errCreate := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &newConfiMap, metav1.CreateOptions{})
			if errCreate != nil {
				return err
			}
		}
	}
	return nil
}

// GetConfigMap execute get the config map
func (k *KubeClient) GetConfigMap(configMapName, namespace string) (*corev1.ConfigMap, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
	}
	return configMap, nil
}

// UpdateConfigMap execute update the config map
func (k *KubeClient) UpdateConfigMap(configMapName string, namespace string, data map[string]string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			newConfiMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
				},
				Data: data,
			}
			_, errCreate := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &newConfiMap, metav1.CreateOptions{})
			if errCreate != nil {
				return err
			}
		}
	}
	configMap.Data = data
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteConfigMap execute remove the config map
func (k *KubeClient) DeleteConfigMap(configMapName string, namespace string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return err
	}
	if err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}
