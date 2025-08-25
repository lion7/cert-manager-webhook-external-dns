{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "external-dns-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "external-dns-webhook.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "external-dns-webhook.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "external-dns-webhook.labels" -}}
helm.sh/chart: {{ include "external-dns-webhook.chart" . }}
{{ include "external-dns-webhook.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "external-dns-webhook.selectorLabels" -}}
app.kubernetes.io/name: {{ include "external-dns-webhook.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "external-dns-webhook.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "external-dns-webhook.fullname" .) }}
{{- end -}}

{{- define "external-dns-webhook.rootCAIssuer" -}}
{{ printf "%s-ca" (include "external-dns-webhook.fullname" .) }}
{{- end -}}

{{- define "external-dns-webhook.rootCACertificate" -}}
{{ printf "%s-ca" (include "external-dns-webhook.fullname" .) }}
{{- end -}}

{{- define "external-dns-webhook.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "external-dns-webhook.fullname" .) }}
{{- end -}}
