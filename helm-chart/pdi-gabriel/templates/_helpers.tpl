{{- define "api-observabilidade.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "api-observabilidade.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "api-observabilidade.name" . -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "api-observabilidade.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" -}}
{{- end -}}

{{- define "api-observabilidade.labels" -}}
helm.sh/chart: {{ include "api-observabilidade.chart" . }}
app.kubernetes.io/name: {{ include "api-observabilidade.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "api-observabilidade.selectorLabels" -}}
app.kubernetes.io/name: {{ include "api-observabilidade.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "api-observabilidade.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
  {{- if .Values.serviceAccount.name -}}
{{ .Values.serviceAccount.name }}
  {{- else -}}
{{ include "api-observabilidade.fullname" . }}
  {{- end -}}
{{- else -}}
default
{{- end -}}
{{- end -}}

# Usado pelo ServiceAccount de todos os componentes.