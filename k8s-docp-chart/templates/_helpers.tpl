{{/*
Expand the name of the chart.
*/}}
{{- define "k8s-docp.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "k8s-docp.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart-specific labels.
*/}}
{{- define "k8s-docp.labels" -}}
helm.sh/chart: {{ include "k8s-docp.name" . }}-{{ .Chart.Version | replace "+" "_" }}
{{ include "k8s-docp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Create the selector labels for the deployment.
*/}}
{{- define "k8s-docp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8s-docp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
