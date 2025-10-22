package operators

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getControllerOwnerDeploymentName get deployment name from owner ref
func (w *WebhookOperator) getControllerOwnerDeploymentName(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string, kind string) (string, error) {
	var obj metav1.Object
	var err error

	switch kind {
	case "ReplicaSet":
		obj, err = clientset.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "StatefulSet":
		obj, err = clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "DaemonSet":
		obj, err = clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "Job":
		obj, err = clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})

	}

	if err != nil {
		return "", err
	}

	if obj != nil {
		for _, ownerRef := range obj.GetOwnerReferences() {
			if ownerRef.Kind == "Deployment" && ownerRef.Controller != nil && *ownerRef.Controller {
				return ownerRef.Name, nil
			}
		}
	}
	return "", nil
}

// getDeploymentNameFromPod return deployment name the pod
func (w *WebhookOperator) getDeploymentNameFromPod(pod *corev1.Pod) (string, error) {
	ctx := context.Background()

	clientset, err := kubernetes.NewForConfig(w.kubeClient.Config)
	if err != nil {
		w.logger.Error("erro create clientset for get ReplicaSet", "error", err)
		return "", err
	}
	for _, ownerRef := range pod.ObjectMeta.OwnerReferences {
		if ownerRef.Controller != nil && *ownerRef.Controller {
			w.logger.Debug("get replica set from owner references the pod", "namespace", pod.Namespace, "rer name", ownerRef.Name)
			deploymentName, err := w.getControllerOwnerDeploymentName(ctx, clientset, pod.Namespace, ownerRef.Name, ownerRef.Kind)
			if err != nil {
				return "", err
			}
			if deploymentName != "" {
				return deploymentName, nil
			}
		}
	}
	return "", nil
}

// populateMapsLabelsOrAnnotations return labels or annotations populated
func (w *WebhookOperator) populateMapsLabelsOrAnnotations(typeOperation string, labelsOrAnnotations []string, lb map[string]string, an map[string]string) map[string]string {
	var newLabelsOrAnnotations map[string]string
	switch typeOperation {
	case "labels":
		newLabelsOrAnnotations = lb
	case "annotations":
		newLabelsOrAnnotations = an
	}
	for _, labelOrAnnotationAdd := range labelsOrAnnotations {
		if strings.Contains(labelOrAnnotationAdd, "=") {
			parsedLabelAdd := w.ParseAndReplaceLabelOrAnnotation(labelOrAnnotationAdd, lb, an)
			w.logger.Debug("parsed label or annotation", "original", labelOrAnnotationAdd, "parsed", parsedLabelAdd)
			keyValLabel := strings.Split(parsedLabelAdd, "=")
			if len(keyValLabel) == 2 {
				newLabelsOrAnnotations[keyValLabel[0]] = keyValLabel[1]
			}
		}
	}
	return newLabelsOrAnnotations
}
