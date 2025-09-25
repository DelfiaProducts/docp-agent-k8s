package operators

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/OryaHub/agent-k8s/dto"
	"github.com/OryaHub/agent-k8s/utils"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// WebhookOperator is struct for webhook operator
type WebhookOperator struct {
	logger     *utils.K8sLogger
	helmClient *utils.HelmClient
	kubeClient *utils.KubeClient
}

// NewWebhookOperator return instance of webhook operator
func NewWebhookOperator(logger *utils.K8sLogger) *WebhookOperator {
	return &WebhookOperator{
		logger:     logger,
		kubeClient: utils.NewKubeClient(),
		helmClient: utils.NewHelmClient(logger),
	}
}

// Setup configure operator
func (w *WebhookOperator) Setup() error {
	if err := w.kubeClient.LoadConfigKube(); err != nil {
		return err
	}
	if err := w.helmClient.Setup(); err != nil {
		return err
	}

	return nil
}

// GetAdmissionReview return admission review from mutation
func (w *WebhookOperator) GetAdmissionReview(body []byte) (admissionv1.AdmissionReview, error) {
	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		w.logger.Error("Error unmarshalling admission review", "error", err)
		return admissionv1.AdmissionReview{}, err
	}
	return admissionReview, nil
}

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

// ParseAndReplaceLabelOrAnnotation execut parse the labels and annotations
func (w *WebhookOperator) ParseAndReplaceLabelOrAnnotation(template string, labels map[string]string, annotations map[string]string) string {
	re := regexp.MustCompile(`\$\[(labels|annotations)\.([^\]]+)\]`)
	if !re.MatchString(template) {
		if strings.Contains(template, "=") {
			keyValue := strings.Split(template, "=")
			if len(keyValue) == 2 {
				return fmt.Sprintf("%s=unknown", keyValue[0])
			}
		}
	}
	return re.ReplaceAllStringFunc(template, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) == 3 {
			w.logger.Debug("parsed label or annotation", "match", match, "submatches", submatches)
			prefix := submatches[1]
			key := submatches[2]
			if prefix == "labels" {
				if value, ok := labels[key]; ok {
					return value
				} else {
					return "unknown"
				}
			} else if prefix == "annotations" {
				if value, ok := annotations[key]; ok {
					return value
				} else {
					return "unknown"
				}
			}
		}
		return template
	})
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

// PopulateLabelsAndAnnotations populate labels and annotations
func (w *WebhookOperator) PopulateLabelsAndAnnotations(admissionReview admissionv1.AdmissionReview, labels dto.K8sConfigSignalLabels, annotations dto.K8sConfigSignalAnnotations) (lb map[string]string, an map[string]string, modified bool, err error) {
	request := admissionReview.Request
	namespace := request.Namespace
	kind := request.Kind.Kind
	w.logger.Debug("populate labels and annotations", "request", request, "namespace", namespace, "kind", kind)
	w.logger.Debug("populate labels ando annotations", "kind", kind)
	if kind == "Pod" {
		pod := &corev1.Pod{}
		if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
			w.logger.Error("Error unmarshalling pod", "error", err)
			return nil, nil, false, err
		}
		deploymentName, err := w.getDeploymentNameFromPod(pod)
		if err != nil {
			return nil, nil, false, err
		}
		w.logger.Debug("populate labels and annotations", "deploymentName", deploymentName)
		existedAnnotations := pod.ObjectMeta.Annotations
		if existedAnnotations == nil {
			existedAnnotations = make(map[string]string)
		}
		existedLabels := pod.ObjectMeta.Labels
		if existedLabels == nil {
			existedLabels = make(map[string]string)
		}
		lb = existedLabels
		an = existedAnnotations
		labelsDeploymentExists := false
		annotationsDeploymentExists := false
		w.logger.Debug("populate labels and annotations", "lb", existedLabels, "an", existedAnnotations)
		// populate when deployment labels
		for _, labelDeployment := range labels.Deployments {
			if deploymentName == labelDeployment.Name {
				modified = true
				labelsDeploymentExists = true
				lb = w.populateMapsLabelsOrAnnotations("labels", labelDeployment.Add, lb, an)
			}
		}
		// polulate when deployment annotations
		for _, annotationDeployment := range annotations.Deployments {
			if deploymentName == annotationDeployment.Name {
				modified = true
				annotationsDeploymentExists = true
				an = w.populateMapsLabelsOrAnnotations("annotations", annotationDeployment.Add, lb, an)
			}
		}
		if !labelsDeploymentExists {
			// populate when namespaces labels
			for _, labelNamespace := range labels.Namespaces {
				if namespace == labelNamespace.Name {
					modified = true
					lb = w.populateMapsLabelsOrAnnotations("labels", labelNamespace.Add, lb, an)
				}
			}
		}
		if !annotationsDeploymentExists {
			// polulate when namespaces annotations
			for _, annotationNamespace := range annotations.Namespaces {
				if namespace == annotationNamespace.Name {
					modified = true
					an = w.populateMapsLabelsOrAnnotations("annotations", annotationNamespace.Add, lb, an)
				}
			}
		}
	}
	return lb, an, modified, nil
}

// ApplyPatchForAdmissionReview apply patch for admission review
func (w *WebhookOperator) ApplyPatchForAdmissionReview(admissionReview admissionv1.AdmissionReview, labels map[string]string, annotations map[string]string, modified bool) (admissionv1.AdmissionReview, error) {
	var patch []map[string]interface{}
	patch = append(patch, map[string]interface{}{
		"op":    "add",
		"path":  "/metadata/annotations",
		"value": annotations,
	})
	patch = append(patch, map[string]interface{}{
		"op":    "add",
		"path":  "/metadata/labels",
		"value": labels,
	})

	request := admissionReview.Request
	response := admissionv1.AdmissionReview{
		TypeMeta: admissionReview.TypeMeta,
		Response: &admissionv1.AdmissionResponse{
			UID:     request.UID,
			Allowed: true,
		},
	}
	w.logger.Debug("apply patch for admission review", "patch", patch, "modified", modified)
	if modified {
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			w.logger.Error("Error marshalling patch", "error", err)
			return admissionv1.AdmissionReview{}, err
		}

		patchType := admissionv1.PatchTypeJSONPatch
		response.Response.PatchType = &patchType
		response.Response.Patch = patchBytes
		w.logger.Debug("Applied patch to the pod")
	}

	return response, nil
}

// GetConfigMap execute get the config map
func (w *WebhookOperator) GetConfigMap(configMapName, namespace string) (*corev1.ConfigMap, error) {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(w.kubeClient.Config)
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
func (w *WebhookOperator) UpdateConfigMap(configMapName string, namespace string, data map[string]string) error {
	ctx := context.Background()
	clientset, err := kubernetes.NewForConfig(w.kubeClient.Config)
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
